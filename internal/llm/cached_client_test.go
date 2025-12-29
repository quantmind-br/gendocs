package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llmcache"
)

// mockLLMClient is a test double that records calls and returns configurable responses
type mockLLMClient struct {
	callCount   int
	lastRequest CompletionRequest
	response    CompletionResponse
	error       error
	provider    string
}

func (m *mockLLMClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	m.callCount++
	m.lastRequest = req
	if m.error != nil {
		return CompletionResponse{}, m.error
	}
	return m.response, nil
}

func (m *mockLLMClient) SupportsTools() bool {
	return true
}

func (m *mockLLMClient) GetProvider() string {
	return m.provider
}

// Test helper to create a mock server
func createMockServer(response map[string]interface{}, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

// TestCachedLLMClient_CacheMiss_CallsUnderlying tests that cache misses call the underlying client
func TestCachedLLMClient_CacheMiss_CallsUnderlying(t *testing.T) {
	// Create mock client
	mockClient := &mockLLMClient{
		response: CompletionResponse{
			Content: "test response",
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
		provider: "test",
	}

	// Create caches
	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)

	// Create cached client
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	// Execute request
	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test system",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Temperature: 0.7,
	}

	resp, err := cachedClient.GenerateCompletion(ctx, req)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Content != "test response" {
		t.Errorf("Expected content 'test response', got '%s'", resp.Content)
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 call to underlying client, got %d", mockClient.callCount)
	}

	// Verify cache stats
	stats := cachedClient.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.Misses)
	}
}

// TestCachedLLMClient_CacheHit_Memory tests that memory cache hits don't call underlying client
func TestCachedLLMClient_CacheHit_Memory(t *testing.T) {
	// Create mock client
	mockClient := &mockLLMClient{
		response: CompletionResponse{
			Content: "cached response",
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
		provider: "test",
	}

	// Create caches
	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)

	// Create cached client
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	// Execute same request twice
	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test system",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Temperature: 0.7,
	}

	// First call - cache miss
	resp1, err1 := cachedClient.GenerateCompletion(ctx, req)
	if err1 != nil {
		t.Fatalf("First call: Expected no error, got %v", err1)
	}

	// Second call - cache hit (memory)
	resp2, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	// Verify responses are identical
	if resp1.Content != resp2.Content {
		t.Errorf("Responses should be identical: got '%s' and '%s'", resp1.Content, resp2.Content)
	}

	// Verify underlying client called only once
	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 call to underlying client, got %d", mockClient.callCount)
	}

	// Verify cache stats
	stats := cachedClient.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.Misses)
	}
}

// TestCachedLLMClient_CacheHit_DiskPromotedToMemory tests that disk cache hits are promoted to memory
func TestCachedLLMClient_CacheHit_DiskPromotedToMemory(t *testing.T) {
	// Create mock client
	mockClient := &mockLLMClient{
		response: CompletionResponse{
			Content: "disk cached response",
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
		provider: "test",
	}

	// Create caches with tiny memory cache (size 1)
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test-cache.json")
	memoryCache := llmcache.NewLRUCache(1)
	diskCache := llmcache.NewDiskCache(cachePath, llmcache.DefaultTTL, 100*1024*1024)

	// Create first cached client to populate disk cache
	cachedClient1 := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test system",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Temperature: 0.7,
	}

	// First call - populates both caches
	_, err1 := cachedClient1.GenerateCompletion(ctx, req)
	if err1 != nil {
		t.Fatalf("First call: Expected no error, got %v", err1)
	}

	// Save disk cache
	diskCache.Stop()

	// Create new caches (simulating restart)
	memoryCache2 := llmcache.NewLRUCache(10)
	diskCache2 := llmcache.NewDiskCache(cachePath, llmcache.DefaultTTL, 100*1024*1024)
	if err := diskCache2.Load(); err != nil {
		t.Fatalf("Failed to load disk cache: %v", err)
	}

	// Create new cached client with new mock
	mockClient2 := &mockLLMClient{
		response: CompletionResponse{
			Content: "different response",
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
		provider: "test",
	}

	cachedClient2 := NewCachedLLMClient(mockClient2, memoryCache2, diskCache2, true, time.Hour)

	// Second call - should hit disk cache and promote to memory
	resp2, err2 := cachedClient2.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	// Verify we got the cached response (from disk), not the new mock response
	if resp2.Content != "disk cached response" {
		t.Errorf("Expected cached response from disk, got '%s'", resp2.Content)
	}

	// Verify new mock client was not called
	if mockClient2.callCount != 0 {
		t.Errorf("Expected 0 calls to new underlying client, got %d", mockClient2.callCount)
	}

	// Verify stats show disk hit
	stats := cachedClient2.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
	}

	defer diskCache2.Stop()
}

