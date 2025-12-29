package cmd

import (
	"fmt"
	"os"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/tui"
)

// CommandContext holds common resources used by CLI commands.
// It is the return value of initialization helpers and centralizes
// the logging, progress display, and repository path configuration.
type CommandContext struct {
	// Logger is the configured logger for the command
	Logger *logging.Logger

	// ShowProgress indicates whether to show progress UI (true) or verbose output (false)
	ShowProgress bool

	// RepoPath is the path to the repository being processed
	RepoPath string
}

// InitLogger creates a configured logger for CLI commands.
// It encapsulates the common logger initialization pattern:
//   - Calculates logDir based on repoPath (.ai/logs or repoPath/.ai/logs)
//   - Determines showProgress from verbose flag (showProgress = !verbose)
//   - Creates Config with appropriate settings
//   - Creates and returns the logger
//
// Parameters:
//   - repoPath: path to the repository being processed ("." for current directory)
//   - debug: enables caller information in logs
//   - verbose: when true, console output is enabled and showProgress is false
//
// Returns the configured logger and any error during initialization.
// The caller is responsible for calling logger.Sync() when done.
func InitLogger(repoPath string, debug bool, verbose bool) (*logging.Logger, error) {
	// Calculate log directory based on repo path
	logDir := ".ai/logs"
	if repoPath != "." {
		logDir = repoPath + "/.ai/logs"
	}

	// Determine showProgress from verbose flag
	// When verbose is true, we show console output instead of progress UI
	showProgress := !verbose

	// Create logger configuration
	logCfg := &logging.Config{
		LogDir:         logDir,
		FileLevel:      logging.LevelFromString("info"),
		ConsoleLevel:   logging.LevelFromString("debug"),
		EnableCaller:   debug,
		ConsoleEnabled: !showProgress,
	}

	// Create and return logger
	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, nil
}

// LLMDefaults holds default values for LLM configuration.
// These values are used when environment variables are not set.
type LLMDefaults struct {
	// Retries is the number of retry attempts for LLM API calls (default: 2)
	Retries int

	// Timeout is the timeout in seconds for LLM API calls (default: 180)
	Timeout int

	// MaxTokens is the maximum number of tokens for LLM responses (default: 8192)
	MaxTokens int

	// Temperature controls randomness in LLM responses (default: 0.0)
	Temperature float64
}

// LLMConfigFromEnv creates an LLMConfig by loading values from environment variables.
// It follows this precedence:
//  1. Prefixed environment variables (e.g., DOCUMENTER_LLM_PROVIDER, AI_RULES_LLM_MODEL)
//  2. ANALYZER_* fallback variables for Provider, Model, and APIKey
//  3. Provided default values for Retries, Timeout, MaxTokens, Temperature
//
// The prefix parameter should be the command-specific prefix (e.g., "DOCUMENTER", "AI_RULES").
// Environment variable pattern: {PREFIX}_LLM_{FIELD} where FIELD is PROVIDER, MODEL, API_KEY, or BASE_URL.
//
// Example usage:
//
//	cfg := LLMConfigFromEnv("DOCUMENTER", LLMDefaults{
//	    Retries:     2,
//	    Timeout:     180,
//	    MaxTokens:   8192,
//	    Temperature: 0.0,
//	})
func LLMConfigFromEnv(prefix string, defaults LLMDefaults) config.LLMConfig {
	cfg := config.LLMConfig{
		Provider:    os.Getenv(prefix + "_LLM_PROVIDER"),
		Model:       os.Getenv(prefix + "_LLM_MODEL"),
		APIKey:      os.Getenv(prefix + "_LLM_API_KEY"),
		BaseURL:     os.Getenv(prefix + "_LLM_BASE_URL"),
		Retries:     defaults.Retries,
		Timeout:     defaults.Timeout,
		MaxTokens:   defaults.MaxTokens,
		Temperature: defaults.Temperature,
	}

	// Fall back to ANALYZER_* environment variables for Provider, Model, and APIKey
	if cfg.Provider == "" {
		cfg.Provider = os.Getenv("ANALYZER_LLM_PROVIDER")
	}
	if cfg.Model == "" {
		cfg.Model = os.Getenv("ANALYZER_LLM_MODEL")
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ANALYZER_LLM_API_KEY")
	}

	return cfg
}

// HandleCommandError encapsulates the duplicated error handling pattern in CLI commands.
// It handles errors by:
//  1. Checking if the error is an *errors.AIDocGenError for user-friendly messages
//  2. Displaying the error via progress UI (if showProgress is true) or stderr
//  3. Returning the original error for proper exit code handling
//
// Parameters:
//   - err: the error to handle (if nil, returns nil immediately)
//   - progress: optional SimpleProgress for displaying errors in progress UI
//   - showProgress: if true, uses progress UI; if false, writes to stderr
//
// Returns the original error unchanged (allows chaining: return HandleCommandError(...))
func HandleCommandError(err error, progress *tui.SimpleProgress, showProgress bool) error {
	if err == nil {
		return nil
	}

	// Check if it's an AIDocGenError for better user messaging
	if docErr, ok := err.(*errors.AIDocGenError); ok {
		if showProgress && progress != nil {
			progress.Error(docErr.GetUserMessage())
			progress.Failed(nil)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
		}
		return docErr
	}

	// For other errors, show via progress or let it propagate
	if showProgress && progress != nil {
		progress.Failed(err)
	}
	return err
}
