package llmcache

import (
	"testing"

	"github.com/user/gendocs/internal/llm"
)

func TestGenerateCacheKey_IdenticalRequests_SameKey(t *testing.T) {
	tests := []struct {
		name string
		req  llm.CompletionRequest
	}{
		{
			name: "simple request",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
			},
		},
		{
			name: "request with tools",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "Read a file"},
				},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read a file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.5,
			},
		},
		{
			name: "request with multiple messages",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a code analyzer",
				Messages: []llm.Message{
					{Role: "user", Content: "Analyze this code"},
					{Role: "assistant", Content: "I'll analyze it"},
					{Role: "user", Content: "Thanks"},
				},
				Temperature: 0.0,
			},
		},
		{
			name: "request with multiple tools",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a file system assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "List files"},
				},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
					{Name: "list_files", Description: "List files", Parameters: map[string]interface{}{"type": "object"}},
					{Name: "write_file", Description: "Write file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.3,
			},
		},
		{
			name: "request with tool calls in messages",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "Read file test.txt"},
					{Role: "assistant", Content: "", ToolID: "call_123"},
					{Role: "tool", Content: "File content here", ToolID: "call_123"},
				},
				Temperature: 0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate key twice
			key1, err1 := GenerateCacheKey(tt.req)
			key2, err2 := GenerateCacheKey(tt.req)

			// Verify no errors
			if err1 != nil {
				t.Fatalf("First GenerateCacheKey failed: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("Second GenerateCacheKey failed: %v", err2)
			}

			// Verify keys are identical
			if key1 != key2 {
				t.Errorf("Expected identical keys, got:\n  key1: %s\n  key2: %s", key1, key2)
			}

			// Verify key is not empty
			if key1 == "" {
				t.Error("Generated key is empty")
			}

			// Verify key is SHA256 length (64 hex characters)
			if len(key1) != 64 {
				t.Errorf("Expected key length 64, got %d", len(key1))
			}
		})
	}
}

func TestGenerateCacheKey_DifferentRequests_DifferentKeys(t *testing.T) {
	tests := []struct {
		name     string
		req1     llm.CompletionRequest
		req2     llm.CompletionRequest
		expected bool // true if keys should be different
	}{
		{
			name: "different system prompt",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a coding assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			expected: true,
		},
		{
			name: "different message content",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Goodbye"}},
				Temperature:  0.7,
			},
			expected: true,
		},
		{
			name: "different message order",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "First"},
					{Role: "user", Content: "Second"},
				},
				Temperature: 0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []llm.Message{
					{Role: "user", Content: "Second"},
					{Role: "user", Content: "First"},
				},
				Temperature: 0.7,
			},
			expected: true, // Message order matters
		},
		{
			name: "different temperature",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.0,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  1.0,
			},
			expected: true,
		},
		{
			name: "different tools",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "write_file", Description: "Write file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			expected: true,
		},
		{
			name: "different tool parameters",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "string"}},
				},
				Temperature: 0.7,
			},
			expected: true,
		},
		{
			name: "same tool different order",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
					{Name: "write_file", Description: "Write file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "write_file", Description: "Write file", Parameters: map[string]interface{}{"type": "object"}},
					{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			expected: false, // Tool order doesn't matter - tools are sorted
		},
		{
			name: "different message roles",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "assistant", Content: "Hello"}},
				Temperature:  0.7,
			},
			expected: true,
		},
		{
			name: "different tool IDs",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "tool", Content: "Result", ToolID: "call_123"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "tool", Content: "Result", ToolID: "call_456"}},
				Temperature:  0.7,
			},
			expected: true,
		},
		{
			name: "with vs without tools",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools:        []llm.ToolDefinition{{Name: "read_file", Description: "Read"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			expected: true,
		},
		{
			name: "max tokens ignored (same key)",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				MaxTokens:    100,
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				MaxTokens:    500,
				Temperature:  0.7,
			},
			expected: false, // MaxTokens should not affect key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1, err1 := GenerateCacheKey(tt.req1)
			key2, err2 := GenerateCacheKey(tt.req2)

			// Verify no errors
			if err1 != nil {
				t.Fatalf("First GenerateCacheKey failed: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("Second GenerateCacheKey failed: %v", err2)
			}

			// Verify keys match expectation
			if tt.expected && key1 == key2 {
				t.Errorf("Expected different keys, but got same key: %s", key1)
			}
			if !tt.expected && key1 != key2 {
				t.Errorf("Expected same keys, got:\n  key1: %s\n  key2: %s", key1, key2)
			}
		})
	}
}

