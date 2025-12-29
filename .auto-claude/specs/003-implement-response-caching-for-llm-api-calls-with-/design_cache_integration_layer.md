# Cache Integration Layer Design

## Overview
This document describes where and how to integrate the caching layer into the LLM call flow, using a decorator pattern around the LLMClient interface.

## Current Architecture Analysis

### LLM Client Interface
```go
type LLMClient interface {
    GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    SupportsTools() bool
    GetProvider() string
}
```

### Current LLM Call Flow
```
┌─────────────────────────────────────────────────────────────┐
│  Agent Creation                                             │
│  - Create LLM Factory                                       │
│  - Factory.CreateClient(cfg) → LLMClient (OpenAI/Anthropic) │
│  - Create BaseAgent with LLMClient                          │
└─────────────────────────────────────────────────────────────┘
                            │
                            v
┌─────────────────────────────────────────────────────────────┐
│  Agent Execution                                            │
│  - BaseAgent.RunOnce()                                      │
│  - Build CompletionRequest                                  │
│  - llmClient.GenerateCompletion(ctx, req)                   │
└─────────────────────────────────────────────────────────────┘
                            │
                            v
┌─────────────────────────────────────────────────────────────┐
│  LLM Provider Client                                        │
│  - OpenAIClient / AnthropicClient / GeminiClient           │
│  - Convert request to provider format                       │
│  - Make HTTP API call                                       │
│  - Return response                                          │
└─────────────────────────────────────────────────────────────┘
```

### Key Integration Point
**All LLM calls go through a single method**: `LLMClient.GenerateCompletion()`

This makes it ideal to use a **decorator pattern** to add caching transparently.

## Integration Strategy: Decorator Pattern

### Design Decision: Decorator Around LLMClient

**Why Decorator Pattern?**

✅ **Non-invasive**: No changes needed to existing LLM client implementations
✅ **Transparent**: Agents and calling code don't need to know about caching
✅ **Flexible**: Can be enabled/disabled via configuration
✅ **Testable**: Cache logic isolated in separate component
✅ **Follows existing patterns**: `RetryClient` already uses a similar pattern

**Architecture:**
```
┌─────────────────────────────────────────────────────────┐
│  CachedLLMClient (Decorator)                            │
│  - Implements LLMClient interface                       │
│  - Wraps underlying LLMClient                           │
│  - Intercepts GenerateCompletion() calls                │
│  - Checks cache before calling underlying client        │
└─────────────────────────────────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────────────┐
│  Underlying LLMClient                                   │
│  - OpenAIClient                                         │
│  - AnthropicClient                                      │
│  - GeminiClient                                         │
└─────────────────────────────────────────────────────────┘
```

## Detailed Integration Flow

### 1. CachedLLMClient Structure

```go
// internal/llm/cached_client.go

package llm

import (
    "context"
    "fmt"

    "github.com/user/gendocs/internal/llmcache"
)

// CachedLLMClient wraps an LLMClient with caching functionality
type CachedLLMClient struct {
    client    LLMClient              // Underlying LLM client
    cache     *llmcache.Cache        // Two-tier cache (memory + disk)
    enabled   bool                   // Enable/disable caching
}

// NewCachedLLMClient creates a new cached LLM client
func NewCachedLLMClient(
    client LLMClient,
    cache *llmcache.Cache,
    enabled bool,
) *CachedLLMClient {
    return &CachedLLMClient{
        client:  client,
        cache:   cache,
        enabled: enabled,
    }
}

// GenerateCompletion implements LLMClient interface with caching
func (c *CachedLLMClient) GenerateCompletion(
    ctx context.Context,
    req CompletionRequest,
) (CompletionResponse, error) {
    // If caching disabled, delegate directly
    if !c.enabled {
        return c.client.GenerateCompletion(ctx, req)
    }

    // 1. Generate cache key from request
    cacheKey, err := llmcache.GenerateCacheKey(req)
    if err != nil {
        // Key generation failed, bypass cache
        return c.client.GenerateCompletion(ctx, req)
    }

    // 2. Check cache for existing response
    if cached, found := c.cache.Get(cacheKey); found {
        // Cache hit - return cached response
        return cached.Response, nil
    }

    // 3. Cache miss - call underlying client
    resp, err := c.client.GenerateCompletion(ctx, req)
    if err != nil {
        // API call failed, don't cache
        return CompletionResponse{}, err
    }

    // 4. Cache the successful response
    cachedResp := &llmcache.CachedResponse{
        Key:       cacheKey,
        Request:   llmcache.CacheKeyRequestFrom(req), // Convert to cache key format
        Response:  resp,
        CreatedAt: time.Now(),
        SizeBytes: calculateSize(resp),
    }

    if err := c.cache.Put(cacheKey, cachedResp); err != nil {
        // Log warning but don't fail - caching is best-effort
        // Cache write failure shouldn't break the flow
    }

    return resp, nil
}

// SupportsTools delegates to underlying client
func (c *CachedLLMClient) SupportsTools() bool {
    return c.client.SupportsTools()
}

// GetProvider returns the underlying provider name
func (c *CachedLLMClient) GetProvider() string {
    return fmt.Sprintf("cached-%s", c.client.GetProvider())
}
```

