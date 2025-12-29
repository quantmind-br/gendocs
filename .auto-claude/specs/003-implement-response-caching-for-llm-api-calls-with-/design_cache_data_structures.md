# Cache Data Structures Design

## Overview
This document describes the design for in-memory and on-disk cache structures for LLM response caching, including TTL and size limit support.

## Architecture

### Two-Tier Cache System
The cache system uses a two-tier architecture:

```
┌─────────────────────────────────────┐
│   Application (LLM Client Layer)    │
└─────────────────┬───────────────────┘
                  │
                  v
┌─────────────────────────────────────┐
│     In-Memory LRU Cache (Fast)      │
│  - Hot entries                      │
│  - LRU eviction                     │
│  - Configurable size limit          │
└─────────────────┬───────────────────┘
                  │
                  v
┌─────────────────────────────────────┐
│    Persistent Disk Cache (Slow)     │
│  - All entries                      │
│  - TTL-based expiration             │
│  - JSON storage                     │
│  - Survives restarts                │
└─────────────────────────────────────┘
```

### Cache Flow
1. **Read**: Check in-memory cache → Check disk cache → Return cached value or miss
2. **Write**: Store in both in-memory cache and disk cache
3. **Eviction**: LRU eviction from memory, TTL cleanup from disk

## 1. In-Memory Cache Structure

### Data Structures

```go
// LRUCache implements a thread-safe LRU cache for LLM responses
type LRUCache struct {
    maxSize    int                       // Maximum number of entries
    size       int                       // Current number of entries
    cache      map[string]*lruEntry      // Hash map for O(1) lookups
    head, tail *lruEntry                // Doubly-linked list for LRU order
    mu         sync.RWMutex             // Read-write mutex for thread safety
    stats      CacheStats               // Cache statistics
}

// lruEntry represents a single cache entry in the LRU list
type lruEntry struct {
    key        string                   // Cache key (SHA256 hash)
    value      *CachedResponse          // Cached response data
    createdAt  time.Time                // When this entry was created
    accessedAt time.Time                // When this entry was last accessed
    sizeBytes  int64                    // Size of entry in bytes
    prev, next *lruEntry                // Doubly-linked list pointers
}

// CachedResponse represents a cached LLM response
type CachedResponse struct {
    Key         string                 `json:"key"`           // Cache key
    Request     CacheKeyRequest        `json:"request"`       // Original request (for validation)
    Response    CompletionResponse     `json:"response"`      // LLM response
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

// CacheStats tracks cache performance metrics
type CacheStats struct {
    Hits          int64         `json:"hits"`
    Misses        int64         `json:"misses"`
    Evictions     int64         `json:"evictions"`
    Size          int           `json:"size"`
    MaxSize       int           `json:"max_size"`
    TotalSizeBytes int64        `json:"total_size_bytes"`
    HitRate       float64       `json:"hit_rate"`
}
```

### LRU Operations

#### Get Operation
```go
func (c *LRUCache) Get(key string) (*CachedResponse, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    entry, exists := c.cache[key]
    if !exists {
        c.stats.Misses++
        c.updateHitRate()
        return nil, false
    }

    // Check TTL
    if time.Now().After(entry.value.ExpiresAt) {
        // Expired, remove from cache
        c.removeEntry(entry)
        c.stats.Misses++
        c.updateHitRate()
        return nil, false
    }

    // Move to front (most recently used)
    c.moveToFront(entry)

    // Update access time and count
    entry.value.AccessedAt = time.Now()
    entry.value.AccessCount++

    c.stats.Hits++
    c.updateHitRate()
    return entry.value, true
}
```

#### Put Operation
```go
func (c *LRUCache) Put(key string, value *CachedResponse) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Check if key already exists
    if entry, exists := c.cache[key]; exists {
        // Update existing entry
        entry.value = value
        entry.accessedAt = time.Now()
        c.moveToFront(entry)
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

    // Evict if over capacity
    for c.size > c.maxSize {
        c.evictLRU()
    }
}
```

