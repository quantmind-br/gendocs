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

func TestGeminiClient_GenerateCompletion_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate API key in query param
		apiKey := r.URL.Query().Get("key")
		if apiKey != "test-key" {
			t.Errorf("Expected API key 'test-key' in query, got '%s'", apiKey)
		}

		// Return mock response
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"text": "test response from gemini",
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     12,
				"candidatesTokenCount": 6,
				"totalTokenCount":      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

	if resp.Content != "test response from gemini" {
		t.Errorf("Expected content 'test response from gemini', got '%s'", resp.Content)
	}

	if resp.Usage.InputTokens != 12 {
		t.Errorf("Expected 12 input tokens, got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 6 {
		t.Errorf("Expected 6 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestGeminiClient_GenerateCompletion_WithToolCalls(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"functionCall": map[string]interface{}{
									"name": "list_files",
									"args": map[string]interface{}{
										"path": "src",
									},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     18,
				"candidatesTokenCount": 10,
				"totalTokenCount":      28,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "list files in src"},
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

	if path, ok := resp.ToolCalls[0].Arguments["path"].(string); !ok || path != "src" {
		t.Errorf("Expected path argument 'src', got %v", resp.ToolCalls[0].Arguments["path"])
	}
}

func TestGeminiClient_GenerateCompletion_InvalidAPIKey(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": 400, "message": "API key not valid", "status": "INVALID_ARGUMENT"}}`))
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGeminiClient_GenerateCompletion_SafetyBlocked(t *testing.T) {
	// Setup mock server - Gemini may block responses for safety reasons
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{},
						"role":  "model",
					},
					"finishReason": "SAFETY",
					"safetyRatings": []map[string]interface{}{
						{
							"category":    "HARM_CATEGORY_DANGEROUS_CONTENT",
							"probability": "HIGH",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	// Execute
	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "potentially unsafe content"},
		},
	})

	// Verify - should return error for safety-blocked content
	if err == nil {
		t.Fatal("Expected error for safety-blocked content, got nil")
	}
}

func TestGeminiClient_GenerateCompletion_RateLimitRetry(t *testing.T) {
	callCount := 0

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call returns rate limit error
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": {"code": 429, "message": "Resource exhausted", "status": "RESOURCE_EXHAUSTED"}}`))
			return
		}

		// Second call succeeds
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"text": "success after retry",
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
				"totalTokenCount":      15,
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
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGeminiClient_SupportsTools(t *testing.T) {
	client := NewGeminiClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "gemini-pro",
	}, nil)

	if !client.SupportsTools() {
		t.Error("Gemini client should support tools")
	}
}

func TestGeminiClient_GetProvider(t *testing.T) {
	client := NewGeminiClient(config.LLMConfig{
		APIKey: "test-key",
		Model:  "gemini-pro",
	}, nil)

	if provider := client.GetProvider(); provider != "gemini" {
		t.Errorf("Expected provider 'gemini', got '%s'", provider)
	}
}

func TestGeminiClient_GenerateCompletion_NoCandidates(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
		t.Fatal("Expected error for no candidates, got nil")
	}
}

func TestGeminiClient_GenerateCompletion_MultipleParts(t *testing.T) {
	// Test response with multiple text parts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"text": "First part. ",
							},
							{
								"text": "Second part.",
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 8,
				"totalTokenCount":      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	// Execute
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should concatenate all text parts
	expected := "First part. Second part."
	if resp.Content != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, resp.Content)
	}
}
