package llmcache

import (
	"sync"
	"testing"
	"time"

	"github.com/user/gendocs/internal/llm"
)

// TestLRUCache_BasicOperations tests basic Get/Put/Delete operations
func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(10)

	t.Run("Put and Get", func(t *testing.T) {
		key := "test-key-1"
		value := &CachedResponse{
			Key:   key,
			Response: llm.CompletionResponse{
				Content: "test response",
			},
		}

		// Put value
		cache.Put(key, value)

		// Get value
		retrieved, found := cache.Get(key)
		if !found {
			t.Fatal("Expected to find value in cache")
		}

		if retrieved.Content != "test response" {
			t.Errorf("Expected content 'test response', got '%s'", retrieved.Content)
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, found := cache.Get("non-existent")
		if found {
			t.Error("Expected not to find non-existent key")
		}
	})

	t.Run("Delete key", func(t *testing.T) {
		key := "test-key-2"
		value := &CachedResponse{
			Key:   key,
			Response: llm.CompletionResponse{
				Content: "to be deleted",
			},
		}

		cache.Put(key, value)
		cache.Delete(key)

		_, found := cache.Get(key)
		if found {
			t.Error("Expected deleted key to not be found")
		}
	})

	t.Run("Update existing key", func(t *testing.T) {
		key := "test-key-3"
		value1 := &CachedResponse{
			Key:   key,
			Response: llm.CompletionResponse{
				Content: "first value",
			},
		}
		value2 := &CachedResponse{
			Key:   key,
			Response: llm.CompletionResponse{
				Content: "second value",
			},
		}

		cache.Put(key, value1)
		cache.Put(key, value2)

		retrieved, _ := cache.Get(key)
		if retrieved.Content != "second value" {
			t.Errorf("Expected 'second value', got '%s'", retrieved.Content)
		}

		// Size should still be 1 (not 2)
		if cache.Size() != 4 { // 3 from previous tests + 1 update
			t.Errorf("Expected cache size 4, got %d", cache.Size())
		}
	})
}

// TestLRUCache_LRUEviction tests that LRU eviction works correctly
func TestLRUCache_LRUEviction(t *testing.T) {
	t.Run("evict when exceeding maxSize", func(t *testing.T) {
		cache := NewLRUCache(3)

		// Fill cache to max
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})
		cache.Put("key2", &CachedResponse{Response: llm.CompletionResponse{Content: "value2"}})
		cache.Put("key3", &CachedResponse{Response: llm.CompletionResponse{Content: "value3"}})

		if cache.Size() != 3 {
			t.Errorf("Expected cache size 3, got %d", cache.Size())
		}

		// Add one more - should evict key1 (least recently used)
		cache.Put("key4", &CachedResponse{Response: llm.CompletionResponse{Content: "value4"}})

		if cache.Size() != 3 {
			t.Errorf("Expected cache size 3 after eviction, got %d", cache.Size())
		}

		// key1 should be evicted
		_, found := cache.Get("key1")
		if found {
			t.Error("Expected key1 to be evicted")
		}

		// key2, key3, key4 should still exist
		for _, key := range []string{"key2", "key3", "key4"} {
			_, found := cache.Get(key)
			if !found {
				t.Errorf("Expected %s to still exist in cache", key)
			}
		}
	})

	t.Run("eviction respects access order", func(t *testing.T) {
		cache := NewLRUCache(3)

		// Fill cache
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})
		cache.Put("key2", &CachedResponse{Response: llm.CompletionResponse{Content: "value2"}})
		cache.Put("key3", &CachedResponse{Response: llm.CompletionResponse{Content: "value3"}})

		// Access key1 to make it more recently used
		cache.Get("key1")

		// Add key4 - should evict key2 (now least recently used)
		cache.Put("key4", &CachedResponse{Response: llm.CompletionResponse{Content: "value4"}})

		// key1 should still exist (was accessed)
		_, found := cache.Get("key1")
		if !found {
			t.Error("Expected key1 to still exist after being accessed")
		}

		// key2 should be evicted
		_, found = cache.Get("key2")
		if found {
			t.Error("Expected key2 to be evicted")
		}

		// key3 and key4 should exist
		for _, key := range []string{"key3", "key4"} {
			_, found := cache.Get(key)
			if !found {
				t.Errorf("Expected %s to still exist", key)
			}
		}
	})

	t.Run("update makes entry recently used", func(t *testing.T) {
		cache := NewLRUCache(3)

		// Fill cache
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})
		cache.Put("key2", &CachedResponse{Response: llm.CompletionResponse{Content: "value2"}})
		cache.Put("key3", &CachedResponse{Response: llm.CompletionResponse{Content: "value3"}})

		// Update key1 to make it recently used
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1-updated"}})

		// Add key4 - should evict key2 (least recently used)
		cache.Put("key4", &CachedResponse{Response: llm.CompletionResponse{Content: "value4"}})

		// key1 should still exist (was updated)
		retrieved, found := cache.Get("key1")
		if !found {
			t.Error("Expected key1 to still exist after being updated")
		}
		if retrieved.Content != "value1-updated" {
			t.Errorf("Expected updated value for key1, got '%s'", retrieved.Content)
		}

		// key2 should be evicted
		_, found = cache.Get("key2")
		if found {
			t.Error("Expected key2 to be evicted")
		}
	})

	t.Run("multiple evictions", func(t *testing.T) {
		cache := NewLRUCache(3)

		// Fill cache
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})
		cache.Put("key2", &CachedResponse{Response: llm.CompletionResponse{Content: "value2"}})
		cache.Put("key3", &CachedResponse{Response: llm.CompletionResponse{Content: "value3"}})

		// Add 3 more entries - should evict key1, key2, key3 in that order
		for i := 4; i <= 6; i++ {
			key := string(rune('0' + i))
			cache.Put("key"+key, &CachedResponse{Response: llm.CompletionResponse{Content: "value" + key}})
		}

		if cache.Size() != 3 {
			t.Errorf("Expected cache size 3, got %d", cache.Size())
		}

		// Only key4, key5, key6 should exist
		expectedKeys := []string{"key4", "key5", "key6"}
		for _, key := range expectedKeys {
			_, found := cache.Get(key)
			if !found {
				t.Errorf("Expected %s to exist", key)
			}
		}

		// key1, key2, key3 should be evicted
		evictedKeys := []string{"key1", "key2", "key3"}
		for _, key := range evictedKeys {
			_, found := cache.Get(key)
			if found {
				t.Errorf("Expected %s to be evicted", key)
			}
		}
	})
}

