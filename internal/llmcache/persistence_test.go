package llmcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/gendocs/internal/llmtypes"
)

// TestDiskCache_BasicOperations tests basic Get/Put/Delete/Clear operations
func TestDiskCache_BasicOperations(t *testing.T) {
	t.Run("Put and Get", func(t *testing.T) {
		// Create temporary cache file
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024) // 100MB

		// Load the cache
		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		key := "test-key-1"
		value := &CachedResponse{
			Key: key,
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}

		// Put value
		if err := cache.Put(key, value); err != nil {
			t.Fatalf("Failed to put value: %v", err)
		}

		// Get value
		retrieved, found := cache.Get(key)
		if !found {
			t.Fatal("Expected to find value in cache")
		}

		if retrieved.Response.Content != "test response" {
			t.Errorf("Expected content 'test response', got '%s'", retrieved.Response.Content)
		}

		// Verify checksum was calculated
		if retrieved.Checksum == "" {
			t.Error("Expected checksum to be calculated")
		}

		// Cleanup
		cache.Stop()
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		_, found := cache.Get("non-existent")
		if found {
			t.Error("Expected not to find non-existent key")
		}

		cache.Stop()
	})

	t.Run("Delete key", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		key := "test-key-2"
		value := &CachedResponse{
			Key: key,
			Response: llmtypes.CompletionResponse{
				Content: "to be deleted",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}

		// Put value
		if err := cache.Put(key, value); err != nil {
			t.Fatalf("Failed to put value: %v", err)
		}

		// Delete value
		if err := cache.Delete(key); err != nil {
			t.Fatalf("Failed to delete value: %v", err)
		}

		// Verify deleted
		_, found := cache.Get(key)
		if found {
			t.Error("Expected key to be deleted")
		}

		cache.Stop()
	})

	t.Run("Clear all entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add multiple entries
		for i := 1; i <= 5; i++ {
			key := "test-key-" + string(rune('0'+i))
			value := &CachedResponse{
				Key: key,
				Response: llmtypes.CompletionResponse{
					Content: "test response",
				},
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now().Add(DefaultTTL),
				AccessCount: 0,
			}
			if err := cache.Put(key, value); err != nil {
				t.Fatalf("Failed to put value %d: %v", i, err)
			}
		}

		// Clear cache
		if err := cache.Clear(); err != nil {
			t.Fatalf("Failed to clear cache: %v", err)
		}

		// Verify all entries are gone
		stats := cache.Stats()
		if stats.TotalEntries != 0 {
			t.Errorf("Expected 0 entries after clear, got %d", stats.TotalEntries)
		}

		cache.Stop()
	})
}

// TestDiskCache_Persistence tests saving and loading cache data
func TestDiskCache_Persistence(t *testing.T) {
	t.Run("Save and load cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")

		// Create cache and add entries
		cache1 := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache1.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add test entries
		entry1 := &CachedResponse{
			Key: "key1",
			Response: llmtypes.CompletionResponse{
				Content: "response 1",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		entry2 := &CachedResponse{
			Key: "key2",
			Response: llmtypes.CompletionResponse{
				Content: "response 2",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}

		if err := cache1.Put("key1", entry1); err != nil {
			t.Fatalf("Failed to put entry1: %v", err)
		}
		if err := cache1.Put("key2", entry2); err != nil {
			t.Fatalf("Failed to put entry2: %v", err)
		}

		// Save cache
		if err := cache1.Save(); err != nil {
			t.Fatalf("Failed to save cache: %v", err)
		}
		cache1.Stop()

		// Load cache into new instance
		cache2 := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache2.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Verify entries were loaded
		stats := cache2.Stats()
		if stats.TotalEntries != 2 {
			t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
		}

		// Verify entry content
		retrieved1, found := cache2.Get("key1")
		if !found {
			t.Fatal("Expected to find key1")
		}
		if retrieved1.Response.Content != "response 1" {
			t.Errorf("Expected 'response 1', got '%s'", retrieved1.Response.Content)
		}

		retrieved2, found := cache2.Get("key2")
		if !found {
			t.Fatal("Expected to find key2")
		}
		if retrieved2.Response.Content != "response 2" {
			t.Errorf("Expected 'response 2', got '%s'", retrieved2.Response.Content)
		}

		cache2.Stop()
	})

	t.Run("Load creates new cache if file doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "nonexistent-cache.json")

		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load nonexistent cache: %v", err)
		}

		// Verify cache is initialized
		stats := cache.Stats()
		if stats.TotalEntries != 0 {
			t.Errorf("Expected 0 entries in new cache, got %d", stats.TotalEntries)
		}

		cache.Stop()
	})
}

