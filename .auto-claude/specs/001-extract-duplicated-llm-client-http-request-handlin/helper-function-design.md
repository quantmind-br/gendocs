# doHTTPRequest Helper Function Design

## Overview

The `doHTTPRequest` helper function is a centralized HTTP request handler that eliminates code duplication across OpenAI, Anthropic, and Gemini LLM clients. It encapsulates the common pattern of HTTP request execution with retry logic, standardizing error handling and response processing.

## Function Signature

```go
// doHTTPRequest executes an HTTP request with retry and standard error handling.
// It handles JSON marshaling, request creation, header setting, execution with retry,
// response reading, and status code validation.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - method: HTTP method (e.g., "GET", "POST")
//   - url: Target URL for the request
//   - headers: Map of HTTP headers to set on the request
//   - body: Request body to marshal as JSON (can be nil for GET requests)
//
// Returns:
//   - []byte: Raw response body bytes for provider-specific parsing
//   - error: Wrapped error with context if any step fails
//
// Error handling:
//   - "failed to marshal request" - JSON marshaling failure
//   - "failed to create request" - HTTP request creation failure
//   - "request failed" - Request execution failure (including retry attempts)
//   - "failed to read response" - Response body reading failure
//   - "API error: status %d, body: %s" - Non-200 status code with response body
func (c *BaseLLMClient) doHTTPRequest(
    ctx context.Context,
    method string,
    url string,
    headers map[string]string,
    body interface{},
) ([]byte, error)
```

## Design Decisions

### 1. Location: BaseLLMClient Method

**Rationale:** Placing `doHTTPRequest` in `BaseLLMClient` allows it to:
- Access the existing `retryClient` field for automatic retries
- Be inherited by all provider-specific clients (OpenAI, Anthropic, Gemini)
- Maintain access control within the LLM package (private method)

**Alternative Considered:** Standalone function in the package
**Rejected:** Would require passing `retryClient` as a parameter, making the API more cumbersome

### 2. Body Parameter: interface{} Type

**Rationale:** Using `interface{}` for the body parameter:
- Allows any struct type to be passed (openaiRequest, anthropicRequest, geminiRequest)
- Maintains type safety at compile-time within each client
- Provides flexibility for future providers

**Usage Pattern:** Each client passes their provider-specific request type:
```go
// OpenAI
oaReq := c.convertRequest(req)
body, err := c.doHTTPRequest(ctx, "POST", url, headers, oaReq)

// Anthropic
anReq := c.convertRequest(req)
body, err := c.doHTTPRequest(ctx, "POST", url, headers, anReq)

// Gemini
geReq := c.convertRequest(req)
body, err := c.doHTTPRequest(ctx, "POST", url, headers, geReq)
```

### 3. Headers Parameter: map[string]string

**Rationale:** Using a map for headers:
- Allows providers to set different headers (Authorization, x-api-key, etc.)
- Simplifies the API compared to using `http.Header` type
- Makes it clear which headers are being set

**Usage Pattern:**
```go
// OpenAI
headers := map[string]string{
    "Content-Type": "application/json",
    "Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
}
body, err := c.doHTTPRequest(ctx, "POST", url, headers, oaReq)

// Anthropic
headers := map[string]string{
    "Content-Type": "application/json",
    "x-api-key": c.apiKey,
    "anthropic-version": "2023-06-01",
}
body, err := c.doHTTPRequest(ctx, "POST", url, headers, anReq)

// Gemini
headers := map[string]string{
    "Content-Type": "application/json",
}
body, err := c.doHTTPRequest(ctx, "POST", url, headers, geReq)
```

### 4. Return Type: Raw []byte

**Rationale:** Returning raw bytes instead of parsing in the helper:
- Allows each provider to parse into their specific response type
- Enables provider-specific error field checking
- Maintains separation of concerns (helper handles HTTP, clients handle parsing)

**Flow:**
```go
// Step 1: Execute HTTP request
body, err := c.doHTTPRequest(ctx, "POST", url, headers, providerReq)
if err != nil {
    return CompletionResponse{}, err
}

// Step 2: Provider-specific parsing
var providerResp providerResponseType
if err := json.Unmarshal(body, &providerResp); err != nil {
    return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
}

// Step 3: Provider-specific error checking
if providerResp.Error != nil {
    return CompletionResponse{}, fmt.Errorf("API error: %s", providerResp.Error.Message)
}

// Step 4: Provider-specific response conversion
return c.convertResponse(providerResp), nil
```

## Implementation Behavior

### Step-by-Step Flow

1. **Marshal Request Body**
   - If `body` parameter is not nil, marshal to JSON
   - Error: `"failed to marshal request: %w"`

2. **Create HTTP Request**
   - Use `http.NewRequestWithContext(ctx, method, url, bodyReader)`
   - If body is nil, use nil reader (for GET requests)
   - Error: `"failed to create request: %w"`

3. **Set Headers**
   - Iterate through `headers` map and set each header
   - Use `httpReq.Header.Set(key, value)`

4. **Execute Request with Retry**
   - Use `c.retryClient.Do(httpReq)`
   - Retry logic is handled by RetryClient (exponential backoff, max attempts)
   - Error: `"request failed: %w"`
   - Always defer `resp.Body.Close()` on success

5. **Read Response Body**
   - Use `io.ReadAll(resp.Body)`
   - Error: `"failed to read response: %w"`

6. **Validate Status Code**
   - Check if `resp.StatusCode != http.StatusOK`
   - Error: `"API error: status %d, body: %s"` (includes status code and response body)

7. **Return Raw Bytes**
   - Return response body bytes for provider-specific parsing

## Error Handling Strategy

