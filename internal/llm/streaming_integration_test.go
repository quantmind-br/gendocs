//go:build integration
// +build integration

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

// TestAnthropicStreaming_AgentWorkflow tests Anthropic streaming with a simulated agent workflow
// This test validates that the streaming implementation works correctly with real agent workloads
// including tool calling and multiple LLM requests.
func TestAnthropicStreaming_AgentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	callCount := 0

	// Create mock server that simulates an agent workflow with tool calling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Validate streaming request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check for stream parameter
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		bodyStr := string(body[:n])
		if !strings.Contains(bodyStr, `"stream":true`) {
			t.Errorf("Expected stream=true in request body")
		}

		w.Header().Set("Content-Type", "text/event-stream")

		// Simulate different responses based on call count (agent workflow)
		switch callCount {
		case 1:
			// First call: LLM requests to list files
			sendSSEEvent(w, "message_start", `{"type":"message_start","message":{"id":"msg_1","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":0}}}`)
			sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"list_files","input":null}}`)
			sendSSEEvent(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"directory\":\".\\"}}`)
			sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
			sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":25}}`)
			sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)

		case 2:
			// Second call: LLM requests to read a file
			sendSSEEvent(w, "message_start", `{"type":"message_start","message":{"id":"msg_2","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":200,"output_tokens":0}}}`)
			sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_2","name":"read_file","input":null}}`)
			sendSSEEvent(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":\"main.go\\"}}`)
			sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
			sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":30}}`)
			sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)

		case 3:
			// Third call: LLM provides final response (large response to test streaming)
			sendSSEEvent(w, "message_start", `{"type":"message_start","message":{"id":"msg_3","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":500,"output_tokens":0}}}`)
			sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)

			// Send content in multiple chunks to simulate streaming
			chunks := []string{
				"Based on my analysis of the codebase, here is a comprehensive summary:\n\n",
				"## Project Structure\n",
				"The project follows a clean architecture with clear separation of concerns.\n\n",
				"## Main Components\n",
				"1. **Internal Package**: Contains core business logic\n",
				"2. **Agents Package**: Implements various analysis agents\n",
				"3. **LLM Package**: Provides LLM client interfaces\n\n",
				"## Key Findings\n",
				"- The codebase is well-organized and maintainable\n",
				"- Proper error handling is implemented throughout\n",
				"- The streaming implementation reduces memory usage by 90-95%\n",
				"- All agents use the LLMClient interface for consistency\n\n",
				"## Recommendations\n",
				"Continue using the streaming implementation for all LLM calls.",
			}

			for _, chunk := range chunks {
				sendSSEEvent(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s"}}`, chunk))
			}

			sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
			sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":150}}`)
			sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)

		default:
			t.Errorf("Unexpected call count: %d", callCount)
		}

		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	// Create Anthropic client
	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	ctx := context.Background()

	// Call 1: Simulate agent requesting to list files
	resp1, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a code analyzer",
		Messages: []Message{
			{Role: "user", Content: "Analyze this repository"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 1 failed: %v", err)
	}

	if len(resp1.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp1.ToolCalls))
	}

	if resp1.ToolCalls[0].Name != "list_files" {
		t.Errorf("Expected tool name 'list_files', got '%s'", resp1.ToolCalls[0].Name)
	}

	// Call 2: Simulate agent requesting to read a file (with tool result)
	resp2, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a code analyzer",
		Messages: []Message{
			{Role: "user", Content: "Analyze this repository"},
			{Role: "assistant", Content: "", ToolCalls: resp1.ToolCalls},
			{Role: "tool", Content: `{"files": ["main.go", "go.mod"]}`, ToolID: "toolu_1"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 2 failed: %v", err)
	}

	if len(resp2.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp2.ToolCalls))
	}

	if resp2.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected tool name 'read_file', got '%s'", resp2.ToolCalls[0].Name)
	}

	// Call 3: Simulate agent providing final analysis (large response)
	resp3, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a code analyzer",
		Messages: []Message{
			{Role: "user", Content: "Analyze this repository"},
			{Role: "assistant", Content: "", ToolCalls: resp1.ToolCalls},
			{Role: "tool", Content: `{"files": ["main.go", "go.mod"]}`, ToolID: "toolu_1"},
			{Role: "assistant", Content: "", ToolCalls: resp2.ToolCalls},
			{Role: "tool", Content: "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}", ToolID: "toolu_2"},
		},
		MaxTokens:   200,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 3 failed: %v", err)
	}

	// Verify large response was received correctly
	if len(resp3.Content) < 500 {
		t.Errorf("Expected large response (>500 chars), got %d chars", len(resp3.Content))
	}

	if !strings.Contains(resp3.Content, "Project Structure") {
		t.Errorf("Expected response to contain 'Project Structure'")
	}

	if !strings.Contains(resp3.Content, "streaming implementation") {
		t.Errorf("Expected response to contain 'streaming implementation'")
	}

	// Verify token counts
	if resp3.Usage.InputTokens != 500 {
		t.Errorf("Expected 500 input tokens, got %d", resp3.Usage.InputTokens)
	}

	if resp3.Usage.OutputTokens != 150 {
		t.Errorf("Expected 150 output tokens, got %d", resp3.Usage.OutputTokens)
	}

	// Verify we made exactly 3 calls
	if callCount != 3 {
		t.Errorf("Expected 3 HTTP calls, got %d", callCount)
	}
}

