# Cache Key Generation Strategy Design

## Overview
This document describes the strategy for generating unique cache keys from LLM request parameters to enable response caching.

## Requirements
1. **Uniqueness**: Identical requests must generate the same cache key
2. **Collision Resistance**: Different requests should have a very low probability of generating the same key
3. **Determinism**: The same input must always produce the same output
4. **Performance**: Key generation should be fast and have low overhead

## Proposed Strategy

### Hash Algorithm
Use **SHA256** for cache key generation, which is already used in the codebase for file hashing (`internal/cache/cache.go`).

**Rationale**:
- Proven cryptographic hash function with excellent collision resistance
- Already available in Go's standard library (`crypto/sha256`)
- Fast computation for cache key generation
- Generates 256-bit (64 hex characters) output

### Key Components to Hash

The cache key will be derived from the following `CompletionRequest` fields:

```go
type CompletionRequest struct {
    SystemPrompt string    // ✓ Include
    Messages     []Message // ✓ Include (all fields)
    Tools        []ToolDefinition // ✓ Include (all fields)
    MaxTokens    int       // ✗ Exclude (doesn't affect response content)
    Temperature  float64   // ✓ Include (affects randomness/creativity)
}
```

### Why These Fields?

**Include - SystemPrompt**:
- The system prompt defines the LLM's behavior and role
- Critical for determining response content

**Include - Messages**:
- Contains the conversation history
- Includes role, content, and tool_call_id
- The core input that determines the LLM's response

**Include - Tools**:
- Tool definitions affect what the LLM can call
- Different tool sets can lead to different responses

**Include - Temperature**:
- Directly affects response randomness and creativity
- Temperature 0.0 produces deterministic responses
- Temperature 0.7 produces more varied responses

**Exclude - MaxTokens**:
- Only limits response length, not response content
- Same prompt with different max_tokens will produce similar initial content
- Excluding allows cache hits even when token limits differ

## Implementation Approach

### Serialization Strategy

To ensure deterministic hashing, we need a canonical JSON serialization:

```go
// CacheKeyRequest represents the fields that affect cache key
type CacheKeyRequest struct {
    SystemPrompt string             `json:"system_prompt"`
    Messages     []CacheKeyMessage  `json:"messages"`
    Tools        []CacheKeyTool     `json:"tools"`
    Temperature  float64            `json:"temperature"`
}

type CacheKeyMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
    ToolID  string `json:"tool_id,omitempty"`
}

type CacheKeyTool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}
```

### Canonicalization Rules

To ensure consistent serialization:

