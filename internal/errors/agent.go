package errors

import (
	"fmt"
)

// AgentError is the base error for all agent-related errors
type AgentError struct {
	*AIDocGenError
}

// NewAgentError creates a new agent error
func NewAgentError(message string) *AgentError {
	return &AgentError{
		AIDocGenError: &AIDocGenError{
			Message:  message,
			ExitCode: ExitAgentError,
		},
	}
}

// LLMConnectionError is raised when connection to LLM provider fails
type LLMConnectionError struct {
	*AIDocGenError
}

// NewLLMConnectionError creates a new LLM connection error
func NewLLMConnectionError(provider string, cause error) *LLMConnectionError {
	return &LLMConnectionError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Failed to connect to LLM provider: %s", provider),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "LLM API Call",
				Component: "LLM Client",
				Details: map[string]interface{}{
					"provider": provider,
				},
				Suggestions: []string{
					"Check your internet connection",
					"Verify the API endpoint is accessible",
					"Check if the API key is valid",
					"Try again later (service may be unavailable)",
				},
				Recoverable: true,
			},
			ExitCode: ExitLLMError,
		},
	}
}

// LLMResponseError is raised when LLM response is invalid or cannot be parsed
type LLMResponseError struct {
	*AIDocGenError
}

// NewLLMResponseError creates a new LLM response error
func NewLLMResponseError(provider, reason string) *LLMResponseError {
	return &LLMResponseError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Invalid response from LLM provider: %s", provider),
			Context: &ErrorContext{
				Operation: "Parsing LLM Response",
				Component: "LLM Client",
				Details: map[string]interface{}{
					"provider": provider,
					"reason":   reason,
				},
				Suggestions: []string{
					"Check if the model name is correct",
					"Try a different model",
					"Report this issue if it persists",
				},
				Recoverable: true,
			},
			ExitCode: ExitLLMError,
		},
	}
}

// AgentTimeoutError is raised when an agent execution times out
type AgentTimeoutError struct {
	*AIDocGenError
}

// NewAgentTimeoutError creates a new agent timeout error
func NewAgentTimeoutError(agentName string, timeoutSeconds int) *AgentTimeoutError {
	return &AgentTimeoutError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Agent '%s' timed out after %d seconds", agentName, timeoutSeconds),
			Context: &ErrorContext{
				Operation: "Agent Execution",
				Component: agentName,
				Details: map[string]interface{}{
					"timeout_seconds": timeoutSeconds,
				},
				Suggestions: []string{
					"Increase the timeout via LLM_TIMEOUT environment variable",
					"Try reducing the size of the codebase",
					"Try a faster model",
				},
				Recoverable: false,
			},
			ExitCode: ExitAgentError,
		},
	}
}

// ToolExecutionError is raised when a tool execution fails
type ToolExecutionError struct {
	*AIDocGenError
}

// NewToolExecutionError creates a new tool execution error
func NewToolExecutionError(toolName string, cause error) *ToolExecutionError {
	return &ToolExecutionError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Tool '%s' execution failed", toolName),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "Tool Execution",
				Component: toolName,
				Details: map[string]interface{}{
					"tool": toolName,
				},
				Suggestions: []string{
					"Check if the file/directory exists",
					"Verify file permissions",
					"Check the error details above",
				},
				Recoverable: true,
			},
			ExitCode: ExitAgentError,
		},
	}
}
