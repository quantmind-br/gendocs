package llmcache

import (
	"encoding/json"
	"time"

	"github.com/user/gendocs/internal/llm"
)

// CachedResponse represents a cached LLM response with metadata
type CachedResponse struct {
	Key         string                 `json:"key"`           // Cache key (SHA256 hash)
	Request     CacheKeyRequest        `json:"request"`       // Original request (for validation)
	Response    llm.CompletionResponse `json:"response"`      // LLM response
	CreatedAt   time.Time              `json:"created_at"`    // When cached
	ExpiresAt   time.Time              `json:"expires_at"`    // When entry expires
	SizeBytes   int64                  `json:"size_bytes"`    // Approximate size in memory
	AccessCount int                    `json:"access_count"`  // Number of times accessed
}

// CacheKeyRequest represents the fields used for cache key generation
type CacheKeyRequest struct {
	SystemPrompt string             `json:"system_prompt"`
	Messages     []CacheKeyMessage  `json:"messages"`
	Tools        []CacheKeyTool     `json:"tools"`
	Temperature  float64            `json:"temperature"`
}

// CacheKeyMessage represents a message in cache key generation
type CacheKeyMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	ToolID  string `json:"tool_id,omitempty"`
}

// CacheKeyTool represents a tool in cache key generation
type CacheKeyTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// EstimateSize calculates the approximate size of this entry in bytes
func (cr *CachedResponse) EstimateSize() int64 {
	// Serialize to JSON to estimate size
	data, err := json.Marshal(cr)
	if err != nil {
		// Fallback estimate if marshaling fails
		return 1024 // Assume 1KB
	}
	return int64(len(data))
}

// IsExpired checks if this cache entry has expired
func (cr *CachedResponse) IsExpired() bool {
	return time.Now().After(cr.ExpiresAt)
}

// RecordAccess updates access metadata when this entry is accessed
func (cr *CachedResponse) RecordAccess() {
	cr.AccessCount++
}
