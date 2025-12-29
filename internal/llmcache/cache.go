package llmcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fiowagd/gendocs/internal/logging"
)

const (
	// CacheVersion is the current cache format version
	CacheVersion = 1
	// DefaultCacheFileName is the default cache file name
	DefaultCacheFileName = ".ai/llm_cache.json"
	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL = 7 * 24 * time.Hour // 7 days
)

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits          int64    `json:"hits"`
	Misses        int64    `json:"misses"`
	Evictions     int64    `json:"evictions"`
	Size          int      `json:"size"`
	MaxSize       int      `json:"max_size"`
	TotalSizeBytes int64   `json:"total_size_bytes"`
	HitRate       float64  `json:"hit_rate"`
	mu            sync.RWMutex
}

// updateHitRate updates the hit rate calculation
func (s *CacheStats) updateHitRate() {
	total := s.Hits + s.Misses
	if total > 0 {
		s.HitRate = float64(s.Hits) / float64(total)
	}
}

// RecordHit records a cache hit
func (s *CacheStats) RecordHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits++
	s.updateHitRate()
}

// RecordMiss records a cache miss
func (s *CacheStats) RecordMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Misses++
	s.updateHitRate()
}

// RecordEviction records a cache eviction
func (s *CacheStats) RecordEviction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Evictions++
}

// lruEntry represents a single cache entry in the LRU list
type lruEntry struct {
	key        string
	value      *CachedResponse
	createdAt  time.Time
	accessedAt time.Time
	sizeBytes  int64
	prev, next *lruEntry
}

// LRUCache implements a thread-safe LRU cache for LLM responses
type LRUCache struct {
	maxSize    int
	size       int
	cache      map[string]*lruEntry
	head, tail *lruEntry
	mu         sync.RWMutex
	stats      CacheStats
	logger     *logging.Logger
}

// NewLRUCache creates a new LRU cache with the given maximum size
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		cache:   make(map[string]*lruEntry),
		stats:   CacheStats{MaxSize: maxSize},
		logger:  logging.NewNopLogger(),
	}
}

// SetLogger sets the logger for the cache
func (c *LRUCache) SetLogger(logger *logging.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (*CachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.cache[key]
	if !exists {
		c.stats.Misses++
		c.stats.updateHitRate()
		c.logger.Debug("cache_miss",
			logging.String("key", key),
			logging.Int64("total_misses", c.stats.Misses),
			logging.Float64("hit_rate", c.stats.HitRate))
		return nil, false
	}

	// Check TTL
	if time.Now().After(entry.value.ExpiresAt) {
		// Expired, remove from cache
		c.removeEntry(entry)
		c.stats.Misses++
		c.stats.updateHitRate()
		c.logger.Debug("cache_miss_expired",
			logging.String("key", key),
			logging.Time("expired_at", entry.value.ExpiresAt))
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(entry)

	// Update access time and count
	entry.accessedAt = time.Now()
	entry.value.RecordAccess()

	c.stats.Hits++
	c.stats.updateHitRate()
	c.logger.Debug("cache_hit",
		logging.String("key", key),
		logging.Int64("total_hits", c.stats.Hits),
		logging.Int("access_count", entry.value.AccessCount),
		logging.Float64("hit_rate", c.stats.HitRate))
	return entry.value, true
}

// Put stores a value in the cache
func (c *LRUCache) Put(key string, value *CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate size if not already set
	if value.SizeBytes == 0 {
		value.SizeBytes = value.EstimateSize()
	}

	// Check if key already exists
	if entry, exists := c.cache[key]; exists {
		// Update existing entry
		oldSize := entry.sizeBytes
		entry.value = value
		entry.accessedAt = time.Now()
		entry.sizeBytes = value.SizeBytes
		c.moveToFront(entry)
		c.stats.TotalSizeBytes += (entry.sizeBytes - oldSize)
		c.logger.Debug("cache_update",
			logging.String("key", key),
			logging.Int64("old_size_bytes", oldSize),
			logging.Int64("new_size_bytes", entry.sizeBytes))
		return
	}

	// Create new entry
	entry := &lruEntry{
		key:        key,
		value:      value,
		createdAt:  time.Now(),
		accessedAt: time.Now(),
		sizeBytes:  value.SizeBytes,
	}

	// Add to cache
	c.cache[key] = entry
	c.addToFront(entry)
	c.size++
	c.stats.Size = c.size
	c.stats.TotalSizeBytes += entry.sizeBytes

	c.logger.Debug("cache_store",
		logging.String("key", key),
		logging.Int64("size_bytes", entry.sizeBytes),
		logging.Int("current_size", c.size),
		logging.Int("max_size", c.maxSize))

	// Evict if over capacity
	for c.size > c.maxSize {
		c.evictLRU()
	}
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.cache[key]; exists {
		c.removeEntry(entry)
	}
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*lruEntry)
	c.head = nil
	c.tail = nil
	c.size = 0
	c.stats.Size = 0
	c.stats.TotalSizeBytes = 0
}

// Size returns the current number of entries in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Stats returns a copy of the cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := c.stats
	return stats
}