### 2. Factory Integration

```go
// internal/llm/factory.go (Modified)

package llm

import (
    "fmt"

    "github.com/user/gendocs/internal/config"
    "github.com/user/gendocs/internal/llmcache"
)

// Factory creates LLM clients
type Factory struct {
    retryClient *RetryClient
    cache       *llmcache.Cache // Cache instance
}

// NewFactory creates a new LLM factory
func NewFactory(retryClient *RetryClient, cache *llmcache.Cache) *Factory {
    return &Factory{
        retryClient: retryClient,
        cache:       cache,
    }
}

// CreateClient creates an LLM client based on the provider configuration
func (f *Factory) CreateClient(cfg config.LLMConfig) (LLMClient, error) {
    // Create base client (without caching)
    var baseClient LLMClient
    switch cfg.Provider {
    case "openai":
        baseClient = NewOpenAIClient(cfg, f.retryClient)
    case "anthropic":
        baseClient = NewAnthropicClient(cfg, f.retryClient)
    case "gemini":
        baseClient = NewGeminiClient(cfg, f.retryClient)
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
    }

    // Wrap with caching if enabled
    if cfg.Cache.Enabled && f.cache != nil {
        return NewCachedLLMClient(baseClient, f.cache, true), nil
    }

    return baseClient, nil
}
```

### 3. Configuration Integration

```go
// internal/config/models.go (Extended)

package config

// LLMConfig holds LLM client configuration
type LLMConfig struct {
    Provider string      `mapstructure:"provider"` // openai, anthropic, gemini
    APIKey   string      `mapstructure:"api_key"`
    BaseURL  string      `mapstructure:"base_url"`
    Model    string      `mapstructure:"model"`
    Cache    CacheConfig `mapstructure:"cache"` // Cache configuration
    Retries  int         `mapstructure:"retries"`
}

// CacheConfig holds LLM response cache configuration
type CacheConfig struct {
    Enabled     bool   `mapstructure:"enabled"`       // Enable/disable caching
    MaxMemoryMB int    `mapstructure:"max_memory_mb"` // Max memory cache size in MB
    MaxDiskMB   int    `mapstructure:"max_disk_mb"`   // Max disk cache size in MB
    TTLDays     int    `mapstructure:"ttl_days"`      // Time-to-live in days
    CachePath   string `mapstructure:"cache_path"`    // Path to cache file
    AutoSaveSec int    `mapstructure:"auto_save_sec"` // Auto-save interval in seconds
}

// GetCacheConfig returns default cache config if not set
func (c *LLMConfig) GetCacheConfig() CacheConfig {
    if c.Cache.MaxMemoryMB == 0 {
        c.Cache.MaxMemoryMB = 100 // Default 100MB
    }
    if c.Cache.MaxDiskMB == 0 {
        c.Cache.MaxDiskMB = 500 // Default 500MB
    }
    if c.Cache.TTLDays == 0 {
        c.Cache.TTLDays = 7 // Default 7 days
    }
    if c.Cache.CachePath == "" {
        c.Cache.CachePath = ".ai/llm_cache.json"
    }
    if c.Cache.AutoSaveSec == 0 {
        c.Cache.AutoSaveSec = 30 // Default 30 seconds
    }
    return c.Cache
}
```

