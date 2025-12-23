package errors

import (
	"fmt"
)

// ValidationError is the base error for all validation-related errors
type ValidationError struct {
	*AIDocGenError
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		AIDocGenError: &AIDocGenError{
			Message:  message,
			ExitCode: ExitValidationError,
		},
	}
}

// MissingFileError is raised when a required file is not found
type MissingFileError struct {
	*AIDocGenError
}

// NewMissingFileError creates a new missing file error
func NewMissingFileError(filePath string) *MissingFileError {
	return &MissingFileError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Required file not found: %s", filePath),
			Context: &ErrorContext{
				Operation: "File Validation",
				Component: "Filesystem",
				Details: map[string]interface{}{
					"file_path": filePath,
				},
				Suggestions: []string{
					"Check that the file exists",
					"Verify the file path is correct",
					"Run analysis first to generate the file",
				},
				Recoverable: false,
			},
			ExitCode: ExitValidationError,
		},
	}
}

// InvalidPathError is raised when a path is invalid
type InvalidPathError struct {
	*AIDocGenError
}

// NewInvalidPathError creates a new invalid path error
func NewInvalidPathError(path string, reason string) *InvalidPathError {
	return &InvalidPathError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Invalid path: %s", path),
			Context: &ErrorContext{
				Operation: "Path Validation",
				Component: "Filesystem",
				Details: map[string]interface{}{
					"path":   path,
					"reason": reason,
				},
				Suggestions: []string{
					"Check that the path exists",
					"Verify the path is a valid directory",
					"Use an absolute path if relative path fails",
				},
				Recoverable: false,
			},
			ExitCode: ExitValidationError,
		},
	}
}

// OutputValidationError is raised when output validation fails
type OutputValidationError struct {
	*AIDocGenError
}

// NewOutputValidationError creates a new output validation error
func NewOutputValidationError(missingFiles []string) *OutputValidationError {
	return &OutputValidationError{
		AIDocGenError: &AIDocGenError{
			Message: "Output validation failed: expected files were not generated",
			Context: &ErrorContext{
				Operation: "Output Validation",
				Component: "Validation",
				Details: map[string]interface{}{
					"missing_files": missingFiles,
				},
				Suggestions: []string{
					"Check if LLM API calls succeeded",
					"Review error logs for individual agent failures",
					"Try running with --debug flag for more details",
				},
				Recoverable: false,
			},
			ExitCode: ExitValidationError,
		},
	}
}