// TestLRUCache_ConcurrentAccess tests thread-safety under concurrent access
func TestLRUCache_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent reads", func(t *testing.T) {
		cache := NewLRUCache(100)

		// Pre-fill cache
		for i := 0; i < 50; i++ {
			key := string(rune('0' + i))
			cache.Put("key"+key, &CachedResponse{
				Response: llm.CompletionResponse{Content: "value" + key},
			})
		}

		// Concurrent reads
		var wg sync.WaitGroup
		numGoroutines := 10
		readsPerGoroutine := 100

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < readsPerGoroutine; i++ {
					key := "key" + string(rune('0'+(i%50)))
					cache.Get(key)
				}
			}(g)
		}

		wg.Wait()

		// Verify cache integrity
		stats := cache.Stats()
		if stats.Hits+stats.Misses != int64(numGoroutines*readsPerGoroutine) {
			t.Errorf("Expected %d total lookups, got %d", numGoroutines*readsPerGoroutine, stats.Hits+stats.Misses)
		}
	})

	t.Run("concurrent writes", func(t *testing.T) {
		cache := NewLRUCache(1000)

		var wg sync.WaitGroup
		numGoroutines := 10
		writesPerGoroutine := 100

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < writesPerGoroutine; i++ {
					key := string(rune('a' + i))
					cache.Put("key"+string(rune('0'+goroutineID))+key, &CachedResponse{
						Response: llm.CompletionResponse{Content: "value"},
					})
				}
			}(g)
		}

		wg.Wait()

		// All writes should succeed
		expectedSize := numGoroutines * writesPerGoroutine
		if cache.Size() != expectedSize {
			t.Errorf("Expected cache size %d, got %d", expectedSize, cache.Size())
		}
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		cache := NewLRUCache(500)

		var wg sync.WaitGroup
		numGoroutines := 10
		operationsPerGoroutine := 100

		for g := 0; g < numGoroutines; g++ {
			wg.Add(2) // One reader, one writer per goroutine

			// Reader
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < operationsPerGoroutine; i++ {
					key := "key" + string(rune('0'+(i%100)))
					cache.Get(key)
				}
			}(g)

			// Writer
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < operationsPerGoroutine; i++ {
					key := string(rune('a' + i))
					cache.Put("key"+string(rune('0'+goroutineID))+key, &CachedResponse{
						Response: llm.CompletionResponse{Content: "value"},
					})
				}
			}(g)
		}

		wg.Wait()

		// Verify cache integrity - should not panic or deadlock
		stats := cache.Stats()
		if stats.Hits+stats.Misses == 0 {
			t.Error("Expected some cache operations to occur")
		}
	})

	t.Run("concurrent updates to same key", func(t *testing.T) {
		cache := NewLRUCache(100)

		key := "shared-key"
		cache.Put(key, &CachedResponse{Response: llm.CompletionResponse{Content: "initial"}})

		var wg sync.WaitGroup
		numGoroutines := 10
		updatesPerGoroutine := 50

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < updatesPerGoroutine; i++ {
					cache.Put(key, &CachedResponse{
						Response: llm.CompletionResponse{Content: "value"},
					})
				}
			}(g)
		}

		wg.Wait()

		// Key should still exist
		_, found := cache.Get(key)
		if !found {
			t.Error("Expected shared key to still exist after concurrent updates")
		}

		// Size should be 1 (all updates to same key)
		if cache.Size() != 1 {
			t.Errorf("Expected cache size 1, got %d", cache.Size())
		}
	})

	t.Run("concurrent deletions", func(t *testing.T) {
		cache := NewLRUCache(1000)

		// Pre-fill cache
		for i := 0; i < 100; i++ {
			key := "key" + string(rune('0'+i))
			cache.Put(key, &CachedResponse{Response: llm.CompletionResponse{Content: "value"}})
		}

		var wg sync.WaitGroup
		numGoroutines := 10

		// Start concurrent deletions
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					key := "key" + string(rune('0'+(i+goroutineID*10)))
					cache.Delete(key)
				}
			}(g)
		}

		wg.Wait()

		// All entries should be deleted
		if cache.Size() != 0 {
			t.Errorf("Expected cache size 0, got %d", cache.Size())
		}
	})
}

