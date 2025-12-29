package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/user/gendocs/internal/llmcache"
)

// CachedLLMClient wraps an LLMClient with caching functionality
type CachedLLMClient struct {
	client      LLMClient                // Underlying LLM client
	memoryCache *llmcache.LRUCache       // In-memory LRU cache
	diskCache   *llmcache.DiskCache      // Persistent disk cache
	enabled     bool                     // Enable/disable caching
	ttl         time.Duration            // Time-to-live for cache entries
}

// NewCachedLLMClient creates a new cached LLM client
func NewCachedLLMClient(
	client LLMClient,
	memoryCache *llmcache.LRUCache,
	diskCache *llmcache.DiskCache,
	enabled bool,
	ttl time.Duration,
) *CachedLLMClient {
	return &CachedLLMClient{
		client:      client,
		memoryCache: memoryCache,
		diskCache:   diskCache,
		enabled:     enabled,
		ttl:         ttl,
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
		// Key generation failed, bypass cache gracefully
		return c.client.GenerateCompletion(ctx, req)
	}

	// 2. Check memory cache first
	if cached, found := c.memoryCache.Get(cacheKey); found {
		// Cache hit in memory - return cached response immediately
		return cached.Response, nil
	}

	// 3. Check disk cache if memory cache miss
	if c.diskCache != nil {
		if cached, found := c.diskCache.Get(cacheKey); found {
			// Cache hit on disk - promote to memory cache and return
			c.memoryCache.Put(cacheKey, &cached)
			return cached.Response, nil
		}
	}

	// 4. Cache miss - call underlying client
	resp, err := c.client.GenerateCompletion(ctx, req)
	if err != nil {
		// API call failed, don't cache error responses
		return CompletionResponse{}, err
	}

	// 5. Cache the successful response
	cachedResp := llmcache.NewCachedResponse(cacheKey, llmcache.CacheKeyRequestFrom(req), resp, c.ttl)

	// Store in memory cache
	c.memoryCache.Put(cacheKey, cachedResp)

	// Store in disk cache (best-effort, non-blocking)
	if c.diskCache != nil {
		if err := c.diskCache.Put(cacheKey, cachedResp); err != nil {
			// Disk cache write failure is acceptable - just lose persistence benefit
			// TODO: Add logging in subtask 4-2
		}
	}

	return resp, nil
}

// SupportsTools delegates to underlying client
func (c *CachedLLMClient) SupportsTools() bool {
	return c.client.SupportsTools()
}

// GetProvider returns the underlying provider name with "cached-" prefix
func (c *CachedLLMClient) GetProvider() string {
	return fmt.Sprintf("cached-%s", c.client.GetProvider())
}

// GetStats returns statistics from both memory and disk cache
func (c *CachedLLMClient) GetStats() llmcache.CacheStats {
	if c.memoryCache == nil {
		return llmcache.CacheStats{}
	}
	return c.memoryCache.Stats()
}

// CleanupExpired removes expired entries from both caches
func (c *CachedLLMClient) CleanupExpired() (memoryExpired, diskExpired int) {
	memoryExpired = 0
	if c.memoryCache != nil {
		memoryExpired = c.memoryCache.CleanupExpired()
	}

	diskExpired = 0
	if c.diskCache != nil {
		if err := c.diskCache.CleanupExpired(); err == nil {
			// Disk cache cleanup succeeded
			// Note: DiskCache.CleanupExpired doesn't return count, so we can't track it
		}
	}

	return memoryExpired, diskExpired
}

// Clear clears both memory and disk cache
func (c *CachedLLMClient) Clear() error {
	if c.memoryCache != nil {
		c.memoryCache.Clear()
	}

	if c.diskCache != nil {
		return c.diskCache.Clear()
	}

	return nil
}

// GetUnderlyingClient returns the underlying unwrapped LLM client
func (c *CachedLLMClient) GetUnderlyingClient() LLMClient {
	return c.client
}