#### Eviction Operation
```go
func (c *LRUCache) evictLRU() {
    if c.tail == nil {
        return
    }

    // Remove from map
    delete(c.cache, c.tail.key)

    // Update size tracking
    c.stats.TotalSizeBytes -= c.tail.sizeBytes
    c.size--
    c.stats.Size = c.size

    // Remove from list
    c.removeEntry(c.tail)

    c.stats.Evictions++
}
```

### Thread-Safety

**Strategy**: Use `sync.RWMutex` for read-write locking

- **Read operations (Get)**: Use `RLock()` for concurrent reads
- **Write operations (Put, Delete)**: Use `Lock()` for exclusive access
- **Statistics updates**: Protected by the same lock

**Rationale**:
- Cache reads are frequent (every LLM call)
- Multiple goroutines may access cache concurrently
- RWMutex allows concurrent reads without blocking
- Simpler than channel-based approach for this use case

## 2. On-Disk Cache Structure

### Data Structures

```go
// DiskCache manages persistent storage of cached responses
type DiskCache struct {
    filePath    string                 // Path to cache file
    ttl         time.Duration          // Default time-to-live
    maxDiskSize int64                  // Maximum disk size in bytes
    mu          sync.Mutex             // Mutex for file operations
    data        *DiskCacheData         // In-memory copy of disk data
    dirty       bool                   // Whether data needs to be saved
    autoSave    bool                   // Enable auto-save
    stopSave    chan struct{}          // Stop channel for background saver
}

// DiskCacheData represents the on-disk cache format
type DiskCacheData struct {
    Version   int                        `json:"version"`             // Cache format version
    CreatedAt time.Time                  `json:"created_at"`          // When cache was created
    UpdatedAt time.Time                  `json:"updated_at"`          // Last update time
    Entries   map[string]CachedResponse  `json:"entries"`             // All cached entries
    Stats     DiskCacheStats             `json:"stats"`               // Cache statistics
}

// DiskCacheStats tracks disk cache statistics
type DiskCacheStats struct {
    TotalEntries   int   `json:"total_entries"`
    ExpiredEntries int   `json:"expired_entries"`
    TotalSizeBytes int64 `json:"total_size_bytes"`
}
```

### File Format

**Location**: `.ai/llm_cache.json` (in project root)

**Format**: JSON (human-readable, diffable)

**Structure**:
```json
{
  "version": 1,
  "created_at": "2025-12-29T12:00:00Z",
  "updated_at": "2025-12-29T13:30:00Z",
  "entries": {
    "a1b2c3d4...": {
      "key": "a1b2c3d4...",
      "request": {
        "system_prompt": "You are a helpful assistant...",
        "messages": [...],
        "tools": [...],
        "temperature": 0.0
      },
      "response": {
        "content": "Here's the response...",
        "tool_calls": [],
        "usage": {
          "input_tokens": 100,
          "output_tokens": 50,
          "total_tokens": 150
        }
      },
      "created_at": "2025-12-29T12:00:00Z",
      "expires_at": "2026-01-05T12:00:00Z",
      "size_bytes": 1024,
      "access_count": 5
    }
  },
  "stats": {
    "total_entries": 100,
    "expired_entries": 10,
    "total_size_bytes": 102400
  }
}
```

### Operations

#### Load Operation
```go
func (dc *DiskCache) Load() error {
    dc.mu.Lock()
    defer dc.mu.Unlock()

    // Read file
    data, err := os.ReadFile(dc.filePath)
    if err != nil {
        if os.IsNotExist(err) {
            // New cache, create empty structure
            dc.data = dc.newCacheData()
            return nil
        }
        return fmt.Errorf("failed to read cache file: %w", err)
    }

    // Unmarshal JSON
    var cacheData DiskCacheData
    if err := json.Unmarshal(data, &cacheData); err != nil {
        // Corrupted cache, start fresh
        dc.data = dc.newCacheData()
        return nil
    }

    // Check version
    if cacheData.Version != CacheVersion {
        // Version mismatch, start fresh
        dc.data = dc.newCacheData()
        return nil
    }

    dc.data = &cacheData
    return nil
}
```