// TestLRUCache_Stats tests statistics tracking
func TestLRUCache_Stats(t *testing.T) {
	cache := NewLRUCache(10)

	t.Run("initial stats", func(t *testing.T) {
		stats := cache.Stats()

		if stats.Size != 0 {
			t.Errorf("Expected initial size 0, got %d", stats.Size)
		}

		if stats.Hits != 0 {
			t.Errorf("Expected initial hits 0, got %d", stats.Hits)
		}

		if stats.Misses != 0 {
			t.Errorf("Expected initial misses 0, got %d", stats.Misses)
		}

		if stats.Evictions != 0 {
			t.Errorf("Expected initial evictions 0, got %d", stats.Evictions)
		}
	})

	t.Run("hit and miss tracking", func(t *testing.T) {
		cache := NewLRUCache(10)
		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})

		// Hit
		cache.Get("key1")

		// Miss
		cache.Get("non-existent")

		stats := cache.Stats()

		if stats.Hits != 1 {
			t.Errorf("Expected 1 hit, got %d", stats.Hits)
		}

		if stats.Misses != 1 {
			t.Errorf("Expected 1 miss, got %d", stats.Misses)
		}

		expectedHitRate := 1.0 / 2.0
		if stats.HitRate != expectedHitRate {
			t.Errorf("Expected hit rate %f, got %f", expectedHitRate, stats.HitRate)
		}
	})

	t.Run("eviction tracking", func(t *testing.T) {
		cache := NewLRUCache(2)

		cache.Put("key1", &CachedResponse{Response: llm.CompletionResponse{Content: "value1"}})
		cache.Put("key2", &CachedResponse{Response: llm.CompletionResponse{Content: "value2"}})
		cache.Put("key3", &CachedResponse{Response: llm.CompletionResponse{Content: "value3"}})

		stats := cache.Stats()

		if stats.Evictions != 1 {
			t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
		}
	})
}

// TestLRUCache_SizeLimit tests that size limits are enforced
func TestLRUCache_SizeLimit(t *testing.T) {
	tests := []struct {
		name      string
		maxSize   int
		numItems  int
		finalSize int
	}{
		{"size limit 1", 1, 10, 1},
		{"size limit 5", 5, 20, 5},
		{"size limit 100", 100, 200, 100},
		{"size limit 1000", 1000, 1500, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLRUCache(tt.maxSize)

			// Add more items than maxSize
			for i := 0; i < tt.numItems; i++ {
				key := "key" + string(rune('0'+(i%10))) + string(rune('a'+(i/10)))
				cache.Put(key, &CachedResponse{Response: llm.CompletionResponse{Content: "value"}})
			}

			// Size should not exceed maxSize
			if cache.Size() != tt.finalSize {
				t.Errorf("Expected cache size %d, got %d", tt.finalSize, cache.Size())
			}

			stats := cache.Stats()
			if stats.Size != tt.finalSize {
				t.Errorf("Expected stats size %d, got %d", tt.finalSize, stats.Size)
			}
		})
	}
}

