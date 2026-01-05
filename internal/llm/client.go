package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// doHTTPRequest executes an HTTP request with JSON payload and returns the response.
// It handles JSON marshaling, request creation, header setting, and execution with retry.
// The caller is responsible for closing the response body and handling status codes.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - method: HTTP method (e.g., "POST")
//   - url: Full URL for the request
//   - headers: Custom headers to apply (Content-Type is set automatically)
//   - payload: Request body to marshal to JSON (can be nil for no body)
//
// Returns the HTTP response. Caller must close resp.Body.
func (b *BaseLLMClient) doHTTPRequest(
	ctx context.Context,
	method string,
	url string,
	headers map[string]string,
	payload interface{},
) (*http.Response, error) {
	var body *bytes.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	var httpReq *http.Request
	var err error
	if body != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, url, body)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := b.retryClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
