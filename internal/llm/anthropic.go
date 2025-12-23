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

// AnthropicClient implements LLMClient for Anthropic Claude
type AnthropicClient struct {
	*BaseLLMClient
	apiKey string
	model  string
}

// anthropicRequest represents the request body for Anthropic API
type anthropicRequest struct {
	Model         string                  `json:"model"`
	Messages      []anthropicMessage      `json:"messages"`
	System        string                  `json:"system,omitempty"`
	MaxTokens     int                     `json:"max_tokens"`
	Temperature   float64                 `json:"temperature,omitempty"`
	Tools         []anthropicTool         `json:"tools,omitempty"`
	Stream        bool                    `json:"stream,omitempty"`
}

// anthropicMessage represents a message in Anthropic format
type anthropicMessage struct {
	Role    string                 `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

// anthropicContentBlock represents a content block
type anthropicContentBlock struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	ToolUse  *anthropicToolUseBlock `json:"tool_use,omitempty"`
	ToolResult *anthropicToolResultBlock `json:"tool_result,omitempty"`
}

// anthropicToolUseBlock represents a tool use call
type anthropicToolUseBlock struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Input    map[string]interface{} `json:"input"`
}

// anthropicToolResultBlock represents a tool result
type anthropicToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// anthropicTool represents a tool definition
type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// anthropicResponse represents the response from Anthropic API
type anthropicResponse struct {
	ID      string                `json:"id"`
	Type    string                `json:"type"`
	Role    string                `json:"role"`
	Content []anthropicContentBlock `json:"content"`
	StopReason string              `json:"stop_reason"`
	Usage   anthropicUsage        `json:"usage"`
	Error   *anthropicError       `json:"error,omitempty"`
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
	return &AnthropicClient{
		BaseLLMClient: NewBaseLLMClient(retryClient),
		apiKey:        cfg.APIKey,
		model:         cfg.Model,
	}
}

// GenerateCompletion generates a completion from Anthropic
func (c *AnthropicClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	// Convert to Anthropic format
	anReq := c.convertRequest(req)

	jsonData, err := json.Marshal(anReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := "https://api.anthropic.com/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Execute with retry
	resp, err := c.retryClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anResp anthropicResponse
	if err := json.Unmarshal(body, &anResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if anResp.Error != nil {
		return CompletionResponse{}, fmt.Errorf("API error: %s", anResp.Error.Message)
	}

	return c.convertResponse(anResp), nil
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
	// Build messages from internal format
	messages := []anthropicMessage{}

	// Convert internal messages to Anthropic format
	for _, msg := range req.Messages {
		if msg.Role == "tool" {
			// Tool result message
			messages = append(messages, anthropicMessage{
				Role: "user",
				Content: []anthropicContentBlock{
					{
						Type: "tool_result",
						ToolResult: &anthropicToolResultBlock{
							Content: msg.Content,
						},
					},
				},
			})
		} else if msg.Role == "assistant" {
			// Assistant message
			contentBlock := anthropicContentBlock{
				Type: "text",
				Text: msg.Content,
			}
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: []anthropicContentBlock{contentBlock},
			})
		}
	}

	// If no messages yet, add initial user message
	if len(messages) == 0 {
		messages = append(messages, anthropicMessage{
			Role: "user",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "Analyze this codebase."},
			},
		})
	}

	// Build tools
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
		Stream:      false,
	}
}

// convertResponse converts Anthropic response to internal format
func (c *AnthropicClient) convertResponse(resp anthropicResponse) CompletionResponse {
	result := CompletionResponse{
		Usage: TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	var textContent strings.Builder
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		if block.Type == "text" {
			textContent.WriteString(block.Text)
		} else if block.Type == "tool_use" && block.ToolUse != nil {
			toolCalls = append(toolCalls, ToolCall{
				Name:      block.ToolUse.Name,
				Arguments: block.ToolUse.Input,
			})
		}
	}

	result.Content = textContent.String()
	result.ToolCalls = toolCalls

	return result
}