// TestDiskCache_CorruptionHandling tests handling of corrupted cache files
func TestDiskCache_CorruptionHandling(t *testing.T) {
	t.Run("Corrupted JSON file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "corrupted-cache.json")

		// Create a corrupted JSON file
		corruptedContent := []byte("{invalid json content")
		if err := os.WriteFile(cachePath, corruptedContent, 0644); err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load corrupted cache: %v", err)
		}

		// Verify cache was reset to empty state
		stats := cache.Stats()
		if stats.TotalEntries != 0 {
			t.Errorf("Expected 0 entries after corrupted load, got %d", stats.TotalEntries)
		}

		// Verify backup was created
		backupFiles, err := filepath.Glob(cachePath + ".corrupted.*")
		if err != nil {
			t.Fatalf("Failed to glob backup files: %v", err)
		}
		if len(backupFiles) != 1 {
			t.Errorf("Expected 1 backup file, got %d", len(backupFiles))
		}

		cache.Stop()
	})

	t.Run("Corrupted entry checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")

		// Create cache with valid entries
		cache1 := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache1.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add entries
		entry1 := &CachedResponse{
			Key: "valid-key",
			Response: llmtypes.CompletionResponse{
				Content: "valid response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache1.Put("valid-key", entry1); err != nil {
			t.Fatalf("Failed to put valid entry: %v", err)
		}

		if err := cache1.Save(); err != nil {
			t.Fatalf("Failed to save cache: %v", err)
		}
		cache1.Stop()

		// Manually corrupt the cache file by modifying an entry's checksum
		data, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("Failed to read cache file: %v", err)
		}

		var cacheData DiskCacheData
		if err := json.Unmarshal(data, &cacheData); err != nil {
			t.Fatalf("Failed to unmarshal cache: %v", err)
		}

		// Add a corrupted entry with invalid checksum
		corruptedEntry := CachedResponse{
			Key: "corrupted-key",
			Response: llmtypes.CompletionResponse{
				Content: "corrupted response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
			Checksum:    "invalid-checksum-0123456789abcdef",
		}
		cacheData.Entries["corrupted-key"] = corruptedEntry

		// Save corrupted data
		corruptedJSON, _ := json.MarshalIndent(cacheData, "", "  ")
		if err := os.WriteFile(cachePath, corruptedJSON, 0644); err != nil {
			t.Fatalf("Failed to write corrupted cache: %v", err)
		}

		// Load corrupted cache
		cache2 := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache2.Load(); err != nil {
			t.Fatalf("Failed to load corrupted cache: %v", err)
		}

		// Verify corrupted entry was removed
		_, found := cache2.Get("corrupted-key")
		if found {
			t.Error("Expected corrupted entry to be removed")
		}

		// Verify valid entry is still present
		_, found = cache2.Get("valid-key")
		if !found {
			t.Error("Expected valid entry to still be present")
		}

		// Verify stats
		stats := cache2.Stats()
		if stats.TotalEntries != 1 {
			t.Errorf("Expected 1 valid entry, got %d", stats.TotalEntries)
		}

		cache2.Stop()
	})

	t.Run("Entry without checksum (backward compatibility)", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")

		// Create cache file with entry without checksum (old format)
		cacheData := DiskCacheData{
			Version:   CacheVersion,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Entries: map[string]CachedResponse{
				"old-key": {
					Key: "old-key",
					Response: llmtypes.CompletionResponse{
						Content: "old response",
					},
					CreatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(DefaultTTL),
					AccessCount: 0,
					Checksum:    "", // No checksum (old format)
				},
			},
			Stats: DiskCacheStats{},
		}

		data, _ := json.MarshalIndent(cacheData, "", "  ")
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			t.Fatalf("Failed to write old format cache: %v", err)
		}

		// Load old format cache
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load old format cache: %v", err)
		}

		// Verify entry is still accessible (backward compatible)
		retrieved, found := cache.Get("old-key")
		if !found {
			t.Fatal("Expected old format entry to be accessible")
		}
		if retrieved.Response.Content != "old response" {
			t.Errorf("Expected 'old response', got '%s'", retrieved.Response.Content)
		}

		cache.Stop()
	})
}

// TestDiskCache_VersionMismatch tests handling of version mismatches
func TestDiskCache_VersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test-cache.json")

	// Create cache file with wrong version
	cacheData := DiskCacheData{
		Version:   999, // Wrong version
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Entries: map[string]CachedResponse{
			"old-key": {
				Key: "old-key",
				Response: llmtypes.CompletionResponse{
					Content: "old response",
				},
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now().Add(DefaultTTL),
				AccessCount: 0,
			},
		},
		Stats: DiskCacheStats{},
	}

	data, _ := json.MarshalIndent(cacheData, "", "  ")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write cache: %v", err)
	}

	// Load cache with version mismatch
	cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
	if err := cache.Load(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify cache was reset to empty state
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after version mismatch, got %d", stats.TotalEntries)
	}

	cache.Stop()
}