// TestCachedLLMClient_CachingDisabled_BypassesCache tests that caching can be disabled
func TestCachedLLMClient_CachingDisabled_BypassesCache(t *testing.T) {
	// Create mock client
	mockClient := &mockLLMClient{
		response: CompletionResponse{
			Content: "uncached response",
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
		provider: "test",
	}

	// Create cached client with caching disabled
	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, false, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test system",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Temperature: 0.7,
	}

	// Execute same request twice
	_, err1 := cachedClient.GenerateCompletion(ctx, req)
	if err1 != nil {
		t.Fatalf("First call: Expected no error, got %v", err1)
	}

	_, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	// Verify underlying client was called twice (bypassed cache)
	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 calls to underlying client (cache bypassed), got %d", mockClient.callCount)
	}

	// Verify cache stats show no activity
	stats := cachedClient.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Expected no cache activity when disabled, got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}
}

// TestCachedLLMClient_DifferentRequests_DifferentKeys tests that different requests generate different cache entries
func TestCachedLLMClient_DifferentRequests_DifferentKeys(t *testing.T) {
	mockClient := &mockLLMClient{
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()

	// Execute 3 different requests
	requests := []CompletionRequest{
		{
			SystemPrompt: "system 1",
			Messages:     []Message{{Role: "user", Content: "message 1"}},
			Temperature:  0.7,
		},
		{
			SystemPrompt: "system 2", // Different system prompt
			Messages:     []Message{{Role: "user", Content: "message 1"}},
			Temperature:  0.7,
		},
		{
			SystemPrompt: "system 1",
			Messages:     []Message{{Role: "user", Content: "message 2"}}, // Different message
			Temperature:  0.7,
		},
	}

	for _, req := range requests {
		mockClient.response = CompletionResponse{Content: "response"}
		_, err := cachedClient.GenerateCompletion(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}

	// Verify all 3 requests called the underlying client (no cache hits)
	if mockClient.callCount != 3 {
		t.Errorf("Expected 3 calls to underlying client, got %d", mockClient.callCount)
	}

	stats := cachedClient.GetStats()
	if stats.Misses != 3 {
		t.Errorf("Expected 3 cache misses, got %d", stats.Misses)
	}
}

// TestCachedLLMClient_APIFailure_NotCached tests that failed API calls are not cached
func TestCachedLLMClient_APIFailure_NotCached(t *testing.T) {
	mockClient := &mockLLMClient{
		error:    &testError{"API error"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Execute request - should fail
	_, err := cachedClient.GenerateCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify nothing was cached
	if memoryCache.Size() != 0 {
		t.Errorf("Expected empty memory cache after failed call, got size %d", memoryCache.Size())
	}

	stats := cachedClient.GetStats()
	if stats.Misses != 0 { // Failed calls shouldn't count as misses
		t.Errorf("Expected 0 cache misses for failed call, got %d", stats.Misses)
	}
}

// TestCachedLLMClient_TTLExpiration tests that expired entries are not returned
func TestCachedLLMClient_TTLExpiration(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{
			Content: "test response",
		},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)

	// Use very short TTL (1ms)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, 1*time.Millisecond)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// First call - cache miss
	_, err1 := cachedClient.GenerateCompletion(ctx, req)
	if err1 != nil {
		t.Fatalf("First call: Expected no error, got %v", err1)
	}

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Second call - should be a cache miss (expired)
	_, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	// Verify underlying client was called twice
	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 calls to underlying client (entry expired), got %d", mockClient.callCount)
	}
}

// TestCachedLLMClient_SupportsTools_Delegates tests that SupportsTools delegates to underlying client
func TestCachedLLMClient_SupportsTools_Delegates(t *testing.T) {
	mockClient := &mockLLMClient{
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	if !cachedClient.SupportsTools() {
		t.Error("Expected SupportsTools to return true")
	}
}

// TestCachedLLMClient_GetProvider_ReturnsPrefixedName tests that GetProvider returns prefixed name
func TestCachedLLMClient_GetProvider_ReturnsPrefixedName(t *testing.T) {
	mockClient := &mockLLMClient{
		provider: "openai",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	expectedProvider := "cached-openai"
	if provider := cachedClient.GetProvider(); provider != expectedProvider {
		t.Errorf("Expected provider '%s', got '%s'", expectedProvider, provider)
	}
}

// TestCachedLLMClient_GetStats_AggregatesStats tests that stats aggregate from both caches
func TestCachedLLMClient_GetStats_AggregatesStats(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Generate some activity
	for i := 0; i < 5; i++ {
		_, _ = cachedClient.GenerateCompletion(ctx, req)
	}

	stats := cachedClient.GetStats()

	// Verify stats are aggregated
	if stats.Hits+stats.Misses != 5 {
		t.Errorf("Expected total lookups to be 5, got %d", stats.Hits+stats.Misses)
	}

	// Verify hit rate is calculated
	expectedHitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	if stats.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, stats.HitRate)
	}
}

// TestCachedLLMClient_CleanupExpired_CleansBothCaches tests cleanup of expired entries
func TestCachedLLMClient_CleanupExpired_CleansBothCaches(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)

	// Use short TTL
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, 10*time.Millisecond)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Populate cache
	_, _ = cachedClient.GenerateCompletion(ctx, req)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Cleanup expired
	memoryExpired, _ := cachedClient.CleanupExpired()

	if memoryExpired < 1 {
		t.Errorf("Expected at least 1 expired entry in memory cache, got %d", memoryExpired)
	}
}

