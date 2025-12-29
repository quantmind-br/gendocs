package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Message represents a chat message
type Message struct {
	Role      string     // "system", "user", "assistant", "tool"
	Content   string
	ToolID    string     // ID of the tool that was called (for role="tool")
	ToolCalls []ToolCall // Tool calls made by assistant (for role="assistant")
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	Name             string                 // Name of the tool to call
	Arguments        map[string]interface{} // Arguments for the tool
	RawFunctionCall  map[string]interface{} // Preserves complete function call data
	ThoughtSignature string                 // Required for Gemini 3 function calling
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
//   - method: HTTP method (e.g., "GET", "POST")
//   - url: Target URL for the request
//   - headers: Map of HTTP headers to set on the request
//   - body: Request body to marshal as JSON (can be nil for GET requests)
//
// Returns:
//   - []byte: Raw response body bytes for provider-specific parsing
//   - error: Wrapped error with context if any step fails
//
// Error handling:
//   - "failed to marshal request" - JSON marshaling failure
//   - "failed to create request" - HTTP request creation failure
//   - "request failed" - Request execution failure (including retry attempts)
//   - "failed to read response" - Response body reading failure
//   - "API error: status %d, body: %s" - Non-200 status code with response body
func (c *BaseLLMClient) doHTTPRequest(
	ctx context.Context,
	method string,
	url string,
	headers map[string]string,
	body interface{},
) ([]byte, error) {
	// Marshal request body to JSON
	var jsonData []byte
	if body != nil {
		var err error
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// Create HTTP request with context
	var bodyReader *bytes.Reader
	if jsonData != nil {
		bodyReader = bytes.NewReader(jsonData)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from map
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request with retry
	resp, err := c.retryClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}