// TestDiskCache_TTLExpiration tests TTL-based expiration
func TestDiskCache_TTLExpiration(t *testing.T) {
	t.Run("Expired entry returns not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, time.Hour, 100*1024*1024) // 1 hour TTL

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Create already-expired entry
		key := "expired-key"
		value := &CachedResponse{
			Key: key,
			Response: llmtypes.CompletionResponse{
				Content: "expired response",
			},
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   time.Now().Add(-1 * time.Hour),
			AccessCount: 0,
		}

		if err := cache.Put(key, value); err != nil {
			t.Fatalf("Failed to put expired entry: %v", err)
		}

		// Try to get expired entry
		_, found := cache.Get(key)
		if found {
			t.Error("Expected expired entry to not be found")
		}

		cache.Stop()
	})

	t.Run("Non-expired entry returns found", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		key := "valid-key"
		value := &CachedResponse{
			Key: key,
			Response: llmtypes.CompletionResponse{
				Content: "valid response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}

		if err := cache.Put(key, value); err != nil {
			t.Fatalf("Failed to put valid entry: %v", err)
		}

		// Get valid entry
		retrieved, found := cache.Get(key)
		if !found {
			t.Fatal("Expected valid entry to be found")
		}
		if retrieved.Response.Content != "valid response" {
			t.Errorf("Expected 'valid response', got '%s'", retrieved.Response.Content)
		}

		cache.Stop()
	})

	t.Run("CleanupExpired removes expired entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add expired entry
		expiredEntry := &CachedResponse{
			Key: "expired-key",
			Response: llmtypes.CompletionResponse{
				Content: "expired response",
			},
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   time.Now().Add(-1 * time.Hour),
			AccessCount: 0,
		}
		if err := cache.Put("expired-key", expiredEntry); err != nil {
			t.Fatalf("Failed to put expired entry: %v", err)
		}

		// Add valid entry
		validEntry := &CachedResponse{
			Key: "valid-key",
			Response: llmtypes.CompletionResponse{
				Content: "valid response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("valid-key", validEntry); err != nil {
			t.Fatalf("Failed to put valid entry: %v", err)
		}

		// Cleanup expired
		if err := cache.CleanupExpired(); err != nil {
			t.Fatalf("Failed to cleanup expired: %v", err)
		}

		// Verify expired entry was removed
		_, found := cache.Get("expired-key")
		if found {
			t.Error("Expected expired entry to be removed")
		}

		// Verify valid entry still exists
		_, found = cache.Get("valid-key")
		if !found {
			t.Error("Expected valid entry to still exist")
		}

		cache.Stop()
	})
}

// TestDiskCache_Statistics tests statistics tracking
func TestDiskCache_Statistics(t *testing.T) {
	t.Run("Initial stats", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		stats := cache.Stats()
		if stats.TotalEntries != 0 {
			t.Errorf("Expected 0 total entries, got %d", stats.TotalEntries)
		}
		if stats.ExpiredEntries != 0 {
			t.Errorf("Expected 0 expired entries, got %d", stats.ExpiredEntries)
		}
		if stats.Hits != 0 {
			t.Errorf("Expected 0 hits, got %d", stats.Hits)
		}
		if stats.Misses != 0 {
			t.Errorf("Expected 0 misses, got %d", stats.Misses)
		}

		cache.Stop()
	})

	t.Run("Hit and miss tracking", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add entry
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		// Hit
		cache.Get("test-key")
		stats := cache.Stats()
		if stats.Hits != 1 {
			t.Errorf("Expected 1 hit, got %d", stats.Hits)
		}

		// Miss
		cache.Get("non-existent")
		stats = cache.Stats()
		if stats.Misses != 1 {
			t.Errorf("Expected 1 miss, got %d", stats.Misses)
		}

		// Check hit rate
		expectedHitRate := 0.5 // 1 hit / 2 total lookups
		if stats.HitRate != expectedHitRate {
			t.Errorf("Expected hit rate %f, got %f", expectedHitRate, stats.HitRate)
		}

		cache.Stop()
	})

	t.Run("Eviction tracking", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add expired entries
		for i := 1; i <= 3; i++ {
			entry := &CachedResponse{
				Key: "expired-" + string(rune('0'+i)),
				Response: llmtypes.CompletionResponse{
					Content: "expired",
				},
				CreatedAt:   time.Now().Add(-2 * time.Hour),
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
				AccessCount: 0,
			}
			if err := cache.Put(entry.Key, entry); err != nil {
				t.Fatalf("Failed to put entry %d: %v", i, err)
			}
		}

		// Cleanup expired entries
		if err := cache.CleanupExpired(); err != nil {
			t.Fatalf("Failed to cleanup expired: %v", err)
		}

		stats := cache.Stats()
		if stats.Evictions != 3 {
			t.Errorf("Expected 3 evictions, got %d", stats.Evictions)
		}

		cache.Stop()
	})
}