#### Save Operation
```go
func (dc *DiskCache) Save() error {
    dc.mu.Lock()
    defer dc.mu.Unlock()

    // Ensure directory exists
    dir := filepath.Dir(dc.filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create cache directory: %w", err)
    }

    // Update metadata
    dc.data.UpdatedAt = time.Now()
    dc.updateStats()

    // Marshal to JSON
    data, err := json.MarshalIndent(dc.data, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal cache: %w", err)
    }

    // Write to temporary file first (atomic write)
    tmpFile := dc.filePath + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return fmt.Errorf("failed to write cache: %w", err)
    }

    // Rename to actual file (atomic on Unix)
    if err := os.Rename(tmpFile, dc.filePath); err != nil {
        os.Remove(tmpFile) // Clean up temp file
        return fmt.Errorf("failed to save cache: %w", err)
    }

    dc.dirty = false
    return nil
}
```

#### Background Auto-Save
```go
func (dc *DiskCache) StartAutoSave(interval time.Duration) {
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
                    dc.mu.Unlock()

                    // Save asynchronously
                    if err := dc.saveData(&dataToSave); err != nil {
                        // Log error but continue
                    }
                } else {
                    dc.mu.Unlock()
                }
            case <-dc.stopSave:
                return
            }
        }
    }()
}
```

## 3. TTL Support

### Configuration

```go
// CacheConfig holds cache configuration
type CacheConfig struct {
    Enabled     bool          `mapstructure:"enabled"`      // Enable/disable caching
    MaxMemoryMB int           `mapstructure:"max_memory_mb"` // Max memory cache size in MB
    MaxDiskMB   int           `mapstructure:"max_disk_mb"`   // Max disk cache size in MB
    TTLDays     int           `mapstructure:"ttl_days"`      // Time-to-live in days
    CachePath   string        `mapstructure:"cache_path"`    // Path to cache file
    AutoSaveSec int           `mapstructure:"auto_save_sec"` // Auto-save interval in seconds
}

// GetTTL returns the TTL as a time.Duration
func (c *CacheConfig) GetTTL() time.Duration {
    if c.TTLDays == 0 {
        return 7 * 24 * time.Hour // Default: 7 days
    }
    return time.Duration(c.TTLDays) * 24 * time.Hour
}
```

### TTL Enforcement

#### In-Memory Cache
```go
func (c *LRUCache) cleanupExpired() {
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
}
```

#### Disk Cache
```go
func (dc *DiskCache) CleanupExpired() error {
    dc.mu.Lock()
    defer dc.mu.Unlock()

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
        if err := dc.Save(); err != nil {
            return err
        }
    }

    return nil
}
```

## 4. Size Limits

### Memory Size Limit

**Strategy**: Limit by entry count (simpler and more predictable)

```go
// Convert MB to entry count (estimated)
func MBToEntryCount(maxMB int) int {
    // Estimate average entry size: 5KB
    const avgEntrySizeKB = 5
    maxKB := maxMB * 1024
    return maxKB / avgEntrySizeKB
}

// Example: 100MB → ~20,000 entries
```

**Alternative**: Actual byte-size tracking
```go
func (c *LRUCache) Put(key string, value *CachedResponse) error {
    // Calculate size
    sizeBytes := calculateSize(value)

    // Check if adding would exceed limit
    if c.stats.TotalSizeBytes+sizeBytes > c.maxSizeBytes {
        // Evict entries until there's room
        for c.stats.TotalSizeBytes+sizeBytes > c.maxSizeBytes && c.size > 0 {
            c.evictLRU()
        }
    }

    // Add entry
    // ...
}
```

### Disk Size Limit