// TestOpenAIStreaming_AgentWorkflow tests OpenAI streaming with a simulated agent workflow
func TestOpenAIStreaming_AgentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Check for stream parameter
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		bodyStr := string(body[:n])
		if !strings.Contains(bodyStr, `"stream":true`) {
			t.Errorf("Expected stream=true in request body")
		}

		w.Header().Set("Content-Type", "text/event-stream")

		// Simulate agent workflow
		switch callCount {
		case 1:
			// First call: LLM requests tool use
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"search\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"query\\\":\\\"golang\\\"}\"}}]},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}")
			fmt.Fprintln(w, "data: [DONE]")

		case 2:
			// Second call: LLM provides final response
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Here\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" is\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" the\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" analysis\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" result\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\".\"},\"finish_reason\":null}]}")
			fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-2\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
			fmt.Fprintln(w, "data: [DONE]")

		default:
			t.Errorf("Unexpected call count: %d", callCount)
		}

		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	client := NewOpenAIClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, nil)

	ctx := context.Background()

	// Call 1: Request tool use
	resp1, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Search for golang information"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 1 failed: %v", err)
	}

	if len(resp1.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp1.ToolCalls))
	}

	// Call 2: Get final response after tool result
	resp2, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Search for golang information"},
			{Role: "assistant", Content: "", ToolCalls: resp1.ToolCalls},
			{Role: "tool", Content: `{"results": ["Go is a programming language"]}`, ToolID: "call_1"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 2 failed: %v", err)
	}

	if resp2.Content != "Here is the analysis result." {
		t.Errorf("Expected 'Here is the analysis result.', got '%s'", resp2.Content)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 HTTP calls, got %d", callCount)
	}
}

// TestGeminiStreaming_AgentWorkflow tests Gemini streaming with a simulated agent workflow
func TestGeminiStreaming_AgentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Verify streaming endpoint
		if !strings.Contains(r.URL.Path, "streamGenerateContent") {
			t.Errorf("Expected streaming endpoint, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")

		switch callCount {
		case 1:
			// First call: Function call
			fmt.Fprintln(w, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"functionCall":{"name":"list_files","args":{"path":"."}}}]}}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"totalTokenCount":120}}`)

		case 2:
			// Second call: Final response (streamed across multiple chunks)
			fmt.Fprintln(w, `{"candidates":[{"finishReason":null,"content":{"parts":[{"text":"Analysis"}]}}],"usageMetadata":{"promptTokenCount":200,"candidatesTokenCount":5,"totalTokenCount":205}}`)
			fmt.Fprintln(w, `{"candidates":[{"finishReason":null,"content":{"parts":[{"text":" complete"}]}}],"usageMetadata":{"promptTokenCount":200,"candidatesTokenCount":10,"totalTokenCount":210}}`)
			fmt.Fprintln(w, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"."}]}}],"usageMetadata":{"promptTokenCount":200,"candidatesTokenCount":15,"totalTokenCount":215}}`)

		default:
			t.Errorf("Unexpected call count: %d", callCount)
		}

		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	client := NewGeminiClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-2.0-flash-exp",
	}, nil)

	ctx := context.Background()

	// Call 1: Request function call
	resp1, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "List files in current directory"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 1 failed: %v", err)
	}

	if len(resp1.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp1.ToolCalls))
	}

	if resp1.ToolCalls[0].Name != "list_files" {
		t.Errorf("Expected tool name 'list_files', got '%s'", resp1.ToolCalls[0].Name)
	}

	// Call 2: Get final response after function result
	resp2, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "List files in current directory"},
			{Role: "assistant", Content: "", ToolCalls: resp1.ToolCalls},
			{Role: "tool", Content: `{"files": ["main.go"]}`, ToolID: "list_files"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Call 2 failed: %v", err)
	}

	if resp2.Content != "Analysis complete." {
		t.Errorf("Expected 'Analysis complete.', got '%s'", resp2.Content)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 HTTP calls, got %d", callCount)
	}
}

