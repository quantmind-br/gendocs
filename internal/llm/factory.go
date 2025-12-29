package llm

import (
	"fmt"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llmcache"
)

// Factory creates LLM clients
type Factory struct {
	retryClient  *RetryClient
	memoryCache  *llmcache.LRUCache
	diskCache    *llmcache.DiskCache
	cacheEnabled bool
	cacheTTL     time.Duration
}

// NewFactory creates a new LLM factory
// Optional cache parameters can be provided to enable caching
func NewFactory(retryClient *RetryClient, memoryCache *llmcache.LRUCache, diskCache *llmcache.DiskCache, cacheEnabled bool, cacheTTL time.Duration) *Factory {
	return &Factory{
		retryClient:  retryClient,
		memoryCache:  memoryCache,
		diskCache:    diskCache,
		cacheEnabled: cacheEnabled,
		cacheTTL:     cacheTTL,
	}
}

// CreateClient creates an LLM client based on the provider configuration
// If caching is enabled and cache instances are available, wraps the client with caching
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
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: openai, anthropic, gemini)", cfg.Provider)
	}

	// Wrap with caching if enabled and cache instances are available
	if f.cacheEnabled && f.memoryCache != nil {
		ttl := f.cacheTTL
		if ttl == 0 {
			ttl = llmcache.DefaultTTL
		}
		return NewCachedLLMClient(baseClient, f.memoryCache, f.diskCache, true, ttl), nil
	}

	return baseClient, nil
}