// TestLRUCache_Clear tests cache clearing
func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(10)

	// Add some entries
	for i := 0; i < 5; i++ {
		key := "key" + string(rune('0'+i))
		cache.Put(key, &CachedResponse{Response: llm.CompletionResponse{Content: "value"}})
	}

	if cache.Size() != 5 {
		t.Errorf("Expected cache size 5, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	// Verify all entries are gone
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	_, found := cache.Get("key0")
	if found {
		t.Error("Expected no entries after clear")
	}
}

// TestLRUCache_TTL tests TTL expiration
func TestLRUCache_TTL(t *testing.T) {
	t.Run("expired entry returns not found", func(t *testing.T) {
		cache := NewLRUCache(10)

		key := "expired-key"
		value := &CachedResponse{
			Key:       key,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
			Response:  llm.CompletionResponse{Content: "expired value"},
		}

		cache.Put(key, value)

		// Get should return not found for expired entry
		_, found := cache.Get(key)
		if found {
			t.Error("Expected expired entry to return not found")
		}

		// Stats should record this as a miss
		stats := cache.Stats()
		if stats.Misses == 0 {
			t.Error("Expected miss for expired entry")
		}
	})

	t.Run("non-expired entry returns found", func(t *testing.T) {
		cache := NewLRUCache(10)

		key := "valid-key"
		value := &CachedResponse{
			Key:       key,
			ExpiresAt: time.Now().Add(1 * time.Hour), // Expires in 1 hour
			Response:  llm.CompletionResponse{Content: "valid value"},
		}

		cache.Put(key, value)

		// Get should return the entry
		retrieved, found := cache.Get(key)
		if !found {
			t.Error("Expected valid entry to be found")
		}

		if retrieved.Content != "valid value" {
			t.Errorf("Expected 'valid value', got '%s'", retrieved.Content)
		}

		// Stats should record this as a hit
		stats := cache.Stats()
		if stats.Hits == 0 {
			t.Error("Expected hit for valid entry")
		}
	})

	t.Run("cleanupExpired removes expired entries", func(t *testing.T) {
		cache := NewLRUCache(10)

		// Add expired entries
		for i := 0; i < 3; i++ {
			key := "expired" + string(rune('0'+i))
			cache.Put(key, &CachedResponse{
				Key:       key,
				ExpiresAt: time.Now().Add(-1 * time.Hour),
				Response:  llm.CompletionResponse{Content: "expired"},
			})
		}

		// Add valid entries
		for i := 0; i < 5; i++ {
			key := "valid" + string(rune('0'+i))
			cache.Put(key, &CachedResponse{
				Key:       key,
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Response:  llm.CompletionResponse{Content: "valid"},
			})
		}

		if cache.Size() != 8 {
			t.Errorf("Expected cache size 8, got %d", cache.Size())
		}

		// Cleanup expired entries
		expiredCount := cache.CleanupExpired()

		if expiredCount != 3 {
			t.Errorf("Expected 3 expired entries, got %d", expiredCount)
		}

		if cache.Size() != 5 {
			t.Errorf("Expected cache size 5 after cleanup, got %d", cache.Size())
		}

		// Verify expired entries are gone
		for i := 0; i < 3; i++ {
			key := "expired" + string(rune('0'+i))
			_, found := cache.Get(key)
			if found {
				t.Errorf("Expected expired entry %s to be removed", key)
			}
		}

		// Verify valid entries still exist
		for i := 0; i < 5; i++ {
			key := "valid" + string(rune('0'+i))
			_, found := cache.Get(key)
			if !found {
				t.Errorf("Expected valid entry %s to still exist", key)
			}
		}
	})
}

// TestLRUCache_AccessCount tests access count tracking
func TestLRUCache_AccessCount(t *testing.T) {
	cache := NewLRUCache(10)

	key := "test-key"
	value := &CachedResponse{
		Key:   key,
		Response: llm.CompletionResponse{
			Content: "test value",
		},
		AccessCount: 0,
	}

	cache.Put(key, value)

	// First access
	cache.Get(key)
	retrieved, _ := cache.Get(key)

	if retrieved.AccessCount != 2 {
		t.Errorf("Expected access count 2, got %d", retrieved.AccessCount)
	}
}
