package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/user/gendocs/internal/config"
)

// AnthropicClient implements LLMClient for Anthropic Claude
type AnthropicClient struct {
	*BaseLLMClient
	apiKey  string
	model   string
	baseURL string
}

// anthropicRequest represents the request body for Anthropic API
type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

// anthropicMessage represents a message in Anthropic format
type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

// anthropicContentBlock represents a content block
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// Tool use fields (flat when type=="tool_use")
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	// Tool result fields (flat when type=="tool_result")
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"` // Can be string for tool results
}

// anthropicTool represents a tool definition
type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// anthropicResponse represents the response from Anthropic API
type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
	Error      *anthropicError         `json:"error,omitempty"`
}

// anthropicUsage represents token usage
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicError represents an error from Anthropic
type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(cfg config.LLMConfig, retryClient *RetryClient) *AnthropicClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicClient{
		BaseLLMClient: NewBaseLLMClient(retryClient),
		apiKey:        cfg.APIKey,
		model:         cfg.Model,
		baseURL:       baseURL,
	}
}

// GenerateCompletion generates a completion from Anthropic
func (c *AnthropicClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	anReq := c.convertRequest(req)

	url := c.baseURL + "/v1/messages"
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := c.doHTTPRequest(ctx, "POST", url, headers, anReq)
	if err != nil {
		return CompletionResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var anResp anthropicResponse
		if err := json.Unmarshal(body, &anResp); err == nil && anResp.Error != nil {
			return CompletionResponse{}, fmt.Errorf("API error: %s", anResp.Error.Message)
		}
		return CompletionResponse{}, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return c.parseStreamingResponse(resp.Body)
}

// parseStreamingResponse parses Anthropic's SSE stream and builds the response
func (c *AnthropicClient) parseStreamingResponse(body io.ReadCloser) (CompletionResponse, error) {
	parser := NewSSEParser(body)
	accumulator := newAnthropicAccumulator()

	for {
		event, err := parser.NextEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return CompletionResponse{}, fmt.Errorf("stream parsing error: %w", err)
		}

		if err := accumulator.HandleEvent(event); err != nil {
			return CompletionResponse{}, fmt.Errorf("event handling error: %w", err)
		}

		if accumulator.IsComplete() {
			break
		}
	}

	if !accumulator.IsComplete() {
		return CompletionResponse{}, fmt.Errorf("stream ended unexpectedly")
	}

	return accumulator.Build(), nil
}

// anthropicAccumulator builds CompletionResponse from Anthropic streaming events
type anthropicAccumulator struct {
	content         strings.Builder
	toolCalls       []ToolCall
	currentTool     *ToolCall
	toolArgsBuilder strings.Builder
	usage           TokenUsage
	stopReason      string
	complete        bool
}

func newAnthropicAccumulator() *anthropicAccumulator {
	return &anthropicAccumulator{}
}

func (a *anthropicAccumulator) HandleEvent(event SSEEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	eventType, _ := data["type"].(string)
	switch eventType {
	case "message_start":
		if msg, ok := data["message"].(map[string]interface{}); ok {
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				if input, ok := usage["input_tokens"].(float64); ok {
					a.usage.InputTokens = int(input)
				}
				if output, ok := usage["output_tokens"].(float64); ok {
					a.usage.OutputTokens = int(output)
				}
			}
		}
	case "content_block_start":
		if block, ok := data["content_block"].(map[string]interface{}); ok {
			blockType, _ := block["type"].(string)
			if blockType == "tool_use" {
				name, _ := block["name"].(string)

				a.currentTool = &ToolCall{
					Name: name,
				}
				a.toolArgsBuilder.Reset()
			}
		}
	case "content_block_delta":
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			switch deltaType, _ := delta["type"].(string); deltaType {
			case "text_delta":
				if text, ok := delta["text"].(string); ok {
					a.content.WriteString(text)
				}
			case "input_json_delta":
				if partial, ok := delta["partial_json"].(string); ok && a.currentTool != nil {
					a.toolArgsBuilder.WriteString(partial)
				}
			}
		}
	case "content_block_stop":
		if a.currentTool != nil {
			if a.toolArgsBuilder.Len() > 0 {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(a.toolArgsBuilder.String()), &args); err != nil {
					return fmt.Errorf("failed to parse tool arguments: %w", err)
				}
				a.currentTool.Arguments = args
			}
			a.toolCalls = append(a.toolCalls, *a.currentTool)
			a.currentTool = nil
			a.toolArgsBuilder.Reset()
		}
	case "message_delta":
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			if output, ok := usage["output_tokens"].(float64); ok {
				a.usage.OutputTokens = int(output)
			}
		}
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			if stopReason, ok := delta["stop_reason"].(string); ok {
				a.stopReason = stopReason
			}
		}
	case "message_stop":
		a.complete = true
	case "error":
		if errMap, ok := data["error"].(map[string]interface{}); ok {
			msg, _ := errMap["message"].(string)
			return fmt.Errorf("API error: %s", msg)
		}
	}
	return nil
}

func (a *anthropicAccumulator) IsComplete() bool {
	return a.complete
}

func (a *anthropicAccumulator) Build() CompletionResponse {
	a.usage.TotalTokens = a.usage.InputTokens + a.usage.OutputTokens
	return CompletionResponse{
		Content:   a.content.String(),
		ToolCalls: a.toolCalls,
		Usage:     a.usage,
	}
}

// SupportsTools returns true
func (c *AnthropicClient) SupportsTools() bool {
	return true
}

// GetProvider returns the provider name
func (c *AnthropicClient) GetProvider() string {
	return "anthropic"
}

// convertRequest converts internal request to Anthropic format
func (c *AnthropicClient) convertRequest(req CompletionRequest) anthropicRequest {
	messages := []anthropicMessage{}

	for _, msg := range req.Messages {
		switch msg.Role {
		case "tool":
			messages = append(messages, anthropicMessage{
				Role: "user",
				Content: []anthropicContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolID,
						Content:   msg.Content,
					},
				},
			})
		case "assistant":
			var contentBlocks []anthropicContentBlock

			if msg.Content != "" {
				contentBlocks = append(contentBlocks, anthropicContentBlock{
					Type: "text",
					Text: msg.Content,
				})
			}

			for _, tc := range msg.ToolCalls {
				contentBlocks = append(contentBlocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.Name,
					Name:  tc.Name,
					Input: tc.Arguments,
				})
			}

			if len(contentBlocks) > 0 {
				messages = append(messages, anthropicMessage{
					Role:    "assistant",
					Content: contentBlocks,
				})
			}
		case "user":
			if msg.Content != "" {
				messages = append(messages, anthropicMessage{
					Role: "user",
					Content: []anthropicContentBlock{
						{Type: "text", Text: msg.Content},
					},
				})
			}
		}
	}

	if len(messages) == 0 {
		messages = append(messages, anthropicMessage{
			Role: "user",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "Analyze this codebase."},
			},
		})
	}

	var tools []anthropicTool
	if len(req.Tools) > 0 {
		tools = make([]anthropicTool, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = anthropicTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			}
		}
	}

	return anthropicRequest{
		Model:       c.model,
		Messages:    messages,
		System:      req.SystemPrompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Tools:       tools,
		Stream:      true,
	}
}
