package errors

import (
	"fmt"
	"strings"
)

// ErrorContext provides rich error information for user-friendly error messages
type ErrorContext struct {
	Operation   string                 // The operation that failed
	Component   string                 // The component that failed
	Details     map[string]interface{} // Additional details about the error
	Suggestions []string               // Actionable suggestions for the user
	Recoverable bool                   // Whether the error is recoverable
	RetryCount  int                    // Current retry count
	MaxRetries  int                    // Maximum retries allowed
}

// Format returns a formatted string representation of the error context
func (ec *ErrorContext) Format() string {
	var sb strings.Builder

	if ec.Operation != "" || ec.Component != "" {
		sb.WriteString("\nWhat happened:\n")
		if ec.Operation != "" && ec.Component != "" {
			sb.WriteString(fmt.Sprintf("  %s failed in %s.\n", ec.Operation, ec.Component))
		} else if ec.Operation != "" {
			sb.WriteString(fmt.Sprintf("  %s failed.\n", ec.Operation))
		} else if ec.Component != "" {
			sb.WriteString(fmt.Sprintf("  Failure in %s.\n", ec.Component))
		}
	}

	if len(ec.Details) > 0 {
		sb.WriteString("\nDetails:\n")
		for key, value := range ec.Details {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
		}
	}

	if len(ec.Suggestions) > 0 {
		sb.WriteString("\nWhat you can do:\n")
		for i, suggestion := range ec.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
	}

	if ec.Recoverable {
		sb.WriteString(fmt.Sprintf("\nRecoverable: Yes (retry %d/%d)\n", ec.RetryCount, ec.MaxRetries))
	}

	return sb.String()
}