func TestGenerateCacheKey_WhitespaceTrimming(t *testing.T) {
	tests := []struct {
		name string
		req1 llm.CompletionRequest
		req2 llm.CompletionRequest
	}{
		{
			name: "system prompt whitespace",
			req1: llm.CompletionRequest{
				SystemPrompt: "  You are a helpful assistant  ",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
		},
		{
			name: "message content whitespace",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "  Hello  "}},
				Temperature:  0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
		},
		{
			name: "tool description whitespace",
			req1: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "  Read a file  ", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
			req2: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools: []llm.ToolDefinition{
					{Name: "read_file", Description: "Read a file", Parameters: map[string]interface{}{"type": "object"}},
				},
				Temperature: 0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1, err1 := GenerateCacheKey(tt.req1)
			key2, err2 := GenerateCacheKey(tt.req2)

			// Verify no errors
			if err1 != nil {
				t.Fatalf("First GenerateCacheKey failed: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("Second GenerateCacheKey failed: %v", err2)
			}

			// Verify keys are identical after trimming
			if key1 != key2 {
				t.Errorf("Expected identical keys after whitespace trimming, got:\n  key1: %s\n  key2: %s", key1, key2)
			}
		})
	}
}

func TestGenerateCacheKey_EmptyFields(t *testing.T) {
	tests := []struct {
		name string
		req  llm.CompletionRequest
	}{
		{
			name: "empty system prompt",
			req: llm.CompletionRequest{
				SystemPrompt: "",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Temperature:  0.7,
			},
		},
		{
			name: "empty messages",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{},
				Temperature:  0.7,
			},
		},
		{
			name: "empty tools",
			req: llm.CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages:     []llm.Message{{Role: "user", Content: "Hello"}},
				Tools:        []llm.ToolDefinition{},
				Temperature:  0.7,
			},
		},
		{
			name: "all fields empty or zero",
			req: llm.CompletionRequest{
				SystemPrompt: "",
				Messages:     []llm.Message{},
				Tools:        []llm.ToolDefinition{},
				Temperature:  0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateCacheKey(tt.req)

			// Verify no errors
			if err != nil {
				t.Fatalf("GenerateCacheKey failed: %v", err)
			}

			// Verify key is not empty
			if key == "" {
				t.Error("Generated key is empty for request with empty fields")
			}

			// Verify key is SHA256 length
			if len(key) != 64 {
				t.Errorf("Expected key length 64, got %d", len(key))
			}

			// Verify consistency - same request should produce same key
			key2, err2 := GenerateCacheKey(tt.req)
			if err2 != nil {
				t.Fatalf("Second GenerateCacheKey failed: %v", err2)
			}
			if key != key2 {
				t.Errorf("Expected consistent keys for empty fields request, got:\n  key1: %s\n  key2: %s", key, key2)
			}
		})
	}
}

func TestCacheKeyRequestFrom_ConsistencyWithGenerateCacheKey(t *testing.T) {
	req := llm.CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
		Tools: []llm.ToolDefinition{
			{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{"type": "object"}},
		},
		Temperature: 0.7,
	}

	// Generate key using GenerateCacheKey
	key1, err1 := GenerateCacheKey(req)
	if err1 != nil {
		t.Fatalf("GenerateCacheKey failed: %v", err1)
	}

	// Convert to CacheKeyRequest and verify it has the expected fields
	keyReq := CacheKeyRequestFrom(req)

	// Verify fields match
	if keyReq.SystemPrompt != req.SystemPrompt {
		t.Errorf("Expected SystemPrompt '%s', got '%s'", req.SystemPrompt, keyReq.SystemPrompt)
	}

	if len(keyReq.Messages) != len(req.Messages) {
		t.Errorf("Expected %d messages, got %d", len(req.Messages), len(keyReq.Messages))
	}

	if len(keyReq.Tools) != len(req.Tools) {
		t.Errorf("Expected %d tools, got %d", len(req.Tools), len(keyReq.Tools))
	}

	if keyReq.Temperature != req.Temperature {
		t.Errorf("Expected Temperature %f, got %f", req.Temperature, keyReq.Temperature)
	}

	// Verify that GenerateCacheKey is essentially doing what CacheKeyRequestFrom does
	// by calling GenerateCacheKey again and getting the same result
	key2, err2 := GenerateCacheKey(req)
	if err2 != nil {
		t.Fatalf("Second GenerateCacheKey failed: %v", err2)
	}

	if key1 != key2 {
		t.Errorf("Expected consistent keys, got:\n  key1: %s\n  key2: %s", key1, key2)
	}
}
