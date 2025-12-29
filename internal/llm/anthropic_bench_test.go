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

// BenchmarkAnthropic_SmallResponse benchmarks a small single-chunk response
func BenchmarkAnthropic_SmallResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Small response - single chunk
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello!"}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_stop","index":0}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":2}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

// BenchmarkAnthropic_MediumResponse benchmarks a medium multi-chunk response
func BenchmarkAnthropic_MediumResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Medium response - multiple chunks
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":15,"output_tokens":0}}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		_, _ = fmt.Fprintln(w)
		for i := 0; i < 10; i++ {
			_, _ = fmt.Fprintln(w, "event: content_block_delta")
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"This is chunk `+fmt.Sprint(i)+` of the response. "}}`)
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, "event: content_block_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_stop","index":0}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":50}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

// BenchmarkAnthropic_LargeResponse benchmarks a large response with many chunks
func BenchmarkAnthropic_LargeResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Large response - many chunks (simulating ~50KB response)
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":20,"output_tokens":0}}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		_, _ = fmt.Fprintln(w)
		for i := 0; i < 100; i++ {
			_, _ = fmt.Fprintln(w, "event: content_block_delta")
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"This is a longer chunk of text that represents a substantial part of the response. Chunk `+fmt.Sprint(i)+` contains useful information. "}}`)
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, "event: content_block_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_stop","index":0}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":500}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

// BenchmarkAnthropic_ToolCall benchmarks a response with a tool call
func BenchmarkAnthropic_ToolCall(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Tool call response
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":25,"output_tokens":0}}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_123","name":"read_file","input":null}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":\"src/main.go\",\"start_line\":1,\"end_line\":100}"}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_stop","index":0}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":25}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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

// BenchmarkAnthropic_TimeToFirstToken measures time to receive first content
func BenchmarkAnthropic_TimeToFirstToken(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Simulate network delay before first chunk
		time.Sleep(10 * time.Millisecond)
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Response"}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: content_block_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_stop","index":0}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
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
