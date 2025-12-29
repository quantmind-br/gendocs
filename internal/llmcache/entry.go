package llmcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/user/gendocs/internal/llm"
)

// NewCachedResponse creates a new CachedResponse with the given TTL
func NewCachedResponse(key string, request CacheKeyRequest, response llm.CompletionResponse, ttl time.Duration) *CachedResponse {
	now := time.Now()
	return &CachedResponse{
		Key:         key,
		Request:     request,
		Response:    response,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		SizeBytes:   0, // Will be estimated on first use
		AccessCount: 0,
	}
}

// CachedResponse represents a cached LLM response with metadata
type CachedResponse struct {
	Key         string                 `json:"key"`           // Cache key (SHA256 hash)
	Request     CacheKeyRequest        `json:"request"`       // Original request (for validation)
	Response    llm.CompletionResponse `json:"response"`      // LLM response
	CreatedAt   time.Time              `json:"created_at"`    // When cached
	ExpiresAt   time.Time              `json:"expires_at"`    // When entry expires
	SizeBytes   int64                  `json:"size_bytes"`    // Approximate size in memory
	AccessCount int                    `json:"access_count"`  // Number of times accessed
	Checksum    string                 `json:"checksum"`      // SHA256 checksum for data integrity
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

// CalculateChecksum computes the SHA256 checksum of the cached response data
// The checksum is computed over the response content (not metadata) to detect data corruption
func (cr *CachedResponse) CalculateChecksum() string {
	// Create a representation of the data to checksum
	// We include all fields that represent the actual cached data
	dataToHash := struct {
		Key      string                 `json:"key"`
		Request  CacheKeyRequest        `json:"request"`
		Response llm.CompletionResponse `json:"response"`
	}{
		Key:      cr.Key,
		Request:  cr.Request,
		Response: cr.Response,
	}

	// Serialize to JSON for consistent hashing
	jsonData, err := json.Marshal(dataToHash)
	if err != nil {
		// Fallback: hash the key if serialization fails
		hash := sha256.Sum256([]byte(cr.Key))
		return hex.EncodeToString(hash[:])
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// ValidateChecksum checks if the stored checksum matches the calculated checksum
// Returns true if the checksum is valid or if no checksum is stored (for backward compatibility)
func (cr *CachedResponse) ValidateChecksum() bool {
	// If no checksum is stored, consider it valid (backward compatibility)
	if cr.Checksum == "" {
		return true
	}

	calculatedChecksum := cr.CalculateChecksum()
	return cr.Checksum == calculatedChecksum
}

// UpdateChecksum recalculates and updates the checksum for this entry
// This should be called after modifying the entry
func (cr *CachedResponse) UpdateChecksum() {
	cr.Checksum = cr.CalculateChecksum()
}