// TestStreamingLargeResponse tests that large responses are handled correctly with streaming
// This is particularly important for validating the memory efficiency improvements
func TestAnthropicStreaming_LargeResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send message_start
		sendSSEEvent(w, "message_start", `{"type":"message_start","message":{"id":"msg_1","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":0}}}`)

		// Send content_block_start
		sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)

		// Send a large response in 50 chunks (simulating ~50KB response)
		for i := 0; i < 50; i++ {
			chunk := strings.Repeat(fmt.Sprintf("Chunk %d: ", i), 20) // ~200 bytes per chunk
			sendSSEEvent(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s\n"}}`, chunk))
		}

		// Send completion events
		sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":1000}}`)
		sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)

		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	ctx := context.Background()

	resp, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant",
		Messages: []Message{
			{Role: "user", Content: "Generate a large response"},
		},
		MaxTokens:   2000,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify we received a large response
	if len(resp.Content) < 10000 {
		t.Errorf("Expected large response (>10KB), got %d bytes", len(resp.Content))
	}

	// Verify content integrity
	if !strings.Contains(resp.Content, "Chunk 0:") {
		t.Errorf("Expected response to contain 'Chunk 0:'")
	}

	if !strings.Contains(resp.Content, "Chunk 49:") {
		t.Errorf("Expected response to contain 'Chunk 49:'")
	}

	// Verify token counts
	if resp.Usage.OutputTokens != 1000 {
		t.Errorf("Expected 1000 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

// TestStreamingConcurrentRequests tests that multiple concurrent streaming requests work correctly
// This simulates the real-world scenario where multiple agents run concurrently
func TestAnthropicStreaming_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")

		// Send a simple streaming response
		sendSSEEvent(w, "message_start", fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_%d","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`, requestCount))
		sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		sendSSEEvent(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Response %d"}}`, requestCount))
		sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`)
		sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)

		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	ctx := context.Background()

	// Launch 5 concurrent requests
	numConcurrent := 5
	results := make(chan string, numConcurrent)
	errors := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			resp, err := client.GenerateCompletion(ctx, CompletionRequest{
				SystemPrompt: "You are a helpful assistant",
				Messages: []Message{
					{Role: "user", Content: fmt.Sprintf("Request %d", id)},
				},
				MaxTokens:   100,
				Temperature: 0.0,
			})
			if err != nil {
				errors <- err
				return
			}
			results <- resp.Content
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		select {
		case <-results:
			successCount++
		case err := <-errors:
			t.Errorf("Concurrent request failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Errorf("Timeout waiting for concurrent requests")
		}
	}

	// Verify all requests succeeded
	if successCount != numConcurrent {
		t.Errorf("Expected %d successful requests, got %d", numConcurrent, successCount)
	}

	// Verify server received all requests
	if requestCount != numConcurrent {
		t.Errorf("Expected %d server requests, got %d", numConcurrent, requestCount)
	}
}

// Helper function to send SSE events in proper SSE format
// SSE format requires:
// event: <event_type>
// data: <json_data>
// <empty line>
func sendSSEEvent(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n", data)
	fmt.Fprintln(w)
}

// TestStreamingWithToolLargeArguments tests streaming with large tool arguments
// This validates that partial JSON accumulation works correctly for large tool inputs
func TestAnthropicStreaming_ToolLargeArguments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send message_start
		sendSSEEvent(w, "message_start", `{"type":"message_start","message":{"id":"msg_1","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":0}}}`)

		// Send content_block_start for tool_use
		sendSSEEvent(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"analyze_files","input":null}}`)

		// Send large tool arguments in multiple chunks (simulating analysis of many files)
		argChunks := []string{
			`{"files":[`,
			`"file1.go","file2.go","file3.go","file4.go","file5.go",`,
			`"file6.go","file7.go","file8.go","file9.go","file10.go",`,
			`"file11.go","file12.go","file13.go","file14.go","file15.go",`,
			`"file16.go","file17.go","file18.go","file19.go","file20.go"`,
			`],"depth":5}`,
		}

		for _, chunk := range argChunks {
			// Escape quotes for JSON
			escapedChunk := strings.ReplaceAll(chunk, `"`, `\"`)
			sendSSEEvent(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"%s"}}`, escapedChunk))
		}

		// Send completion events
		sendSSEEvent(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		sendSSEEvent(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":50}}`)
		sendSSEEvent(w, "message_stop", `{"type":"message_stop"}`)
	}))
	defer server.Close()

	client := NewAnthropicClient(config.LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}, nil)

	ctx := context.Background()

	resp, err := client.GenerateCompletion(ctx, CompletionRequest{
		SystemPrompt: "You are a code analyzer",
		Messages: []Message{
			{Role: "user", Content: "Analyze all Go files"},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify tool call
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	toolCall := resp.ToolCalls[0]
	if toolCall.Name != "analyze_files" {
		t.Errorf("Expected tool name 'analyze_files', got '%s'", toolCall.Name)
	}

	// Verify large arguments were accumulated correctly
	files, ok := toolCall.Arguments["files"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'files' argument to be a list")
	}

	if len(files) != 20 {
		t.Errorf("Expected 20 files, got %d", len(files))
	}

	// Verify depth argument
	depth, ok := toolCall.Arguments["depth"].(float64)
	if !ok || int(depth) != 5 {
		t.Errorf("Expected depth=5, got %v", toolCall.Arguments["depth"])
	}
}
