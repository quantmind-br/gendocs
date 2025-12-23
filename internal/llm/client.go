package llm

import (
	"context"
)

// Message represents a chat message
type Message struct {
	Role    string // "system", "user", "assistant", "tool"
	Content string
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	Name      string
	Arguments map[string]interface{}
}

// CompletionRequest is a request for LLM completion
type CompletionRequest struct {
	SystemPrompt string
	Messages     []Message
	Tools        []ToolDefinition
	MaxTokens    int
	Temperature  float64
}

// CompletionResponse is the response from LLM
type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
	Usage     TokenUsage
}

// TokenUsage tracks token usage
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ToolDefinition defines a tool for the LLM
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// LLMClient is the interface for LLM providers
type LLMClient interface {
	// GenerateCompletion generates a completion from the LLM
	GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// SupportsTools returns true if the client supports tool calling
	SupportsTools() bool

	// GetProvider returns the provider name
	GetProvider() string
}

// BaseLLMClient provides common functionality for all LLM clients
type BaseLLMClient struct {
	retryClient *RetryClient
}

// NewBaseLLMClient creates a new base LLM client
func NewBaseLLMClient(retryClient *RetryClient) *BaseLLMClient {
	return &BaseLLMClient{
		retryClient: retryClient,
	}
}
