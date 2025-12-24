package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/gendocs/internal/config"
)

func TestOpenAIClient_GenerateCompletion_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got '%s'", auth)
		}

		// Return mock response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "test response",
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

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "You are a test assistant",
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

	if resp.Content != "test response" {
		t.Errorf("Expected content 'test response', got '%s'", resp.Content)
	}

	if resp.Usage.InputTokens != 10 {
		t.Errorf("Expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 5 {
		t.Errorf("Expected 5 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestOpenAIClient_GenerateCompletion_WithToolCalls(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "read_file",
									"arguments": `{"file_path": "test.go"}`,
								},
							},
						},
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "You are a test assistant",
		Messages: []Message{
			{Role: "user", Content: "read test.go"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{
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

	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected tool call name 'read_file', got '%s'", resp.ToolCalls[0].Name)
	}

	if filePath, ok := resp.ToolCalls[0].Arguments["file_path"].(string); !ok || filePath != "test.go" {
		t.Errorf("Expected file_path argument 'test.go', got %v", resp.ToolCalls[0].Arguments["file_path"])
	}
}

func TestOpenAIClient_GenerateCompletion_InvalidAPIKey(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`))
	}))
	defer server.Close()

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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

func TestOpenAIClient_GenerateCompletion_RateLimitRetry(t *testing.T) {
	callCount := 0

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call returns rate limit error
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`))
			return
		}

		// Second call succeeds
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "success after retry",
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

	// Create retry client with short delays
	retryClient := NewRetryClient(&RetryConfig{
		MaxAttempts:       2,
		Multiplier:        1,
		MaxWaitPerAttempt: 10 * time.Millisecond,
		MaxTotalWait:      100 * time.Millisecond,
	})

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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

func TestOpenAIClient_SupportsTools(t *testing.T) {
	client := NewOpenAIClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "gpt-4",
	}, nil)

	if !client.SupportsTools() {
		t.Error("OpenAI client should support tools")
	}
}

func TestOpenAIClient_GetProvider(t *testing.T) {
	client := NewOpenAIClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "gpt-4",
	}, nil)

	if provider := client.GetProvider(); provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", provider)
	}
}

func TestOpenAIClient_GenerateCompletion_EmptyResponse(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with no choices
		response := map[string]interface{}{
			"choices": []map[string]interface{}{},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 0,
				"total_tokens":      10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Verify - empty response should not error, just return empty content
	if err != nil {
		t.Fatalf("Expected no error for empty response, got %v", err)
	}

	if resp.Content != "" {
		t.Errorf("Expected empty content, got '%s'", resp.Content)
	}
}

func TestOpenAIClient_GenerateCompletion_ContextCanceled(t *testing.T) {
	// Setup mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should never complete due to context cancellation
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	// Create client
	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute
	_, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Verify
	if err == nil {
		t.Fatal("Expected error for canceled context, got nil")
	}

	// Error should be wrapped, so use errors.Is
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