1. **Empty Slices**: Encode empty slices as `[]` not `null`
2. **Map Keys**: Sort map keys alphabetically before encoding
3. **Float Precision**: Normalize temperature to consistent precision
4. **Whitespace**: Trim whitespace from strings before hashing
5. **Message Order**: Preserve message order (it's semantically significant)

### Algorithm

```
1. Create CacheKeyRequest from CompletionRequest
2. Marshal to JSON with canonicalization:
   - Sort all map keys
   - Handle empty slices correctly
   - Use stable JSON encoding
3. Compute SHA256 hash of JSON bytes
4. Encode hash as hexadecimal string (64 chars)
5. Return hex string as cache key
```

### Example

```go
func GenerateCacheKey(req llm.CompletionRequest) (string, error) {
    // 1. Extract cache-relevant fields
    keyReq := CacheKeyRequest{
        SystemPrompt: strings.TrimSpace(req.SystemPrompt),
        Temperature:  req.Temperature,
    }

    // 2. Convert messages (preserving order)
    for _, msg := range req.Messages {
        keyReq.Messages = append(keyReq.Messages, CacheKeyMessage{
            Role:    msg.Role,
            Content: strings.TrimSpace(msg.Content),
            ToolID:  msg.ToolID,
        })
    }

    // 3. Convert tools with sorted parameters
    for _, tool := range req.Tools {
        keyReq.Tools = append(keyReq.Tools, CacheKeyTool{
            Name:        tool.Name,
            Description: tool.Description,
            Parameters:  tool.Parameters,
        })
    }

    // 4. Marshal to canonical JSON
    data, err := json.Marshal(keyReq)
    if err != nil {
        return "", fmt.Errorf("failed to marshal cache key request: %w", err)
    }

    // 5. Compute SHA256 hash
    hash := sha256.Sum256(data)

    // 6. Return hex-encoded hash
    return hex.EncodeToString(hash[:]), nil
}
```

## Edge Cases and Considerations

### 1. Empty/Nil Fields
- **Empty SystemPrompt**: Include as empty string in hash
- **Empty Messages**: Still hashable (represents initial call)
- **No Tools**: Include as empty array `[]`
- **Zero Temperature**: Include as `0.0`

### 2. Message Order
- Message order is semantically significant (conversation history)
- Must preserve order in hash computation
- Different order = different cache key (correct behavior)

### 3. Tool Parameters
- Tool parameters are maps (unordered)
- Must canonicalize by sorting keys before JSON encoding
- Use `json.Marshal` which sorts map keys by default in Go

### 4. Temperature Precision
- Float comparison can be tricky
- Different bit representations of same value should hash identically
- Solution: JSON marshaling normalizes floats to consistent representation

### 5. Whitespace
- Leading/trailing whitespace shouldn't create different cache keys
- Solution: Trim strings before including in hash
- Internal whitespace is significant (code snippets, etc.)

### 6. Tool Ordering
- Tools could be provided in different orders
- Tool order doesn't affect LLM behavior
- **Decision**: Sort tools by name before hashing to allow reordering flexibility

### 7. Large Messages
- Long conversations will have large JSON payloads
- SHA256 is efficient even for large inputs
- Performance impact is acceptable for API call savings

## Collision Analysis

### Probability
- SHA256 has 2^256 possible values
- For 1 million cached requests, birthday paradox gives collision probability of ~10^-69
- Effectively zero for practical purposes

### What if Collision Occurs?
- False positive: Different requests return same cached response
- Impact: Wrong response returned to user
- Mitigation:
  - Validate cache entry before returning (check system prompt matches)
  - Cache hit rate is still high enough that collision risk is acceptable
  - Can add checksum validation if needed

## Performance Considerations

### Benchmark Estimates
- JSON marshaling: ~1-5μs per request
- SHA256 computation: ~500ns per request
- Total overhead: ~2-10μs per API call
- API call time: ~500-5000ms
- **Overhead: <0.002% of API call time**

### Memory Usage
- CacheKeyRequest struct: ~size of original request
- Temporary JSON bytes: ~size of original request
- Released after hash computation
- **Acceptable for the performance benefit**

## Alternatives Considered

### 1. Simple String Concatenation
```
key = systemPrompt + "|" + messages + "|" + tools + "|" + temperature
```
- ❌ Not collision-resistant
- ❌ Delimiter collision issues
- ❌ No canonicalization

### 2. Hash Each Field Separately
```
key = hash(systemPrompt) + hash(messages) + ...
```
- ✅ More flexible for partial cache invalidation
- ❌ More complex
- ❌ Not needed for current use case

### 3. MD5 Hash
- ❌ MD5 is cryptographically broken
- ❌ Higher collision probability
- ✅ Faster (but not worth the risk)

### 4. Full Request Hashing (Include MaxTokens)
- ❌ Reduces cache hit rate
- ❌ MaxTokens doesn't affect response content
- ❌ Unnecessary differentiation

## Testing Strategy

### Unit Tests
1. **Determinism**: Same request produces same key
2. **Uniqueness**: Different requests produce different keys
3. **Field Sensitivity**: Changing each field produces different key
4. **Order Independence**: Tool reordering produces same key (after sorting)
5. **Order Dependence**: Message reordering produces different key
6. **Edge Cases**: Empty fields, nil slices, special characters

### Integration Tests
1. **Real Workloads**: Generate keys from actual agent requests
2. **Collision Testing**: Verify no collisions in realistic dataset
3. **Performance**: Measure key generation time

## Future Enhancements

### 1. Selective Field Hashing
- Allow excluding certain message content (timestamps, IDs)
- Configurable field selection for specific use cases

### 2. Semantic Normalization
- Normalize code formatting before hashing
- Remove irrelevant whitespace in code blocks

### 3. Fuzzy Matching
- Detect "similar enough" requests for partial cache hits
- Use embedding similarity for semantic caching

## Conclusion

The proposed strategy uses SHA256 hashing of canonicalized JSON containing:
- System prompt
- Messages (role, content, tool_id)
- Tools (sorted by name)
- Temperature

This approach provides:
- ✅ Strong uniqueness guarantees
- ✅ Deterministic key generation
- ✅ Minimal performance overhead
- ✅ Simple implementation
- ✅ Compatibility with existing codebase patterns

The design is ready for implementation in subtask 2-2.
