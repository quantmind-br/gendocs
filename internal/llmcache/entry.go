package llmcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/user/gendocs/internal/llm"
)

// CachedResponse represents a cached LLM response with metadata.
// It stores the original request, the LLM's response, and various metadata
// for cache management and validation.
type CachedResponse struct {
	Key         string                 `json:"key"`           // Cache key (SHA256 hash)
	Request     CacheKeyRequest        `json:"request"`       // Original request (for validation)
	Response    llm.CompletionResponse `json:"response"`      // LLM response content
	CreatedAt   time.Time              `json:"created_at"`    // Timestamp when the entry was cached
	ExpiresAt   time.Time              `json:"expires_at"`    // Timestamp when the entry expires
	SizeBytes   int64                  `json:"size_bytes"`    // Approximate size in memory (in bytes)
	AccessCount int                    `json:"access_count"`  // Number of times this entry has been accessed
	Checksum    string                 `json:"checksum"`      // SHA256 checksum for data integrity validation
}

// NewCachedResponse creates a new CachedResponse with the given TTL.
//
// The TTL (time-to-live) determines how long the cached response remains valid.
// After expiration, the cached response will not be used even if found in cache.
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

// EstimateSize calculates the approximate size of this entry in bytes.
//
// This is useful for cache management to track memory usage.
// If JSON marshaling fails, it returns a conservative 1KB estimate.
func (cr *CachedResponse) EstimateSize() int64 {
	// Serialize to JSON to estimate size
	data, err := json.Marshal(cr)
	if err != nil {
		// Fallback estimate if marshaling fails
		return 1024 // Assume 1KB
	}
	return int64(len(data))
}

// IsExpired checks if this cache entry has expired.
//
// Returns true if the current time is past the ExpiresAt timestamp,
// meaning the cached response should not be used.
func (cr *CachedResponse) IsExpired() bool {
	return time.Now().After(cr.ExpiresAt)
}

// RecordAccess updates access metadata when this entry is accessed.
//
// This increments the AccessCount counter, which can be useful
// for analytics and cache management decisions.
func (cr *CachedResponse) RecordAccess() {
	cr.AccessCount++
}

// CalculateChecksum computes the SHA256 checksum of the cached response data.
//
// The checksum is computed over the key, request, and response (not metadata)
// to detect data corruption. This is important for disk cache integrity validation.
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

// ValidateChecksum checks if the stored checksum matches the calculated checksum.
//
// Returns true if the checksum is valid or if no checksum is stored (for backward compatibility).
// This is used to detect data corruption in cached entries, particularly for disk cache.
func (cr *CachedResponse) ValidateChecksum() bool {
	// If no checksum is stored, consider it valid (backward compatibility)
	if cr.Checksum == "" {
		return true
	}

	calculatedChecksum := cr.CalculateChecksum()
	return cr.Checksum == calculatedChecksum
}

// UpdateChecksum recalculates and updates the checksum for this entry.
//
// This should be called after modifying the entry to ensure data integrity.
func (cr *CachedResponse) UpdateChecksum() {
	cr.Checksum = cr.CalculateChecksum()
}
