package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/user/gendocs/internal/config"
)

// OpenAIClient implements LLMClient for OpenAI-compatible APIs
type OpenAIClient struct {
	*BaseLLMClient
	apiKey  string
	baseURL string
	model   string
}

// openaiRequest represents the request body for OpenAI API
type openaiRequest struct {
	Model       string         `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
	Tools       []openaiTool   `json:"tools,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

// openaiMessage represents a message in OpenAI format
type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// openaiTool represents a tool definition in OpenAI format
type openaiTool struct {
	Type     string              `json:"type"`
	Function openaiToolFunction  `json:"function"`
}

// openaiToolFunction represents tool function parameters
type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// openaiToolCall represents a tool call in OpenAI format
type openaiToolCall struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Function openaiToolCallFunc    `json:"function"`
}

// openaiToolCallFunc represents function call details
type openaiToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openaiResponse represents the response from OpenAI API
type openaiResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openaiChoice     `json:"choices"`
	Usage   openaiUsage        `json:"usage"`
	Error   *openaiErrorDetail `json:"error,omitempty"`
}

// openaiChoice represents a choice in the response
type openaiChoice struct {
	Index        int              `json:"index"`
	Message      openaiMessage    `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

// openaiUsage represents token usage
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openaiErrorDetail represents an error from OpenAI
type openaiErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// openaiStreamChunk represents a single chunk in OpenAI's streaming response
type openaiStreamChunk struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []openaiStreamChoice  `json:"choices"`
}

// openaiStreamChoice represents a choice in streaming chunks
type openaiStreamChoice struct {
	Index        int                  `json:"index"`
	Delta        openaiStreamDelta    `json:"delta"`
	FinishReason string               `json:"finish_reason,omitempty"`
}

// openaiStreamDelta represents the delta field in streaming chunks
type openaiStreamDelta struct {
	Content   string                `json:"content,omitempty"`
	Role      string                `json:"role,omitempty"`
	ToolCalls []openaiToolCallDelta `json:"tool_calls,omitempty"`
}

// openaiToolCallDelta represents a tool call in the delta
type openaiToolCallDelta struct {
	Index    int                    `json:"index"`
	Function openaiToolCallFuncDelta `json:"function,omitempty"`
}

// openaiToolCallFuncDelta represents function call details in streaming
type openaiToolCallFuncDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// openaiAccumulator builds CompletionResponse from OpenAI streaming chunks
type openaiAccumulator struct {
	content    strings.Builder
	toolCalls  []openaiToolCall
	partialArgs map[int]strings.Builder // Accumulates arguments by index
	usage      openaiUsage
	finishReason string
	complete   bool
}

// newOpenAIAccumulator creates a new accumulator
func newOpenAIAccumulator() *openaiAccumulator {
	return &openaiAccumulator{
		partialArgs: make(map[int]strings.Builder),
	}
}

// HandleChunk processes a single streaming chunk
func (a *openaiAccumulator) HandleChunk(data []byte) error {
	// Check for [DONE] marker
	if IsSSEDone(data) {
		a.complete = true
		return nil
	}

	var chunk openaiStreamChunk
	if err := ParseSSEData(data, &chunk); err != nil {
		return fmt.Errorf("failed to parse chunk: %w", err)
	}

	if len(chunk.Choices) == 0 {
		return nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// Accumulate content
	if delta.Content != "" {
		a.content.WriteString(delta.Content)
	}

	// Accumulate tool calls
	for _, tc := range delta.ToolCalls {
		idx := tc.Index

		// Ensure toolCalls slice is large enough
		for len(a.toolCalls) <= idx {
			a.toolCalls = append(a.toolCalls, openaiToolCall{
				Type: "function",
			})
		}

		// Set ID if not set
		if a.toolCalls[idx].ID == "" {
			a.toolCalls[idx].ID = chunk.ID + "-" + fmt.Sprintf("%d", idx)
		}

		// Accumulate function name
		if tc.Function.Name != "" {
			a.toolCalls[idx].Function.Name = tc.Function.Name
		}

		// Accumulate arguments incrementally
		if tc.Function.Arguments != "" {
			if _, exists := a.partialArgs[idx]; !exists {
				a.partialArgs[idx] = strings.Builder{}
			}
			a.partialArgs[idx].WriteString(tc.Function.Arguments)
		}
	}

	// Check for completion
	if choice.FinishReason != "" {
		a.finishReason = choice.FinishReason

		// Parse accumulated tool arguments
		for idx, argBuilder := range a.partialArgs {
			if idx < len(a.toolCalls) && argBuilder.Len() > 0 {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(argBuilder.String()), &args); err != nil {
					return fmt.Errorf("failed to parse tool %d arguments: %w", idx, err)
				}
				a.toolCalls[idx].Function.Arguments = argBuilder.String()
			}
		}
	}

	return nil
}

// Build constructs the final CompletionResponse
func (a *openaiAccumulator) Build() CompletionResponse {
	result := CompletionResponse{
		Content: a.content.String(),
		Usage: TokenUsage{
			InputTokens:  a.usage.PromptTokens,
			OutputTokens: a.usage.CompletionTokens,
			TotalTokens:  a.usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(a.toolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(a.toolCalls))
		for i, tc := range a.toolCalls {
			var args map[string]interface{}
			if tc.Function.Arguments != "" {
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}

			result.ToolCalls[i] = ToolCall{
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
	}

	return result
}

// IsComplete returns true when stream is complete
func (a *openaiAccumulator) IsComplete() bool {
	return a.complete || a.finishReason != ""
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg config.LLMConfig, retryClient *RetryClient) *OpenAIClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIClient{
		BaseLLMClient: NewBaseLLMClient(retryClient),
		apiKey:        cfg.APIKey,
		baseURL:       baseURL,
		model:         cfg.Model,
	}
}

// GenerateCompletion generates a completion from OpenAI
func (c *OpenAIClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	// Convert to OpenAI format
	oaReq := c.convertRequest(req)

	jsonData, err := json.Marshal(oaReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute with retry
	resp, err := c.retryClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse streaming response
	return c.parseStreamingResponse(resp.Body)
}

// parseStreamingResponse parses OpenAI's SSE stream and builds the response
func (c *OpenAIClient) parseStreamingResponse(body io.ReadCloser) (CompletionResponse, error) {
	parser := NewSSEParser(body)
	accumulator := newOpenAIAccumulator()

	for {
		event, err := parser.NextEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return CompletionResponse{}, fmt.Errorf("stream parsing error: %w", err)
		}

		// Handle event data (OpenAI doesn't use event types, all chunks are in data field)
		if err := accumulator.HandleChunk(event.Data); err != nil {
			return CompletionResponse{}, fmt.Errorf("chunk handling error: %w", err)
		}

		// Check if stream is complete
		if accumulator.IsComplete() {
			break
		}
	}

	return accumulator.Build(), nil
}

// SupportsTools returns true
func (c *OpenAIClient) SupportsTools() bool {
	return true
}

// GetProvider returns the provider name
func (c *OpenAIClient) GetProvider() string {
	return "openai"
}

// convertRequest converts internal request to OpenAI format
func (c *OpenAIClient) convertRequest(req CompletionRequest) openaiRequest {
	messages := []openaiMessage{}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, openaiMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Add messages
	for _, msg := range req.Messages {
		messages = append(messages, openaiMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	oaReq := openaiRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		oaReq.Tools = make([]openaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			oaReq.Tools[i] = openaiTool{
				Type: "function",
				Function: openaiToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	oaReq.Stream = true // Enable streaming response

	return oaReq
}

// convertResponse converts OpenAI response to internal format
func (c *OpenAIClient) convertResponse(resp openaiResponse) CompletionResponse {
	if len(resp.Choices) == 0 {
		return CompletionResponse{
			Usage: TokenUsage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
				TotalTokens:  resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]
	result := CompletionResponse{
		Content: choice.Message.Content,
		Usage: TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			// Parse arguments JSON string
			var args map[string]interface{}
			if tc.Function.Arguments != "" {
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}

			result.ToolCalls[i] = ToolCall{
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
	}

	return result
}