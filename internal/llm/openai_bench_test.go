package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/gendocs/internal/config"
)

// BenchmarkOpenAI_SmallResponse benchmarks a small single-chunk response
func BenchmarkOpenAI_SmallResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Small response - single content chunk
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello!"},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkOpenAI_MediumResponse benchmarks a medium multi-chunk response
func BenchmarkOpenAI_MediumResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Medium response - multiple chunks
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		for i := 0; i < 10; i++ {
			_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"This is chunk `+fmt.Sprint(i)+` of the response. "},"finish_reason":null}]}`)
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Tell me a story"},
		},
		MaxTokens:   1000,
		Temperature: 0.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkOpenAI_LargeResponse benchmarks a large response with many chunks
func BenchmarkOpenAI_LargeResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Large response - many chunks (simulating ~50KB response)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		for i := 0; i < 100; i++ {
			_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"This is a longer chunk of text that represents a substantial part of the response. Chunk `+fmt.Sprint(i)+` contains useful information. "},"finish_reason":null}]}`)
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Analyze this codebase in detail"},
		},
		MaxTokens:   4000,
		Temperature: 0.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkOpenAI_ToolCall benchmarks a response with a tool call
func BenchmarkOpenAI_ToolCall(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Tool call response with delta accumulation
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"file_path\":\"src/main.go\",\"start_line\":1,\"end_line\":100}"}}]},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Read the main.go file"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file",
						},
						"start_line": map[string]interface{}{
							"type": "integer",
						},
						"end_line": map[string]interface{}{
							"type": "integer",
						},
					},
					"required": []string{"file_path"},
				},
			},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkOpenAI_MultipleToolCalls benchmarks multiple tool calls in one response
func BenchmarkOpenAI_MultipleToolCalls(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Multiple tool calls
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"read_file","arguments":""}},{"index":1,"id":"call_124","type":"function","function":{"name":"list_files","arguments":""}}]},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"file1.go\"}"}}]},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"path\":\"src\"}"}}]},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Read file1.go and list files in src"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"path"},
				},
			},
			{
				Name:        "list_files",
				Description: "List files",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkOpenAI_TimeToFirstToken measures time to receive first content
func BenchmarkOpenAI_TimeToFirstToken(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Simulate network delay before first chunk
		time.Sleep(10 * time.Millisecond)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Response"},"finish_reason":null}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: [DONE]`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := client.GenerateCompletion(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
		elapsed := time.Since(start)
		b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
	}
}