All errors are wrapped with context using `fmt.Errorf` with `%w` verb:
- Maintains error chain for error unwrapping
- Provides clear context about which step failed
- Preserves original error for debugging

### Error Message Consistency

The helper maintains **identical** error messages to current implementations:
- `"failed to marshal request: %w"`
- `"failed to create request: %w"`
- `"request failed: %w"`
- `"failed to read response: %w"`
- `"API error: status %d, body: %s"`

This ensures zero breaking changes to error handling behavior.

## Usage Examples

### OpenAI Client (After Refactoring)

```go
func (c *OpenAIClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Provider-specific: convert request format
    oaReq := c.convertRequest(req)

    // COMMON: execute HTTP request
    url := fmt.Sprintf("%s/chat/completions", c.baseURL)
    headers := map[string]string{
        "Content-Type": "application/json",
        "Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
    }
    body, err := c.doHTTPRequest(ctx, "POST", url, headers, oaReq)
    if err != nil {
        return CompletionResponse{}, err
    }

    // Provider-specific: parse and validate response
    var oaResp openaiResponse
    if err := json.Unmarshal(body, &oaResp); err != nil {
        return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
    }

    if oaResp.Error != nil {
        return CompletionResponse{}, fmt.Errorf("API error: %s", oaResp.Error.Message)
    }

    return c.convertResponse(oaResp), nil
}
```

### Anthropic Client (After Refactoring)

```go
func (c *AnthropicClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Provider-specific: convert request format
    anReq := c.convertRequest(req)

    // COMMON: execute HTTP request
    url := fmt.Sprintf("%s/v1/messages", c.baseURL)
    headers := map[string]string{
        "Content-Type": "application/json",
        "x-api-key": c.apiKey,
        "anthropic-version": "2023-06-01",
    }
    body, err := c.doHTTPRequest(ctx, "POST", url, headers, anReq)
    if err != nil {
        return CompletionResponse{}, err
    }

    // Provider-specific: parse and validate response
    var anResp anthropicResponse
    if err := json.Unmarshal(body, &anResp); err != nil {
        return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
    }

    if anResp.Error != nil {
        return CompletionResponse{}, fmt.Errorf("API error: %s", anResp.Error.Message)
    }

    return c.convertResponse(anResp), nil
}
```

### Gemini Client (After Refactoring)

```go
func (c *GeminiClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Provider-specific: convert request format
    geReq := c.convertRequest(req)

    // COMMON: execute HTTP request
    url := fmt.Sprintf("%s/v1beta/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
    headers := map[string]string{
        "Content-Type": "application/json",
    }
    body, err := c.doHTTPRequest(ctx, "POST", url, headers, geReq)
    if err != nil {
        return CompletionResponse{}, err
    }

    // Provider-specific: parse and validate response
    var geResp geminiResponse
    if err := json.Unmarshal(body, &geResp); err != nil {
        return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
    }

    if geResp.Error != nil {
        return CompletionResponse{}, fmt.Errorf("API error: %s", geResp.Error.Message)
    }

    // Additional Gemini-specific validation
    if len(geResp.Candidates) == 0 {
        return CompletionResponse{}, fmt.Errorf("no candidates in response")
    }

    return c.convertResponse(geResp), nil
}
```

## Benefits

### 1. Code Reduction
- **Before:** ~101 lines of duplicated code across 3 files
- **After:** ~30 lines in one location (client.go) + simplified client methods
- **Reduction:** ~70 lines of code

### 2. Maintainability
- Single source of truth for HTTP request handling
- Bug fixes and improvements applied in one place
- Consistent behavior across all providers

### 3. Testability
- HTTP request logic can be tested once
- Easier to mock and test error scenarios
- Reduced test duplication

### 4. Consistency
- Identical error messages across all providers
- Same retry behavior for all providers
- Uniform status code handling

### 5. Provider-Specific Flexibility
- Each provider maintains their unique logic
- Request/response conversion stays in client
- Provider-specific headers can vary
- Provider-specific error field checking preserved

## Implementation Notes

### Thread Safety
- The method is thread-safe as it operates only on:
  - Input parameters (ctx, method, url, headers, body)
  - The `retryClient` field (which is thread-safe)
- No mutable state is modified

### Context Support
- Context is properly passed to `http.NewRequestWithContext`
- Enables request cancellation and timeout
- Respects context cancellation during retry attempts

### Memory Management
- Response body is always closed via `defer resp.Body.Close()`
- Response body is read completely into memory
- For very large responses, streaming could be considered (not needed for current use case)

### Extensibility
- The design allows for easy addition of new providers
- New providers only need to implement:
  - `convertRequest()` - to create provider-specific request
  - `convertResponse()` - to parse provider-specific response
  - Provider-specific error checking
- No modifications to `doHTTPRequest` needed

## Verification Criteria

The implementation will be considered complete when:

1. **Function Signature**: Method added to BaseLLMClient with exact signature specified
2. **JSON Marshaling**: Body parameter correctly marshaled to JSON
3. **Request Creation**: HTTP request created with context
4. **Header Setting**: All headers from map applied to request
5. **Retry Execution**: Request executed via retryClient.Do
6. **Response Reading**: Response body fully read and bytes returned
7. **Error Handling**: All error cases properly wrapped with correct messages
8. **Status Validation**: Non-200 responses return error with status and body
9. **Resource Cleanup**: Response body always closed
10. **Tests Pass**: All existing LLM client tests continue to pass

## Next Steps

This design document provides the blueprint for **Phase 2: Implement HTTP Request Helper**, which will implement the `doHTTPRequest` method in `internal/llm/client.go` following the exact specification outlined above.
