package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/user/gendocs/internal/llmcache"
)

// CachedLLMClient wraps an LLMClient with caching functionality.
//
// The client implements a two-tier caching strategy:
// 1. Memory cache (LRU): Fast in-memory cache for frequently accessed responses
// 2. Disk cache: Persistent cache across program restarts
//
// Cache hits avoid making API calls entirely, saving both cost and latency.
// When an entry is found in the disk cache, it's promoted to the memory cache.
type CachedLLMClient struct {
	client      LLMClient                // Underlying LLM client
	memoryCache *llmcache.LRUCache       // In-memory LRU cache
	diskCache   *llmcache.DiskCache      // Persistent disk cache
	enabled     bool                     // Enable/disable caching
	ttl         time.Duration            // Time-to-live for cache entries
}

// NewCachedLLMClient creates a new cached LLM client.
//
// client: The underlying LLM client to wrap
// memoryCache: In-memory LRU cache (can be nil)
// diskCache: Persistent disk cache (can be nil)
// enabled: Whether caching is enabled
// ttl: Time-to-live for cached responses
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

// GenerateCompletion implements LLMClient interface with caching.
//
// The caching strategy is:
// 1. If caching is disabled, delegate directly to the underlying client
// 2. Check memory cache for a hit
// 3. Check disk cache for a hit (promote to memory cache if found)
// 4. Call underlying client and cache the successful response
//
// Cache key generation failures are handled gracefully by bypassing the cache.
// API errors are not cached.
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

// SupportsTools delegates to underlying client.
func (c *CachedLLMClient) SupportsTools() bool {
	return c.client.SupportsTools()
}

// GetProvider returns the underlying provider name with "cached-" prefix.
// This makes it easy to identify when a cached client is being used.
func (c *CachedLLMClient) GetProvider() string {
	return fmt.Sprintf("cached-%s", c.client.GetProvider())
}

// GetStats returns aggregated statistics from both memory and disk cache.
//
// Combines hits, misses, and evictions from both caches.
// Recalculates the hit rate based on the combined data.
func (c *CachedLLMClient) GetStats() llmcache.CacheStats {
	if c.memoryCache == nil {
		return llmcache.CacheStats{}
	}

	// Get memory cache stats
	memStats := c.memoryCache.Stats()

	// Aggregate with disk cache stats if available
	if c.diskCache != nil {
		diskStats := c.diskCache.Stats()

		// Combine statistics
		memStats.Hits += diskStats.Hits
		memStats.Misses += diskStats.Misses
		memStats.Evictions += diskStats.Evictions
		// Recalculate hit rate with combined data
		total := memStats.Hits + memStats.Misses
		if total > 0 {
			memStats.HitRate = float64(memStats.Hits) / float64(total)
		}
	}

	return memStats
}

// CleanupExpired removes expired entries from both caches.
//
// Returns the number of entries removed from each cache.
// Note: diskExpired is always 0 as the disk cache doesn't report counts.
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

// Clear clears both memory and disk cache.
// The disk cache is immediately saved to disk after clearing.
func (c *CachedLLMClient) Clear() error {
	if c.memoryCache != nil {
		c.memoryCache.Clear()
	}

	if c.diskCache != nil {
		return c.diskCache.Clear()
	}

	return nil
}

// GetUnderlyingClient returns the underlying unwrapped LLM client.
// This is useful when you need to bypass the caching layer.
func (c *CachedLLMClient) GetUnderlyingClient() LLMClient {
	return c.client
}