// moveToFront moves an entry to the front of the LRU list
func (c *LRUCache) moveToFront(entry *lruEntry) {
	if entry == c.head {
		return
	}

	// Remove from current position
	c.removeEntryList(entry)

	// Add to front
	c.addToFront(entry)
}

// addToFront adds an entry to the front of the LRU list
func (c *LRUCache) addToFront(entry *lruEntry) {
	entry.prev = nil
	entry.next = c.head

	if c.head != nil {
		c.head.prev = entry
	}

	c.head = entry

	if c.tail == nil {
		c.tail = entry
	}
}

// removeEntry removes an entry from the cache and LRU list
func (c *LRUCache) removeEntry(entry *lruEntry) {
	// Remove from map
	delete(c.cache, entry.key)

	// Update size tracking
	c.stats.TotalSizeBytes -= entry.sizeBytes
	c.size--
	c.stats.Size = c.size

	// Remove from list
	c.removeEntryList(entry)
}

// removeEntryList removes an entry from the LRU list only
func (c *LRUCache) removeEntryList(entry *lruEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		c.head = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		c.tail = entry.prev
	}
}

// evictLRU evicts the least recently used entry
func (c *LRUCache) evictLRU() {
	if c.tail == nil {
		return
	}

	evictedKey := c.tail.key
	evictedSize := c.tail.sizeBytes

	// Remove from map
	delete(c.cache, c.tail.key)

	// Update size tracking
	c.stats.TotalSizeBytes -= c.tail.sizeBytes
	c.size--
	c.stats.Size = c.size

	// Remove from list
	if c.tail.prev != nil {
		c.tail.prev.next = nil
		c.tail = c.tail.prev
	} else {
		// Cache is now empty
		c.head = nil
		c.tail = nil
	}

	c.stats.Evictions++
	c.logger.Debug("cache_evict",
		logging.String("key", evictedKey),
		logging.Int64("size_bytes", evictedSize),
		logging.Int64("total_evictions", c.stats.Evictions),
		logging.Int("current_size", c.size))
}

// CleanupExpired removes all expired entries from the cache
func (c *LRUCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := []*lruEntry{}

	// Find expired entries
	for _, entry := range c.cache {
		if now.After(entry.value.ExpiresAt) {
			expired = append(expired, entry)
		}
	}

	// Remove expired entries
	for _, entry := range expired {
		c.removeEntry(entry)
		c.stats.Evictions++
	}

	if len(expired) > 0 {
		c.logger.Info("cache_cleanup_expired",
			logging.Int("expired_count", len(expired)),
			logging.Int("remaining_size", c.size))
	}

	return len(expired)
}

