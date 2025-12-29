package llm

import (
	"context"
	"fmt"
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

		// Send streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"test response\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
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

	if resp.Usage.InputTokens != 0 {
		t.Errorf("Expected 0 input tokens (not tracked in streaming), got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 0 {
		t.Errorf("Expected 0 output tokens (not tracked in streaming), got %d", resp.Usage.OutputTokens)
	}
}

func TestOpenAIClient_GenerateCompletion_WithToolCalls(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send streaming response with tool call
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_123\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"file_path\\\":\\\"test.go\\\"}\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
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

		// Second call succeeds with streaming
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"success after retry\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
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
		// Send streaming response with empty content
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
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

func TestOpenAIClient_Streaming_MultipleChunks(t *testing.T) {
	// Test large response split across multiple chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		// Send multiple content chunks
		chunks := []string{"This is ", "a large ", "response ", "split across ", "multiple chunks."}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"%s\"},\"finish_reason\":null}]}\n", chunk)
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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
}

func TestOpenAIClient_Streaming_ToolCallsWithChunks(t *testing.T) {
	// Test tool calls with arguments split across multiple chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		// Start tool call
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_123\",\"type\":\"function\",\"function\":{\"name\":\"search\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		// Send arguments in multiple chunks
		argChunks := []string{
			"{\"query\":",
			"\"large search request ",
			"with multiple parameters",
			" split across chunks\"}",
		}
		for _, chunk := range argChunks {
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"%s\"}}]},\"finish_reason\":null}]}\n", chunk)
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "search"},
		},
		Tools: []ToolDefinition{
			{Name: "search", Description: "Search", Parameters: map[string]interface{}{}},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("Expected tool name 'search', got '%s'", resp.ToolCalls[0].Name)
	}

	expectedQuery := "large search request with multiple parameters split across chunks"
	if query, ok := resp.ToolCalls[0].Arguments["query"].(string); !ok || query != expectedQuery {
		t.Errorf("Expected query '%s', got '%v'", expectedQuery, resp.ToolCalls[0].Arguments["query"])
	}
}

func TestOpenAIClient_Streaming_IncompleteStream(t *testing.T) {
	// Test incomplete stream (missing [DONE] marker and finish_reason)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial response\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		// Stream ends here without [DONE] or finish_reason
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	// Should return an error due to incomplete stream
	if err == nil {
		t.Fatal("Expected error for incomplete stream, got nil")
	}
}

func TestOpenAIClient_Streaming_MalformedChunk(t *testing.T) {
	// Test malformed JSON in chunk
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {invalid json here}")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	if err == nil {
		t.Fatal("Expected error for malformed chunk, got nil")
	}
}

func TestOpenAIClient_Streaming_MultipleToolCalls(t *testing.T) {
	// Test multiple tool calls in streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		// First tool call
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"path\\\":\\\"file1.txt\\\"}\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		// Second tool call
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":1,\"id\":\"call_2\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":1,\"function\":{\"arguments\":\"{\\\"path\\\":\\\"file2.txt\\\"}\"}}]},\"finish_reason\":null}]}")
		fmt.Fprintln(w)

		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "read two files"},
		},
		Tools: []ToolDefinition{
			{Name: "read_file", Description: "Read file", Parameters: map[string]interface{}{}},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Fatalf("Expected 2 tool calls, got %d", len(resp.ToolCalls))
	}

	// Verify first tool call
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected first tool call name 'read_file', got '%s'", resp.ToolCalls[0].Name)
	}

	// Verify second tool call
	if resp.ToolCalls[1].Name != "read_file" {
		t.Errorf("Expected second tool call name 'read_file', got '%s'", resp.ToolCalls[1].Name)
	}
}