### 4. Application Initialization

```go
// cmd/root.go or main initialization

import (
    "github.com/user/gendocs/internal/config"
    "github.com/user/gendocs/internal/llm"
    "github.com/user/gendocs/internal/llmcache"
)

// Initialize LLM factory with cache support
func initializeLLMFactory(cfg *config.Config) (*llm.Factory, error) {
    // Create retry client
    retryClient := llm.NewRetryClient(nil)

    // Create cache if enabled
    var cache *llmcache.Cache
    if cfg.LLM.Cache.Enabled {
        var err error
        cache, err = llmcache.NewCache(cfg.LLM.Cache)
        if err != nil {
            return nil, fmt.Errorf("failed to create cache: %w", err)
        }
    }

    // Create factory with cache
    factory := llm.NewFactory(retryClient, cache)

    return factory, nil
}
```

## Complete Call Flow with Caching

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent Execution                                                │
│  - BaseAgent.RunOnce()                                          │
│  - Build CompletionRequest                                      │
│  - llmClient.GenerateCompletion(ctx, req)                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                v
┌─────────────────────────────────────────────────────────────────┐
│  CachedLLMClient (Decorator)                                   │
│  1. Generate cache key from request                             │
│  2. Check cache.Get(cacheKey)                                   │
│     ├─ Cache Hit → Return cached response immediately           │
│     └─ Cache Miss → Continue                                    │
│  3. Call underlying client.GenerateCompletion()                 │
│  4. On success, cache.Put(cacheKey, response)                   │
│  5. Return response                                             │
└─────────────────────────────────────────────────────────────────┘
                                │
                                v
┌─────────────────────────────────────────────────────────────────┐
│  Cache (Two-Tier)                                               │
│  1. Check in-memory LRU cache (O(1))                            │
│  2. If miss, check disk cache                                   │
│  3. On disk hit, promote to memory                              │
│  4. On miss, return not-found                                   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                v
┌─────────────────────────────────────────────────────────────────┐
│  Underlying LLMClient (OpenAI/Anthropic/Gemini)                 │
│  - Convert request to provider format                           │
│  - Make HTTP API call via RetryClient                          │
│  - Return response                                              │
└─────────────────────────────────────────────────────────────────┘
```

## Error Handling Strategy

### Cache Failures Should Be Non-Blocking

**Principle**: Cache failures should never prevent the application from working

1. **Cache Key Generation Failure**
   - Action: Bypass cache, call underlying client directly
   - Reason: Key generation errors indicate malformed requests

2. **Cache Read Failure**
   - Action: Treat as cache miss, call underlying client
   - Reason: Cache corruption or I/O error shouldn't break functionality

3. **Cache Write Failure**
   - Action: Log warning, return response successfully
   - Reason: Failed to cache is acceptable, just lose optimization benefit

4. **Underlying Client Failure**
   - Action: Return error to caller, don't cache error responses
   - Reason: Errors shouldn't be cached (might be transient)

```go
// Example error handling
func (c *CachedLLMClient) GenerateCompletion(
    ctx context.Context,
    req CompletionRequest,
) (CompletionResponse, error) {
    if !c.enabled {
        return c.client.GenerateCompletion(ctx, req)
    }

    // Generate cache key - fail gracefully
    cacheKey, err := llmcache.GenerateCacheKey(req)
    if err != nil {
        // Key generation failed, bypass cache
        return c.client.GenerateCompletion(ctx, req)
    }

    // Try cache - fail open on errors
    if cached, found := c.cache.Get(cacheKey); found {
        return cached.Response, nil
    }

    // Call underlying client
    resp, err := c.client.GenerateCompletion(ctx, req)
    if err != nil {
        // Don't cache errors
        return CompletionResponse{}, err
    }

    // Cache response - best effort
    cachedResp := &llmcache.CachedResponse{...}
    if err := c.cache.Put(cacheKey, cachedResp); err != nil {
        // Log but don't fail - cache write failure is acceptable
    }

    return resp, nil
}
```

## Configuration Examples

### Minimal Configuration (Caching Disabled)
```yaml
llm:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: gpt-4
  cache:
    enabled: false