```go
func (dc *DiskCache) checkDiskSize() error {
    // Get current file size
    info, err := os.Stat(dc.filePath)
    if err != nil {
        return err
    }

    // If over limit, remove oldest entries
    if info.Size() > dc.maxDiskSize {
        dc.evictOldestEntries(float64(dc.maxDiskSize) * 0.8) // Evict to 80% of limit
    }

    return nil
}

func (dc *DiskCache) evictOldestEntries(targetSize float64) {
    // Sort entries by creation time (oldest first)
    entries := make([]*CachedResponse, 0, len(dc.data.Entries))
    for _, entry := range dc.data.Entries {
        entries = append(entries, &entry)
    }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].CreatedAt.Before(entries[j].CreatedAt)
    })

    // Remove oldest entries until under target size
    currentSize := dc.calculateTotalSize()
    idx := 0
    for currentSize > targetSize && idx < len(entries) {
        key := entries[idx].Key
        delete(dc.data.Entries, key)
        currentSize -= entries[idx].SizeBytes
        idx++
    }

    dc.dirty = true
}
```

## 5. Integration Layer

### Unified Cache Interface

```go
// Cache provides a unified cache interface
type Cache struct {
    memory     *LRUCache              // In-memory LRU cache
    disk       *DiskCache             // Persistent disk cache
    ttl        time.Duration          // Time-to-live
    mu         sync.RWMutex           // Protects cache operations
}

// NewCache creates a new two-tier cache
func NewCache(config CacheConfig) (*Cache, error) {
    // Create in-memory cache
    memoryCache := NewLRUCache(
        MBToEntryCount(config.MaxMemoryMB),
    )

    // Create disk cache
    diskCache, err := NewDiskCache(
        config.CachePath,
        config.GetTTL(),
        int64(config.MaxDiskMB)*1024*1024, // Convert MB to bytes
    )
    if err != nil {
        return nil, err
    }

    // Load existing disk cache
    if err := diskCache.Load(); err != nil {
        return nil, err
    }

    // Start auto-save
    if config.AutoSaveSec > 0 {
        diskCache.StartAutoSave(
            time.Duration(config.AutoSaveSec) * time.Second,
        )
    }

    return &Cache{
        memory: memoryCache,
        disk:   diskCache,
        ttl:    config.GetTTL(),
    }, nil
}
```

### Get Operation (Two-Tier)

```go
func (c *Cache) Get(key string) (*CachedResponse, bool) {
    // Check in-memory cache first
    if value, found := c.memory.Get(key); found {
        return value, true
    }

    // Check disk cache
    c.mu.RLock()
    value, found := c.disk.Get(key)
    c.mu.RUnlock()

    if !found {
        return nil, false
    }

    // Promote to memory cache
    c.memory.Put(key, value)

    return value, true
}
```

### Put Operation (Two-Tier)

```go
func (c *Cache) Put(key string, value *CachedResponse) error {
    // Set expiration
    value.ExpiresAt = time.Now().Add(c.ttl)

    // Store in both caches
    c.memory.Put(key, value)

    c.mu.Lock()
    defer c.mu.Unlock()

    return c.disk.Put(key, value)
}
```

## 6. Error Handling

### Corruption Recovery

```go
func (dc *DiskCache) Load() error {
    data, err := os.ReadFile(dc.filePath)
    if err != nil {
        if os.IsNotExist(err) {
            // First run, create empty cache
            dc.data = dc.newCacheData()
            return nil
        }
        return fmt.Errorf("failed to read cache: %w", err)
    }

    var cacheData DiskCacheData
    if err := json.Unmarshal(data, &cacheData); err != nil {
        // Cache corrupted - backup and create new
        dc.backupCorruptedCache()
        dc.data = dc.newCacheData()
        return nil
    }

    dc.data = &cacheData
    return nil
}

func (dc *DiskCache) backupCorruptedCache() {
    timestamp := time.Now().Format("20060102-150405")
    backupPath := dc.filePath + ".corrupted." + timestamp
    os.Rename(dc.filePath, backupPath)
}
```

### Write Failures