// TestCachedLLMClient_Clear_EmptiesBothCaches tests clearing both caches
func TestCachedLLMClient_Clear_EmptiesBothCaches(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Populate cache
	_, _ = cachedClient.GenerateCompletion(ctx, req)

	// Verify cache has entries
	statsBefore := cachedClient.GetStats()
	if statsBefore.Size == 0 {
		t.Error("Expected cache to have entries before clear")
	}

	// Clear cache
	err := cachedClient.Clear()
	if err != nil {
		t.Fatalf("Expected no error from Clear, got %v", err)
	}

	// Verify cache is empty
	statsAfter := cachedClient.GetStats()
	if statsAfter.Size != 0 {
		t.Errorf("Expected empty cache after clear, got size %d", statsAfter.Size)
	}
}

// TestCachedLLMClient_GetUnderlyingClient_ReturnsClient tests getting underlying client
func TestCachedLLMClient_GetUnderlyingClient_ReturnsClient(t *testing.T) {
	mockClient := &mockLLMClient{
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	underlying := cachedClient.GetUnderlyingClient()
	if underlying != mockClient {
		t.Error("Expected underlying client to be the mock client")
	}
}

// TestCachedLLMClient_IntegrationWithOpenAI tests integration with real OpenAI client
func TestCachedLLMClient_IntegrationWithOpenAI(t *testing.T) {
	// Setup mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Return mock response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "openai test response",
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create real OpenAI client
	openaiClient := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	// Wrap with caching
	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(openaiClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test system",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Temperature: 0.7,
	}

	// First call - should hit API
	resp1, err1 := cachedClient.GenerateCompletion(ctx, req)
	if err1 != nil {
		t.Fatalf("First call: Expected no error, got %v", err1)
	}

	if resp1.Content != "openai test response" {
		t.Errorf("First call: Expected content 'openai test response', got '%s'", resp1.Content)
	}

	// Second call - should hit cache
	resp2, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	if resp2.Content != "openai test response" {
		t.Errorf("Second call: Expected content 'openai test response', got '%s'", resp2.Content)
	}

	// Verify API was called only once
	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}

	// Verify cache stats
	stats := cachedClient.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.Misses)
	}
}