```

### Development Configuration
```yaml
llm:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: gpt-4
  cache:
    enabled: true
    max_memory_mb: 10
    max_disk_mb: 50
    ttl_days: 1
    cache_path: ".ai/llm_cache.json"
    auto_save_sec: 60
```

### Production Configuration
```yaml
llm:
  provider: anthropic
  api_key: ${ANTHROPIC_API_KEY}
  model: claude-3-opus-20240229
  cache:
    enabled: true
    max_memory_mb: 100
    max_disk_mb: 500
    ttl_days: 7
    cache_path: ".ai/llm_cache.json"
    auto_save_sec: 30
```

## Benefits of This Approach

### 1. Separation of Concerns
- Cache logic is isolated in `CachedLLMClient`
- LLM providers don't need to know about caching
- Agents don't need to know about caching

### 2. Testability
- Can test cache independently
- Can test LLM clients independently
- Can test decorator behavior separately

### 3. Flexibility
- Caching can be enabled/disabled via config
- Different providers can have different cache settings
- Can swap cache implementations without touching client code

### 4. Maintainability
- Clear, single responsibility for each component
- Easy to add new caching features (TTL, eviction, etc.)
- Easy to debug (can disable caching to isolate issues)

### 5. Performance
- Zero overhead when caching disabled
- Minimal overhead when enabled (key generation + cache lookup)
- Cache hits avoid entire API call latency

## Alternative Approaches Considered

### 1. Cache Inside Each Provider Client
```
❌ Rejected
- Requires duplicating cache logic in each client
- Violates DRY principle
- Harder to maintain and test
```

### 2. Cache in BaseAgent
```
❌ Rejected
- Mixes agent logic with caching
- Requires modifying all agent types
- Can't cache non-agent LLM calls
```

### 3. Middleware/Interceptor Pattern
```
✅ Viable alternative
- Similar to decorator but more complex
- Would allow chaining multiple behaviors
- Rejected: Overkill for this use case
```

### 4. Global Cache with Manual Cache Calls
```
❌ Rejected
- Requires modifying every call site
- Error-prone (easy to forget cache check)
- Invasive changes throughout codebase
```

## Migration Strategy

### Phase 1: Add Cache Infrastructure (No Impact)
1. Create `internal/llmcache` package
2. Implement cache structures
3. Add configuration to config system
4. Update factory to accept cache (but don't use it yet)

### Phase 2: Implement Decorator
1. Create `CachedLLMClient` decorator
2. Update factory to wrap clients when cache enabled
3. Add unit tests for decorator
4. Integration test with real cache

### Phase 3: Enable Caching Gradually
1. Test with caching disabled (existing behavior)
2. Enable caching for development/testing
3. Monitor cache hit rates and performance
4. Enable for production when confident

## Testing Strategy

### Unit Tests for CachedLLMClient
```go
func TestCachedLLMClient_CacheHit(t *testing.T) {
    // Setup mock underlying client
    mockClient := &MockLLMClient{
        response: CompletionResponse{Content: "test"},
    }

    // Setup cache with pre-populated entry
    cache := setupCacheWithEntry(t)

    // Create cached client
    cachedClient := NewCachedLLMClient(mockClient, cache, true)

    // Make request - should hit cache
    resp, err := cachedClient.GenerateCompletion(ctx, req)

    // Verify: response from cache, underlying client not called
    assert.Equal(t, "test", resp.Content)
    assert.Equal(t, 0, mockClient.callCount)
}

