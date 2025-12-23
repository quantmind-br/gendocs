package errors

import (
	"fmt"
)

// GitLabError is the base error for all GitLab-related errors
type GitLabError struct {
	*AIDocGenError
}

// NewGitLabError creates a new GitLab error
func NewGitLabError(message string) *GitLabError {
	return &GitLabError{
		AIDocGenError: &AIDocGenError{
			Message:  message,
			ExitCode: ExitGitLabError,
		},
	}
}

// GitLabAuthError is raised when GitLab authentication fails
type GitLabAuthError struct {
	*AIDocGenError
}

// NewGitLabAuthError creates a new GitLab authentication error
func NewGitLabAuthError(cause error) *GitLabAuthError {
	return &GitLabAuthError{
		AIDocGenError: &AIDocGenError{
			Message: "GitLab authentication failed",
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "GitLab Authentication",
				Component: "GitLab Client",
				Suggestions: []string{
					"Verify GITLAB_OAUTH_TOKEN is set correctly",
					"Check if the token has not expired",
					"Ensure token has required permissions (api, read_repository)",
					"Generate a new token at GitLab user settings > access tokens",
				},
				Recoverable: false,
			},
			ExitCode: ExitGitLabError,
		},
	}
}

// GitLabAPIError is raised when GitLab API call fails
type GitLabAPIError struct {
	*AIDocGenError
}

// NewGitLabAPIError creates a new GitLab API error
func NewGitLabAPIError(operation, statusCode string, cause error) *GitLabAPIError {
	return &GitLabAPIError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("GitLab API error during %s", operation),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "GitLab API Call",
				Component: "GitLab Client",
				Details: map[string]interface{}{
					"operation":   operation,
					"status_code": statusCode,
				},
				Suggestions: []string{
					"Check GitLab API URL is correct",
					"Verify the project/group exists",
					"Check API rate limits",
					"Try again later",
				},
				Recoverable: true,
			},
			ExitCode: ExitGitLabError,
		},
	}
}

// GitCloneError is raised when git clone fails
type GitCloneError struct {
	*AIDocGenError
}

// NewGitCloneError creates a new git clone error
func NewGitCloneError(repoURL, reason string, cause error) *GitCloneError {
	return &GitCloneError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Failed to clone repository: %s", repoURL),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "Git Clone",
				Component: "Git",
				Details: map[string]interface{}{
					"repo_url": repoURL,
					"reason":   reason,
				},
				Suggestions: []string{
					"Check if the repository URL is correct",
					"Verify you have access to the repository",
					"Check git is installed and accessible",
					"Ensure sufficient disk space",
				},
				Recoverable: false,
			},
			ExitCode: ExitGitLabError,
		},
	}
}