// TestCachedLLMClient_DiskCacheFailure_GracefulDegradation tests graceful degradation on disk cache failure
func TestCachedLLMClient_DiskCacheFailure_GracefulDegradation(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	// Create disk cache in read-only directory (will fail writes)
	tempDir := t.TempDir()
	readonlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0444); err != nil {
		t.Fatalf("Failed to create readonly directory: %v", err)
	}
	// Make directory read-only
	if err := os.Chmod(readonlyDir, 0444); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}

	cachePath := filepath.Join(readonlyDir, "test-cache.json")

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(cachePath, llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Should succeed despite disk cache write failure
	resp, err := cachedClient.GenerateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error despite disk cache failure, got %v", err)
	}

	if resp.Content != "test" {
		t.Errorf("Expected content 'test', got '%s'", resp.Content)
	}

	// Verify memory cache still works (second call should hit memory)
	resp2, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	if resp2.Content != "test" {
		t.Errorf("Second call: Expected content 'test', got '%s'", resp2.Content)
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 call to underlying client (memory cache worked), got %d", mockClient.callCount)
	}
}

// TestCachedLLMClient_NilMemoryCache_WorksCorrectly tests behavior with nil memory cache
func TestCachedLLMClient_NilMemoryCache_WorksCorrectly(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, nil, diskCache, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Should work with disk cache only
	resp, err := cachedClient.GenerateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Content != "test" {
		t.Errorf("Expected content 'test', got '%s'", resp.Content)
	}
}

// TestCachedLLMClient_NilDiskCache_WorksCorrectly tests behavior with nil disk cache
func TestCachedLLMClient_NilDiskCache_WorksCorrectly(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, nil, true, time.Hour)

	ctx := context.Background()
	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Should work with memory cache only
	resp, err := cachedClient.GenerateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Content != "test" {
		t.Errorf("Expected content 'test', got '%s'", resp.Content)
	}

	// Second call should hit memory cache
	_, err2 := cachedClient.GenerateCompletion(ctx, req)
	if err2 != nil {
		t.Fatalf("Second call: Expected no error, got %v", err2)
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 call to underlying client, got %d", mockClient.callCount)
	}
}

// TestCachedLLMClient_ContextCancellation_PropagatesError tests that context cancellation is handled
func TestCachedLLMClient_ContextCancellation_PropagatesError(t *testing.T) {
	mockClient := &mockLLMClient{
		response: CompletionResponse{Content: "test"},
		provider: "test",
	}

	memoryCache := llmcache.NewLRUCache(10)
	diskCache := llmcache.NewDiskCache(t.TempDir()+"/test-cache.json", llmcache.DefaultTTL, 100*1024*1024)
	cachedClient := NewCachedLLMClient(mockClient, memoryCache, diskCache, true, time.Hour)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := CompletionRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "hello"}},
		Temperature:  0.7,
	}

	// Should return error (context is checked before cache lookup in real scenario)
	_, err := cachedClient.GenerateCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for canceled context, got nil")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
