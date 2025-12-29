package llm

import (
	"context"

	"github.com/user/gendocs/internal/llmtypes"
)

// Type aliases for backward compatibility
type Message = llmtypes.Message
type ToolCall = llmtypes.ToolCall
type CompletionRequest = llmtypes.CompletionRequest
type CompletionResponse = llmtypes.CompletionResponse
type TokenUsage = llmtypes.TokenUsage
type ToolDefinition = llmtypes.ToolDefinition

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
	// If no retry client provided, create a default one
	if retryClient == nil {
		retryClient = NewRetryClient(nil) // Uses default config
	}
	return &BaseLLMClient{
		retryClient: retryClient,
	}
}

// doHTTPRequest executes an HTTP request with retry and standard error handling.
// It handles JSON marshaling, request creation, header setting, execution with retry,
// response reading, and status code validation.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