// DiskCacheData represents the on-disk cache format
type DiskCacheData struct {
	Version   int                         `json:"version"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
	Entries   map[string]CachedResponse   `json:"entries"`
	Stats     DiskCacheStats              `json:"stats"`
	mu        sync.RWMutex                // Protects stats fields
}

// DiskCacheStats tracks disk cache statistics
type DiskCacheStats struct {
	TotalEntries   int   `json:"total_entries"`
	ExpiredEntries int   `json:"expired_entries"`
	TotalSizeBytes int64 `json:"total_size_bytes"`
	Hits           int64 `json:"hits"`
	Misses         int64 `json:"misses"`
	Evictions      int64 `json:"evictions"`
	HitRate        float64 `json:"hit_rate"`
}

// updateHitRate updates the hit rate calculation for disk cache stats
func (s *DiskCacheStats) updateHitRate() {
	total := s.Hits + s.Misses
	if total > 0 {
		s.HitRate = float64(s.Hits) / float64(total)
	}
}

// DiskCache manages persistent storage of cached responses
type DiskCache struct {
	filePath    string
	ttl         time.Duration
	maxDiskSize int64
	mu          sync.Mutex
	data        *DiskCacheData
	dirty       bool
	autoSave    bool
	stopSave    chan struct{}
	logger      *logging.Logger
}

// NewDiskCache creates a new disk cache
func NewDiskCache(filePath string, ttl time.Duration, maxDiskSize int64) *DiskCache {
	return &DiskCache{
		filePath:    filePath,
		ttl:         ttl,
		maxDiskSize: maxDiskSize,
		logger:      logging.NewNopLogger(),
	}
}

// SetLogger sets the logger for the disk cache
func (dc *DiskCache) SetLogger(logger *logging.Logger) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.logger = logger
}

// Load loads the cache from disk
func (dc *DiskCache) Load() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Read file
	data, err := os.ReadFile(dc.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// New cache, create empty structure
			dc.data = dc.newCacheData()
			dc.logger.Info("disk_cache_load", logging.String("status", "new_cache"))
			return nil
		}
		dc.logger.Error("disk_cache_load_failed", logging.Error(err))
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Unmarshal JSON
	var cacheData DiskCacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		// Corrupted cache, backup and start fresh
		dc.backupCorruptedCache()
		dc.data = dc.newCacheData()
		dc.logger.Warn("disk_cache_corrupted", logging.String("action", "backup_and_reset"))
		return nil
	}

	// Check version
	if cacheData.Version != CacheVersion {
		// Version mismatch, start fresh
		dc.data = dc.newCacheData()
		dc.logger.Warn("disk_cache_version_mismatch",
			logging.Int("loaded_version", cacheData.Version),
			logging.Int("expected_version", CacheVersion),
			logging.String("action", "reset"))
		return nil
	}

	// Validate entry checksums and remove corrupted entries
	corruptedCount := 0
	for key, entry := range cacheData.Entries {
		if !entry.ValidateChecksum() {
			// Checksum validation failed, remove corrupted entry
			delete(cacheData.Entries, key)
			corruptedCount++
			dc.logger.Warn("disk_cache_corrupted_entry",
				logging.String("key", key),
				logging.String("action", "removed"))
		}
	}

	if corruptedCount > 0 {
		dc.logger.Info("disk_cache_validation",
			logging.Int("corrupted_entries", corruptedCount),
			logging.Int("valid_entries", len(cacheData.Entries)),
			logging.String("file_path", dc.filePath))
	}

	dc.data = &cacheData
	dc.logger.Info("disk_cache_load",
		logging.String("status", "success"),
		logging.Int("entries", len(dc.data.Entries)),
		logging.String("file_path", dc.filePath))
	return nil
}

// Save saves the cache to disk
func (dc *DiskCache) Save() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return dc.saveLocked()
}

// saveLocked saves the cache to disk (must be called with lock held)
func (dc *DiskCache) saveLocked() error {
	// Ensure directory exists
	dir := filepath.Dir(dc.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		dc.logger.Error("disk_cache_save_failed", logging.Error(err))
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Update metadata
	dc.data.UpdatedAt = time.Now()
	dc.updateStats()

	// Marshal to JSON
	data, err := json.MarshalIndent(dc.data, "", "  ")
	if err != nil {
		dc.logger.Error("disk_cache_save_failed", logging.Error(err))
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to temporary file first (atomic write)
	tmpFile := dc.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		dc.logger.Error("disk_cache_save_failed", logging.Error(err))
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Rename to actual file (atomic on Unix)
	if err := os.Rename(tmpFile, dc.filePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		dc.logger.Error("disk_cache_save_failed", logging.Error(err))
		return fmt.Errorf("failed to save cache: %w", err)
	}

	dc.dirty = false
	dc.logger.Debug("disk_cache_save",
		logging.Int("entries", len(dc.data.Entries)),
		logging.Int("total_size_bytes", dc.data.Stats.TotalSizeBytes),
		logging.String("file_path", dc.filePath))
	return nil
}

// Get retrieves a value from the disk cache
func (dc *DiskCache) Get(key string) (*CachedResponse, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.data == nil {
		dc.recordMiss()
		dc.logger.Debug("disk_cache_miss", logging.String("reason", "cache_not_loaded"))
		return nil, false
	}

	entry, exists := dc.data.Entries[key]
	if !exists {
		dc.recordMiss()
		dc.logger.Debug("disk_cache_miss",
			logging.String("key", key),
			logging.Int64("total_misses", dc.data.Stats.Misses),
			logging.Float64("hit_rate", dc.data.Stats.HitRate))
		return nil, false
	}

	// Check TTL
	if entry.IsExpired() {
		delete(dc.data.Entries, key)
		dc.dirty = true
		dc.recordMiss()
		dc.logger.Debug("disk_cache_miss_expired",
			logging.String("key", key),
			logging.Time("expired_at", entry.ExpiresAt))
		return nil, false
	}

	// Record hit
	dc.recordHit()

	dc.logger.Debug("disk_cache_hit",
		logging.String("key", key),
		logging.Int64("total_hits", dc.data.Stats.Hits),
		logging.Float64("hit_rate", dc.data.Stats.HitRate))

	// Return a copy to avoid race conditions
	result := entry
	return &result, true
}

// Put stores a value in the disk cache
func (dc *DiskCache) Put(key string, value *CachedResponse) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.data == nil {
		dc.data = dc.newCacheData()
	}

	// Ensure checksum is calculated and up-to-date before storing
	value.UpdateChecksum()

	isNew := !dc.data.Entries[key].IsExpired()
	dc.data.Entries[key] = *value
	dc.dirty = true

	if isNew {
		dc.logger.Debug("disk_cache_store",
			logging.String("key", key),
			logging.Int64("size_bytes", value.SizeBytes),
			logging.Int("total_entries", len(dc.data.Entries)))
	} else {
		dc.logger.Debug("disk_cache_update",
			logging.String("key", key),
			logging.Int64("size_bytes", value.SizeBytes))
	}

	return nil
}

// recordHit records a disk cache hit
func (dc *DiskCache) recordHit() {
	if dc.data == nil {
		return
	}
	dc.data.mu.Lock()
	defer dc.data.mu.Unlock()
	dc.data.Stats.Hits++
	dc.data.Stats.updateHitRate()
}

// recordMiss records a disk cache miss
func (dc *DiskCache) recordMiss() {
	if dc.data == nil {
		return
	}
	dc.data.mu.Lock()
	defer dc.data.mu.Unlock()
	dc.data.Stats.Misses++
	dc.data.Stats.updateHitRate()
}

// recordEviction records a disk cache eviction
func (dc *DiskCache) recordEviction(count int) {
	if dc.data == nil {
		return
	}
	dc.data.mu.Lock()
	defer dc.data.mu.Unlock()
	dc.data.Stats.Evictions += int64(count)
}

// Stats returns a copy of the disk cache statistics
func (dc *DiskCache) Stats() DiskCacheStats {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.data == nil {
		return DiskCacheStats{}
	}

	dc.data.mu.RLock()
	defer dc.data.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := dc.data.Stats
	return stats
}

// Delete removes a value from the disk cache
func (dc *DiskCache) Delete(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.data == nil {
		return nil
	}

	if _, exists := dc.data.Entries[key]; exists {
		delete(dc.data.Entries, key)
		dc.dirty = true
	}

	return nil
}

// Clear removes all entries from the disk cache
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.data = dc.newCacheData()
	dc.dirty = true

	return dc.saveLocked()
}

// CleanupExpired removes expired entries from the disk cache
func (dc *DiskCache) CleanupExpired() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.data == nil {
		return nil
	}

	now := time.Now()
	expiredCount := 0

	// Remove expired entries
	for key, entry := range dc.data.Entries {
		if now.After(entry.ExpiresAt) {
			delete(dc.data.Entries, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		dc.dirty = true
		dc.recordEviction(expiredCount)
		dc.logger.Info("disk_cache_cleanup_expired",
			logging.Int("expired_count", expiredCount),
			logging.Int("remaining_entries", len(dc.data.Entries)),
			logging.String("file_path", dc.filePath))
		return dc.saveLocked()
	}

	return nil
}

// newCacheData creates a new empty cache data structure
func (dc *DiskCache) newCacheData() *DiskCacheData {
	return &DiskCacheData{
		Version:   CacheVersion,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Entries:   make(map[string]CachedResponse),
	}
}

// updateStats updates the disk cache statistics
func (dc *DiskCache) updateStats() {
	if dc.data == nil {
		return
	}

	now := time.Now()
	expiredCount := 0
	totalSize := int64(0)

	for _, entry := range dc.data.Entries {
		if now.After(entry.ExpiresAt) {
			expiredCount++
		}
		totalSize += entry.SizeBytes
	}

	dc.data.Stats = DiskCacheStats{
		TotalEntries:   len(dc.data.Entries),
		ExpiredEntries: expiredCount,
		TotalSizeBytes: totalSize,
	}
}

// backupCorruptedCache backs up a corrupted cache file
func (dc *DiskCache) backupCorruptedCache() {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := dc.filePath + ".corrupted." + timestamp
	os.Rename(dc.filePath, backupPath)
}

// StartAutoSave starts background auto-save with the given interval
func (dc *DiskCache) StartAutoSave(interval time.Duration) {
	dc.mu.Lock()
	if dc.autoSave {
		// Already started
		dc.mu.Unlock()
		return
	}
	dc.autoSave = true
	dc.mu.Unlock()

	dc.stopSave = make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dc.mu.Lock()
				if dc.dirty {
					// Save without holding lock (copy data first)
					dataToSave := *dc.data
					dirtyFlag := dc.dirty
					dc.mu.Unlock()

					// Save asynchronously
					if dirtyFlag {
						if err := dc.saveData(&dataToSave); err != nil {
							// Log error but continue - non-blocking
							// TODO: Add logging in subtask 4-2
						}
					}
				} else {
					dc.mu.Unlock()
				}
			case <-dc.stopSave:
				// Stop signal received, do one final save if dirty
				dc.mu.Lock()
				if dc.dirty {
					dataToSave := *dc.data
					dc.mu.Unlock()
					dc.saveData(&dataToSave)
				} else {
					dc.mu.Unlock()
				}
				return
			}
		}
	}()
}

// Stop stops the disk cache and performs final save if needed
func (dc *DiskCache) Stop() {
	dc.mu.Lock()
	if !dc.autoSave {
		dc.mu.Unlock()
		return
	}
	dc.autoSave = false
	dc.mu.Unlock()

	// Signal the background goroutine to stop
	if dc.stopSave != nil {
		close(dc.stopSave)
		dc.stopSave = nil
	}
}

// saveData saves the cache data to disk without holding the lock
func (dc *DiskCache) saveData(data *DiskCacheData) error {
	// Ensure directory exists
	dir := filepath.Dir(dc.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Update metadata
	data.UpdatedAt = time.Now()

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to temporary file first (atomic write)
	tmpFile := dc.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Rename to actual file (atomic on Unix)
	if err := os.Rename(tmpFile, dc.filePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to save cache: %w", err)
	}

	// Clear dirty flag after successful save
	dc.mu.Lock()
	dc.dirty = false
	dc.mu.Unlock()

	return nil
}
