package errors

import (
	"fmt"
)

// HandlerError is the base error for all handler-related errors
type HandlerError struct {
	*AIDocGenError
}

// NewHandlerError creates a new handler error
func NewHandlerError(message string) *HandlerError {
	return &HandlerError{
		AIDocGenError: &AIDocGenError{
			Message:  message,
			ExitCode: ExitGeneralError,
		},
	}
}

// AnalysisError is raised when analysis fails
type AnalysisError struct {
	*AIDocGenError
}

// NewAnalysisError creates a new analysis error
func NewAnalysisError(reason string, cause error) *AnalysisError {
	return &AnalysisError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Analysis failed: %s", reason),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "Codebase Analysis",
				Component: "AnalyzerAgent",
				Suggestions: []string{
					"Check if the repository path is valid",
					"Verify LLM configuration",
					"Try with --debug flag for more information",
					"Check if any agents were excluded",
				},
				Recoverable: false,
			},
			ExitCode: ExitAgentError,
		},
	}
}

// DocumentationError is raised when documentation generation fails
type DocumentationError struct {
	*AIDocGenError
}

// NewDocumentationError creates a new documentation error
func NewDocumentationError(docType string, cause error) *DocumentationError {
	return &DocumentationError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Failed to generate %s documentation", docType),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "Documentation Generation",
				Component: docType + "Agent",
				Suggestions: []string{
					"Ensure analysis has been run first",
					"Check that analysis files exist in .ai/docs/",
					"Verify LLM configuration",
				},
				Recoverable: false,
			},
			ExitCode: ExitAgentError,
		},
	}
}

// CronjobError is raised when cronjob execution fails
type CronjobError struct {
	*AIDocGenError
}

// NewCronjobError creates a new cronjob error
func NewCronjobError(reason string, cause error) *CronjobError {
	return &CronjobError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Cronjob failed: %s", reason),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "GitLab Cronjob",
				Component: "CronjobHandler",
				Suggestions: []string{
					"Check GitLab API credentials",
					"Verify group project ID",
					"Check GitLab API URL",
					"Review cronjob logs",
				},
				Recoverable: false,
			},
			ExitCode: ExitGeneralError,
		},
	}
}
