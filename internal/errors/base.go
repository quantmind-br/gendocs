package errors

import (
	"fmt"
)

// AIDocGenError is the base error type for all application errors
type AIDocGenError struct {
	Message  string        // Human-readable error message
	Context  *ErrorContext // Rich error context
	Cause    error         // Underlying error (for wrapping)
	ExitCode ExitCode      // Exit code for CLI
}

// Error returns the error message with cause if present
func (e *AIDocGenError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause
func (e *AIDocGenError) Unwrap() error {
	return e.Cause
}

// GetUserMessage returns a user-friendly error message with context
func (e *AIDocGenError) GetUserMessage() string {
	msg := fmt.Sprintf("ERROR: %s", e.Message)

	if e.Cause != nil {
		msg += fmt.Sprintf("\nCause: %v", e.Cause)
	}

	if e.Context != nil {
		msg += e.Context.Format()
	}

	return msg
}

// NewError creates a new AIDocGenError with the given message and exit code
func NewError(message string, exitCode ExitCode) *AIDocGenError {
	return &AIDocGenError{
		Message:  message,
		ExitCode: exitCode,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(cause error, message string, exitCode ExitCode) *AIDocGenError {
	return &AIDocGenError{
		Message:  message,
		Cause:    cause,
		ExitCode: exitCode,
	}
}

// WrapErrorWithContext wraps an error with full context
func WrapErrorWithContext(cause error, message string, exitCode ExitCode, context *ErrorContext) *AIDocGenError {
	return &AIDocGenError{
		Message:  message,
		Context:  context,
		Cause:    cause,
		ExitCode: exitCode,
	}
}
