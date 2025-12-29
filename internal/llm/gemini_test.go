package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

		// Send NDJSON streaming response
		w.Header().Set("Content-Type", "application/json")
		// Chunk 1: Partial text
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"test response"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":2,"totalTokenCount":14}}`)
		// Chunk 2: More text
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":" from gemini"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":4,"totalTokenCount":16}}`)
		// Chunk 3: Final with finishReason
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":6,"totalTokenCount":18}}`)
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
		// Send NDJSON streaming response with function call
		w.Header().Set("Content-Type", "application/json")
		// Chunk 1: Text response
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"I'll list the files."}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":18,"candidatesTokenCount":4,"totalTokenCount":22}}`)
		// Chunk 2: Function call (complete, not partial)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"list_files","args":{"path":"src"}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":18,"candidatesTokenCount":10,"totalTokenCount":28}}`)
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
		// Send NDJSON streaming response with safety finish reason
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"I cannot"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"SAFETY","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3,"totalTokenCount":13}}`)
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

		// Second call succeeds with streaming response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"success after"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":" retry"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`)
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
	// Test response with multiple text parts in a single chunk
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send NDJSON streaming response with multiple parts in one chunk
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"First part. "},{"text":"Second part."}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":8,"totalTokenCount":18}}`)
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

func TestGeminiClient_Streaming_MultipleChunks(t *testing.T) {
	// Test large response split across multiple NDJSON chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Send multiple text chunks
		chunks := []string{
			`{"candidates":[{"content":{"parts":[{"text":"This is"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`,
			`{"candidates":[{"content":{"parts":[{"text":" a large"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3,"totalTokenCount":13}}`,
			`{"candidates":[{"content":{"parts":[{"text":" response"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":4,"totalTokenCount":14}}`,
			`{"candidates":[{"content":{"parts":[{"text":" split"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`,
			`{"candidates":[{"content":{"parts":[{"text":" across"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":6,"totalTokenCount":16}}`,
			`{"candidates":[{"content":{"parts":[{"text":" multiple"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":7,"totalTokenCount":17}}`,
			`{"candidates":[{"content":{"parts":[{"text":" chunks."}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":8,"totalTokenCount":18}}`,
		}
		for _, chunk := range chunks {
			fmt.Fprintln(w, chunk)
		}
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "generate a large response"},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "This is a large response split across multiple chunks."
	if resp.Content != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, resp.Content)
	}

	if resp.Usage.OutputTokens != 8 {
		t.Errorf("Expected 8 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestGeminiClient_Streaming_IncompleteStream(t *testing.T) {
	// Test incomplete stream (connection closes without finishReason)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Send some text but never send finishReason
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Partial response"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`)
		// Connection closes here without finishReason
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	// Should succeed with partial content (no error - scanner just reaches EOF)
	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error for incomplete stream, got %v", err)
	}

	// Should have accumulated partial content
	if resp.Content != "Partial response" {
		t.Errorf("Expected partial content, got '%s'", resp.Content)
	}
}

func TestGeminiClient_Streaming_MalformedChunk(t *testing.T) {
	// Test malformed JSON in stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Valid chunk"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`)
		// Send malformed JSON
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"broken`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Should return error for malformed JSON
	if err == nil {
		t.Fatal("Expected error for malformed chunk, got nil")
	}
}

func TestGeminiClient_Streaming_APIErrorInStream(t *testing.T) {
	// Test API error in stream chunk
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Starting"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":1,"totalTokenCount":11}}`)
		// Send chunk with error
		fmt.Fprintln(w, `{"error":{"code":400,"message":"Invalid request","status":"INVALID_ARGUMENT"}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Should return error from stream
	if err == nil {
		t.Fatal("Expected error for API error in stream, got nil")
	}
}

func TestGeminiClient_Streaming_FunctionCallComplete(t *testing.T) {
	// Test that function calls arrive complete (not partial like Anthropic/OpenAI)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Text chunks
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"I'll"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":1,"totalTokenCount":16}}`)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":" search"}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":2,"totalTokenCount":17}}`)
		// Function call arrives complete in one chunk (not partial JSON)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"search","args":{"query":"example","limit":10}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":8,"totalTokenCount":23}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "search for example"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "Search",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
						"limit": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("Expected tool call name 'search', got '%s'", resp.ToolCalls[0].Name)
	}

	if query, ok := resp.ToolCalls[0].Arguments["query"].(string); !ok || query != "example" {
		t.Errorf("Expected query argument 'example', got %v", resp.ToolCalls[0].Arguments["query"])
	}

	if limit, ok := resp.ToolCalls[0].Arguments["limit"].(float64); !ok || int(limit) != 10 {
		t.Errorf("Expected limit argument 10, got %v", resp.ToolCalls[0].Arguments["limit"])
	}

	// Text should also be accumulated
	if resp.Content != "I'll search" {
		t.Errorf("Expected text content 'I'll search', got '%s'", resp.Content)
	}
}

func TestGeminiClient_Streaming_MultipleFunctionCalls(t *testing.T) {
	// Test multiple function calls in a single chunk
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Multiple function calls in one chunk
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"search","args":{"query":"cats"}}},{"functionCall":{"name":"search","args":{"query":"dogs"}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":10,"totalTokenCount":30}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "search for cats and dogs"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "Search",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Fatalf("Expected 2 tool calls, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "search" || resp.ToolCalls[1].Name != "search" {
		t.Errorf("Expected both tool calls to be 'search', got '%s' and '%s'", resp.ToolCalls[0].Name, resp.ToolCalls[1].Name)
	}

	if query0, ok := resp.ToolCalls[0].Arguments["query"].(string); !ok || query0 != "cats" {
		t.Errorf("Expected first query 'cats', got %v", resp.ToolCalls[0].Arguments["query"])
	}

	if query1, ok := resp.ToolCalls[1].Arguments["query"].(string); !ok || query1 != "dogs" {
		t.Errorf("Expected second query 'dogs', got %v", resp.ToolCalls[1].Arguments["query"])
	}
}