func TestCachedLLMClient_CacheMiss(t *testing.T) {
    // Setup mock underlying client
    mockClient := &MockLLMClient{
        response: CompletionResponse{Content: "test"},
    }

    // Setup empty cache
    cache := setupEmptyCache(t)

    // Create cached client
    cachedClient := NewCachedLLMClient(mockClient, cache, true)

    // Make request - should miss cache
    resp, err := cachedClient.GenerateCompletion(ctx, req)

    // Verify: underlying client called, response cached
    assert.Equal(t, "test", resp.Content)
    assert.Equal(t, 1, mockClient.callCount)
    assertCached(t, cache, req)
}

func TestCachedLLMClient_CacheDisabled(t *testing.T) {
    // Setup mock underlying client
    mockClient := &MockLLMClient{
        response: CompletionResponse{Content: "test"},
    }

    // Setup cache
    cache := setupEmptyCache(t)

    // Create cached client with caching disabled
    cachedClient := NewCachedLLMClient(mockClient, cache, false)

    // Make request
    resp, err := cachedClient.GenerateCompletion(ctx, req)

    // Verify: underlying client called, not cached
    assert.Equal(t, "test", resp.Content)
    assert.Equal(t, 1, mockClient.callCount)
    assertNotCached(t, cache, req)
}
```

### Integration Tests
```go
func TestE2E_CachingFlow(t *testing.T) {
    // Setup real LLM client with caching
    factory := setupFactoryWithCache(t)
    client, _ := factory.CreateClient(config.LLMConfig{
        Provider: "openai",
        Cache:    config.CacheConfig{Enabled: true},
    })

    // First call - cache miss
    resp1, err := client.GenerateCompletion(ctx, req1)
    assert.NoError(t, err)

    // Identical call - cache hit
    resp2, err := client.GenerateCompletion(ctx, req1)
    assert.NoError(t, err)

    // Verify responses are identical
    assert.Equal(t, resp1, resp2)

    // Different call - cache miss
    resp3, err := client.GenerateCompletion(ctx, req2)
    assert.NoError(t, err)
    assert.NotEqual(t, resp1, resp3)
}
```

## Observability and Metrics

### Cache Statistics
```go
type CacheStats struct {
    Hits          int64   // Number of cache hits
    Misses        int64   // Number of cache misses
    Evictions     int64   // Number of entries evicted
    HitRate       float64 // Hits / (Hits + Misses)
    Size          int     // Current number of entries
    TotalSizeBytes int64  // Total cache size in bytes
}

// GetStats returns cache statistics
func (c *CachedLLMClient) GetStats() CacheStats {
    return c.cache.GetStats()
}
```

### Logging
```go
// Cache hit
logger.Debug("Cache hit",
    logging.String("key", cacheKey[:16]+"..."),
    logging.Int("age_seconds", int(time.Since(cached.CreatedAt).Seconds())),
)

// Cache miss
logger.Debug("Cache miss",
    logging.String("key", cacheKey[:16]+"..."),
)

// Response cached
logger.Debug("Response cached",
    logging.String("key", cacheKey[:16]+"..."),
    logging.Int("size_bytes", cached.SizeBytes),
)
```

## Summary

The cache integration layer uses a **decorator pattern** to add caching transparently to LLM clients:

✅ **CachedLLMClient** wraps any LLMClient implementation
✅ **Intercepts GenerateCompletion()** to check cache before calling API
✅ **Factory integration** allows easy enabling/disabling via config
✅ **Non-blocking** - cache failures don't break functionality
✅ **Zero overhead** when disabled
✅ **Minimal overhead** when enabled (SHA256 hash + cache lookup)
✅ **Clear separation** of concerns (cache logic isolated)
✅ **Follows existing patterns** (similar to RetryClient)

This design is ready for implementation in **subtask 3-1** (Create caching LLM client decorator).

## Next Steps

1. **Subtask 2-1 to 2-5**: Implement cache infrastructure (llmcache package)
2. **Subtask 3-1**: Implement CachedLLMClient decorator
3. **Subtask 3-2**: Update LLM factory to support caching
4. **Subtask 3-3**: Add cache configuration to config system
5. **Subtask 3-4**: Integrate caching in agent creation
