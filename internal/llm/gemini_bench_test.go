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

// BenchmarkGemini_SmallResponse benchmarks a small single-chunk response
func BenchmarkGemini_SmallResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Small response - single text chunk
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Hello!"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

// BenchmarkGemini_MediumResponse benchmarks a medium multi-chunk response
func BenchmarkGemini_MediumResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Medium response - multiple chunks
		for i := 0; i < 10; i++ {
			fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"This is chunk `+fmt.Sprint(i)+` of the response. "}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":`+fmt.Sprint(2+i*5)+`,"totalTokenCount":`+fmt.Sprint(17+i*5)+`}}`)
		}
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":50,"totalTokenCount":65}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

// BenchmarkGemini_LargeResponse benchmarks a large response with many chunks
func BenchmarkGemini_LargeResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Large response - many chunks (simulating ~50KB response)
		for i := 0; i < 100; i++ {
			fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"This is a longer chunk of text that represents a substantial part of the response. Chunk `+fmt.Sprint(i)+` contains useful information. "}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":`+fmt.Sprint(2+i*5)+`,"totalTokenCount":`+fmt.Sprint(22+i*5)+`}}`)
		}
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":500,"totalTokenCount":520}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

// BenchmarkGemini_FunctionCall benchmarks a response with a function call
func BenchmarkGemini_FunctionCall(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Function call response (complete, not partial like Anthropic/OpenAI)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"I'll read the file for you."}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":5,"totalTokenCount":25}}`)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read_file","args":{"file_path":"src/main.go","start_line":1,"end_line":100}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":15,"totalTokenCount":35}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

// BenchmarkGemini_MultipleFunctionCalls benchmarks multiple function calls in one response
func BenchmarkGemini_MultipleFunctionCalls(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Multiple function calls in a single chunk
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"I'll read the file and list the directory."}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":25,"candidatesTokenCount":8,"totalTokenCount":33}}`)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read_file","args":{"path":"main.go"}}},{"functionCall":{"name":"list_files","args":{"path":"src"}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":25,"candidatesTokenCount":20,"totalTokenCount":45}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Read main.go and list files in src"},
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

// BenchmarkGemini_TimeToFirstToken measures time to receive first content
func BenchmarkGemini_TimeToFirstToken(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate network delay before first chunk
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Response"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

// BenchmarkGemini_MixedContent benchmarks a response with text followed by function call
func BenchmarkGemini_MixedContent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Text response across multiple chunks
		for i := 0; i < 5; i++ {
			fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Chunk `+fmt.Sprint(i)+` "}],"role":"model"},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":`+fmt.Sprint(2+i*2)+`,"totalTokenCount":`+fmt.Sprint(17+i*2)+`}}`)
		}
		// Final chunk with function call
		fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"list_files","args":{"path":"src"}}}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":15,"totalTokenCount":30}}`)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, nil)

	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "List files in src"},
		},
		Tools: []ToolDefinition{
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