// TestDiskCache_ConcurrentAccess tests thread-safety
func TestDiskCache_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent reads", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add entry
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		// Concurrent reads
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				cache.Get("test-key")
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify cache is still functional
		_, found := cache.Get("test-key")
		if !found {
			t.Error("Expected entry to still be found after concurrent reads")
		}

		cache.Stop()
	})

	t.Run("Concurrent writes", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Concurrent writes
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(i int) {
				key := "key-" + string(rune('0'+i))
				entry := &CachedResponse{
					Key: key,
					Response: llmtypes.CompletionResponse{
						Content: "response",
					},
					CreatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(DefaultTTL),
					AccessCount: 0,
				}
				cache.Put(key, entry)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all entries were added
		stats := cache.Stats()
		if stats.TotalEntries != 10 {
			t.Errorf("Expected 10 entries, got %d", stats.TotalEntries)
		}

		cache.Stop()
	})

	t.Run("Concurrent reads and writes", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		done := make(chan bool)
		// Start readers
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					cache.Get("key-1")
				}
				done <- true
			}()
		}
		// Start writers
		for i := 0; i < 5; i++ {
			go func(i int) {
				for j := 0; j < 10; j++ {
					key := "key-" + string(rune('0'+i))
					entry := &CachedResponse{
						Key: key,
						Response: llmtypes.CompletionResponse{
							Content: "response",
						},
						CreatedAt:   time.Now(),
						ExpiresAt:   time.Now().Add(DefaultTTL),
						AccessCount: 0,
					}
					cache.Put(key, entry)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify cache is still functional
		stats := cache.Stats()
		if stats.TotalEntries == 0 {
			t.Error("Expected entries to exist after concurrent operations")
		}

		cache.Stop()
	})
}

// TestDiskCache_AutoSave tests background auto-save functionality
func TestDiskCache_AutoSave(t *testing.T) {
	t.Run("Auto-save starts and stops", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Start auto-save with 100ms interval
		cache.StartAutoSave(100 * time.Millisecond)

		// Add entry
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		// Wait for auto-save to trigger
		time.Sleep(200 * time.Millisecond)

		// Stop auto-save (should do final save)
		cache.Stop()

		// Verify data was persisted by loading into new cache
		cache2 := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)
		if err := cache2.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		_, found := cache2.Get("test-key")
		if !found {
			t.Error("Expected entry to be persisted by auto-save")
		}

		cache2.Stop()
	})

	t.Run("Multiple StartAutoSave calls are idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Start auto-save multiple times
		cache.StartAutoSave(100 * time.Millisecond)
		cache.StartAutoSave(100 * time.Millisecond)
		cache.StartAutoSave(100 * time.Millisecond)

		// Add entry
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		// Wait and stop
		time.Sleep(200 * time.Millisecond)
		cache.Stop()

		// Verify cache is still functional
		_, found := cache.Get("test-key")
		if !found {
			t.Error("Expected entry to exist")
		}
	})

	t.Run("Stop without auto-save started is safe", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Stop without starting auto-save
		cache.Stop()

		// Verify cache is still functional
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		_, found := cache.Get("test-key")
		if !found {
			t.Error("Expected entry to exist")
		}
	})
}

// TestDiskCache_AtomicWrite tests atomic write pattern
func TestDiskCache_AtomicWrite(t *testing.T) {
	t.Run("Save uses atomic write pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test-cache.json")
		cache := NewDiskCache(cachePath, DefaultTTL, 100*1024*1024)

		if err := cache.Load(); err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Add entry
		entry := &CachedResponse{
			Key: "test-key",
			Response: llmtypes.CompletionResponse{
				Content: "test response",
			},
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(DefaultTTL),
			AccessCount: 0,
		}
		if err := cache.Put("test-key", entry); err != nil {
			t.Fatalf("Failed to put entry: %v", err)
		}

		// Save cache
		if err := cache.Save(); err != nil {
			t.Fatalf("Failed to save cache: %v", err)
		}

		// Verify temp file was cleaned up
		tmpFile := cachePath + ".tmp"
		if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
			t.Error("Expected temp file to be cleaned up after successful save")
		}

		// Verify main file exists
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("Expected cache file to exist after save")
		}

		cache.Stop()
	})
}