```go
func (c *Cache) Put(key string, value *CachedResponse) error {
    // Always succeed for memory cache (in-memory)
    c.memory.Put(key, value)

    // Best-effort for disk cache
    c.mu.Lock()
    defer c.mu.Unlock()

    if err := c.disk.Put(key, value); err != nil {
        // Log error but don't fail - memory cache still works
        // Cache will be missing after restart but not fatal
    }

    return nil
}
```

## 7. Performance Considerations

### In-Memory Cache

**Complexity**:
- Get: O(1) - hash map lookup
- Put: O(1) - hash map insert + linked list update
- Delete: O(1) - hash map delete + linked list remove

**Memory Overhead**:
- Per entry: ~64 bytes (pointers) + cached response
- For 10,000 entries: ~640KB overhead + response data

### Disk Cache

**Load Time**:
- Cold start: ~10-50ms for typical cache (100-500 entries)
- Incremental load: Not implemented (full load on startup)

**Save Time**:
- Small cache (<1000 entries): ~10-20ms
- Large cache (>10000 entries): ~100-500ms
- Mitigation: Background auto-save every 30-60 seconds

**File Size**:
- Typical entry: ~2-10KB (depends on response size)
- 1000 entries: ~2-10MB
- 10000 entries: ~20-100MB

### Optimization Strategies

1. **Lazy Loading**: Load disk cache entries into memory on-demand
2. **Write Coalescing**: Batch multiple puts before saving to disk
3. **Compression**: Compress large responses before storing
4. **Sharding**: Split disk cache into multiple files for very large caches

## 8. Configuration Examples

### Minimal Configuration (Development)
```yaml
cache:
  enabled: true
  max_memory_mb: 10    # ~2,000 entries
  max_disk_mb: 50      # ~5,000-10,000 entries
  ttl_days: 1          # Short TTL for development
  cache_path: ".ai/llm_cache.json"
  auto_save_sec: 60
```

### Typical Configuration (Production)
```yaml
cache:
  enabled: true
  max_memory_mb: 100   # ~20,000 entries
  max_disk_mb: 500     # ~50,000-250,000 entries
  ttl_days: 7          # 1 week TTL
  cache_path: ".ai/llm_cache.json"
  auto_save_sec: 30
```

### High-Performance Configuration
```yaml
cache:
  enabled: true
  max_memory_mb: 500   # ~100,000 entries
  max_disk_mb: 2000    # ~200,000-1,000,000 entries
  ttl_days: 30         # 1 month TTL
  cache_path: ".ai/llm_cache.json"
  auto_save_sec: 10
```

## 9. Testing Strategy

### Unit Tests
- LRU eviction behavior
- TTL expiration
- Concurrent access (thread-safety)
- Size limit enforcement
- Cache statistics accuracy

### Integration Tests
- Two-tier cache coordination
- Disk cache persistence
- Cache corruption recovery
- Performance benchmarks

### Manual Testing
- Real-world workload simulation
- Cache hit rate measurement
- Memory usage profiling
- Disk I/O impact assessment

## 10. Migration and Versioning

### Version Check
```go
const CacheVersion = 1

func (dc *DiskCache) Load() error {
    // ...
    if cacheData.Version != CacheVersion {
        // Version mismatch - migrate or clear
        return dc.migrateCache(&cacheData)
    }
    // ...
}
```

### Migration Strategy
- Version 0 → Version 1: Clear cache (breaking change)
- Version 1 → Version 2: Preserve entries, add new fields
- Always backup old cache before migration

## Summary

The cache data structures design provides:

✅ **Two-tier architecture** for fast access and persistence
✅ **LRU eviction** for memory management
✅ **TTL support** for cache freshness
✅ **Size limits** for resource control
✅ **Thread-safety** for concurrent access
✅ **Error handling** for robustness
✅ **Performance** with O(1) operations
✅ **Configurability** for different use cases

The design is ready for implementation in subtask 2-1 (Create LLM response cache package).
