package tools

import (
	"context"
	"fmt"
)

// ModelRetryError is raised when a tool encounters a recoverable error
// This error type triggers a retry at the agent level
type ModelRetryError struct {
	Message string
}

func (e *ModelRetryError) Error() string {
	return e.Message
}

// Tool is the interface that all tools must implement
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Parameters returns the JSON schema for the tool's parameters
	Parameters() map[string]interface{}

	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// BaseTool provides common functionality for all tools
type BaseTool struct {
	MaxRetries int
}

// NewBaseTool creates a new base tool
func NewBaseTool(maxRetries int) BaseTool {
	return BaseTool{
		MaxRetries: maxRetries,
	}
}

// RetryableExecute executes a function with retry logic
func (bt *BaseTool) RetryableExecute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt < bt.MaxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is recoverable (ModelRetryError)
		if _, ok := err.(*ModelRetryError); !ok {
			// Not recoverable, return immediately
			return nil, err
		}

		// If it's the last attempt, don't wait
		if attempt == bt.MaxRetries-1 {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("tool failed after %d retries: %w", bt.MaxRetries, lastErr)
	}

	return nil, fmt.Errorf("tool failed after %d retries", bt.MaxRetries)
}
