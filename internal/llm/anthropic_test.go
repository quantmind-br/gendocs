package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

		// Send streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":15,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"test response from claude\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":8}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
		fmt.Fprintln(w)
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
		// Send streaming response for tool call
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":20,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_123\",\"name\":\"list_files\",\"input\":null}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\":\\\".\\\"}\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":12}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
		fmt.Fprintln(w)
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

		// Second call succeeds with streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"success after retry\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":5}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
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
		// Send streaming response with mixed content
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":25,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		// First content block: text
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"I'll read the file for you.\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		// Second content block: tool_use
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_123\",\"name\":\"read_file\",\"input\":null}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"file_path\\\":\\\"main.go\\\"}\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":1}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":15}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
		fmt.Fprintln(w)
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

func TestAnthropicClient_Streaming_MultipleChunks(t *testing.T) {
	// Test large response split across multiple chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)

		// Send multiple text chunks
		chunks := []string{"This is ", "a large ", "response ", "split across ", "multiple chunks."}
		for _, chunk := range chunks {
			fmt.Fprintln(w, "event: content_block_delta")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"%s\"}}\n", chunk)
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":10}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

func TestAnthropicClient_Streaming_IncompleteStream(t *testing.T) {
	// Test incomplete stream (missing message_stop event)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)
		// Stream ends here without message_stop
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

func TestAnthropicClient_Streaming_APIError(t *testing.T) {
	// Test API error event in stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: error")
		fmt.Fprintln(w, "data: {\"type\":\"error\",\"error\":{\"type\":\"content_filter\",\"message\":\"Content was filtered\"}}")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	_, err := client.GenerateCompletion(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	if err == nil {
		t.Fatal("Expected error for API error event, got nil")
	}

	if !strings.Contains(err.Error(), "Content was filtered") {
		t.Errorf("Expected error message to contain 'Content was filtered', got '%s'", err.Error())
	}
}

func TestAnthropicClient_Streaming_MalformedChunk(t *testing.T) {
	// Test malformed JSON in chunk
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {invalid json here}")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

func TestAnthropicClient_Streaming_LargeToolArguments(t *testing.T) {
	// Test tool call with large arguments split across chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "event: message_start")
		fmt.Fprintln(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-sonnet-20240229\",\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: content_block_start")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_123\",\"name\":\"search\",\"input\":null}}")
		fmt.Fprintln(w)

		// Send partial JSON chunks
		chunks := []string{
			"{\"query\":",
			"\"large search request ",
			"with multiple parameters",
			" split across chunks\"}",
		}
		for _, chunk := range chunks {
			fmt.Fprintln(w, "event: content_block_delta")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"%s\"}}\n", chunk)
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w, "event: content_block_stop")
		fmt.Fprintln(w, "data: {\"type\":\"content_block_stop\",\"index\":0}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_delta")
		fmt.Fprintln(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":15}}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {\"type\":\"message_stop\"}")
		fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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
