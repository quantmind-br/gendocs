package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/gendocs/internal/config"
)

func TestAnthropicClient_GenerateCompletion_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request headers
		if apiKey := r.Header.Get("x-api-key"); apiKey != "test-key" {
			t.Errorf("Expected x-api-key 'test-key', got '%s'", apiKey)
		}

		if version := r.Header.Get("anthropic-version"); version == "" {
			t.Error("Expected anthropic-version header to be set")
		}

		// Return mock response
		response := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "test response from claude",
				},
			},
			"model": "claude-3-sonnet-20240229",
			"usage": map[string]interface{}{
				"input_tokens":  15,
				"output_tokens": 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Content != "test response from claude" {
		t.Errorf("Expected content 'test response from claude', got '%s'", resp.Content)
	}

	if resp.Usage.InputTokens != 15 {
		t.Errorf("Expected 15 input tokens, got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 8 {
		t.Errorf("Expected 8 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestAnthropicClient_GenerateCompletion_WithToolCalls(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "tool_use",
					"id":   "toolu_123",
					"name": "list_files",
					"input": map[string]interface{}{
						"path": ".",
					},
				},
			},
			"model": "claude-3-sonnet-20240229",
			"usage": map[string]interface{}{
				"input_tokens":  20,
				"output_tokens": 12,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "list files"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "list_files",
				Description: "List files in directory",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	})

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "list_files" {
		t.Errorf("Expected tool call name 'list_files', got '%s'", resp.ToolCalls[0].Name)
	}

	if path, ok := resp.ToolCalls[0].Arguments["path"].(string); !ok || path != "." {
		t.Errorf("Expected path argument '.', got %v", resp.ToolCalls[0].Arguments["path"])
	}
}

func TestAnthropicClient_GenerateCompletion_InvalidAPIKey(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type": "error", "error": {"type": "authentication_error", "message": "Invalid API key"}}`))
	}))
	defer server.Close()

	// Create client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	// Execute
	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Verify
	if err == nil {
		t.Fatal("Expected error for invalid API key, got nil")
	}
}

func TestAnthropicClient_GenerateCompletion_RateLimitRetry(t *testing.T) {
	callCount := 0

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call returns rate limit error
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"type": "error", "error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`))
			return
		}

		// Second call succeeds
		response := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "success after retry",
				},
			},
			"model": "claude-3-sonnet-20240229",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create retry client with short delays
	retryClient := NewRetryClient(&RetryConfig{
		MaxAttempts:       2,
		Multiplier:        1,
		MaxWaitPerAttempt: 10 * time.Millisecond,
		MaxTotalWait:      100 * time.Millisecond,
	})

	// Create client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, retryClient)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Verify
	if err != nil {
		t.Fatalf("Expected no error after retry, got %v", err)
	}

	if resp.Content != "success after retry" {
		t.Errorf("Expected content 'success after retry', got '%s'", resp.Content)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls (1 fail + 1 success), got %d", callCount)
	}
}

func TestAnthropicClient_SupportsTools(t *testing.T) {
	client := NewAnthropicClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "claude-3-sonnet-20240229",
	}, nil)

	if !client.SupportsTools() {
		t.Error("Anthropic client should support tools")
	}
}

func TestAnthropicClient_GetProvider(t *testing.T) {
	client := NewAnthropicClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "claude-3-sonnet-20240229",
	}, nil)

	if provider := client.GetProvider(); provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", provider)
	}
}

func TestAnthropicClient_GenerateCompletion_MixedContentTypes(t *testing.T) {
	// Test response with both text and tool_use content blocks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "I'll read the file for you.",
				},
				{
					"type": "tool_use",
					"id":   "toolu_123",
					"name": "read_file",
					"input": map[string]interface{}{
						"file_path": "main.go",
					},
				},
			},
			"model": "claude-3-sonnet-20240229",
			"usage": map[string]interface{}{
				"input_tokens":  25,
				"output_tokens": 15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "read main.go"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters:  map[string]interface{}{},
			},
		},
	})

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should extract text content
	if resp.Content != "I'll read the file for you." {
		t.Errorf("Expected text content, got '%s'", resp.Content)
	}

	// Should also extract tool calls
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected tool call name 'read_file', got '%s'", resp.ToolCalls[0].Name)
	}
}
