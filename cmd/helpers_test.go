package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	appErrors "github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/tui"
)

// TestInitLogger_CurrentDirectory tests logger initialization with current directory
func TestInitLogger_CurrentDirectory(t *testing.T) {
	// Create a temp directory to avoid polluting the actual repo
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	logger, err := InitLogger(".", false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Sync()

	// Verify log directory was created at .ai/logs
	if _, err := os.Stat(filepath.Join(tmpDir, ".ai", "logs")); os.IsNotExist(err) {
		t.Error("Expected .ai/logs directory to be created")
	}
}

// TestInitLogger_CustomRepoPath tests logger initialization with a custom repo path
func TestInitLogger_CustomRepoPath(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "my-repo")
	os.MkdirAll(repoPath, 0755)

	logger, err := InitLogger(repoPath, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Sync()

	// Verify log directory was created at repoPath/.ai/logs
	logDir := filepath.Join(repoPath, ".ai", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("Expected %s directory to be created", logDir)
	}
}

// TestInitLogger_DebugFlag tests that debug flag enables caller info
func TestInitLogger_DebugFlag(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// With debug = true
	logger, err := InitLogger(".", true, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Sync()

	// Logger should be created successfully with debug enabled
	// The actual caller info is internal to the logger; we just verify creation
	if logger == nil {
		t.Error("Expected logger to be created with debug flag")
	}
}

// TestInitLogger_VerboseFlag tests that verbose flag affects console output
func TestInitLogger_VerboseFlag(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// With verbose = true (showProgress = false, console enabled)
	logger, err := InitLogger(".", false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Sync()

	if logger == nil {
		t.Error("Expected logger to be created with verbose flag")
	}
}

// TestInitLogger_AllFlagCombinations tests various combinations of debug and verbose
func TestInitLogger_AllFlagCombinations(t *testing.T) {
	testCases := []struct {
		name    string
		debug   bool
		verbose bool
	}{
		{"debug=false,verbose=false", false, false},
		{"debug=false,verbose=true", false, true},
		{"debug=true,verbose=false", true, false},
		{"debug=true,verbose=true", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(originalDir)

			logger, err := InitLogger(".", tc.debug, tc.verbose)
			if err != nil {
				t.Fatalf("Expected no error for %s, got %v", tc.name, err)
			}
			defer logger.Sync()

			if logger == nil {
				t.Errorf("Expected logger to be created for %s", tc.name)
			}
		})
	}
}

// TestLLMConfigFromEnv_PrefixedVariables tests loading config from prefixed env vars
func TestLLMConfigFromEnv_PrefixedVariables(t *testing.T) {
	// Save and clear environment
	os.Clearenv()
	defer os.Clearenv()

	// Set prefixed env vars
	os.Setenv("DOCUMENTER_LLM_PROVIDER", "openai")
	os.Setenv("DOCUMENTER_LLM_MODEL", "gpt-4")
	os.Setenv("DOCUMENTER_LLM_API_KEY", "doc-api-key")
	os.Setenv("DOCUMENTER_LLM_BASE_URL", "https://custom.openai.com")

	defaults := LLMDefaults{
		Retries:     3,
		Timeout:     120,
		MaxTokens:   4096,
		Temperature: 0.5,
	}

	cfg := LLMConfigFromEnv("DOCUMENTER", defaults)

	if cfg.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", cfg.Provider)
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", cfg.Model)
	}
	if cfg.APIKey != "doc-api-key" {
		t.Errorf("Expected API key 'doc-api-key', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://custom.openai.com" {
		t.Errorf("Expected base URL 'https://custom.openai.com', got '%s'", cfg.BaseURL)
	}
	if cfg.Retries != 3 {
		t.Errorf("Expected retries 3, got %d", cfg.Retries)
	}
	if cfg.Timeout != 120 {
		t.Errorf("Expected timeout 120, got %d", cfg.Timeout)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("Expected max tokens 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", cfg.Temperature)
	}
}

// TestLLMConfigFromEnv_FallbackToAnalyzer tests fallback to ANALYZER_* vars
func TestLLMConfigFromEnv_FallbackToAnalyzer(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	// Only set ANALYZER_* fallback vars (no DOCUMENTER_* vars)
	os.Setenv("ANALYZER_LLM_PROVIDER", "anthropic")
	os.Setenv("ANALYZER_LLM_MODEL", "claude-3")
	os.Setenv("ANALYZER_LLM_API_KEY", "analyzer-key")

	defaults := LLMDefaults{
		Retries:     2,
		Timeout:     180,
		MaxTokens:   8192,
		Temperature: 0.0,
	}

	cfg := LLMConfigFromEnv("DOCUMENTER", defaults)

	// Should fall back to ANALYZER_* values
	if cfg.Provider != "anthropic" {
		t.Errorf("Expected fallback provider 'anthropic', got '%s'", cfg.Provider)
	}
	if cfg.Model != "claude-3" {
		t.Errorf("Expected fallback model 'claude-3', got '%s'", cfg.Model)
	}
	if cfg.APIKey != "analyzer-key" {
		t.Errorf("Expected fallback API key 'analyzer-key', got '%s'", cfg.APIKey)
	}
}

// TestLLMConfigFromEnv_PrefixOverridesAnalyzer tests that prefixed vars override ANALYZER_*
func TestLLMConfigFromEnv_PrefixOverridesAnalyzer(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	// Set both prefixed and ANALYZER_* vars
	os.Setenv("AI_RULES_LLM_PROVIDER", "gemini")
	os.Setenv("AI_RULES_LLM_MODEL", "gemini-pro")
	os.Setenv("AI_RULES_LLM_API_KEY", "ai-rules-key")
	os.Setenv("ANALYZER_LLM_PROVIDER", "openai")
	os.Setenv("ANALYZER_LLM_MODEL", "gpt-4")
	os.Setenv("ANALYZER_LLM_API_KEY", "analyzer-key")

	defaults := LLMDefaults{}

	cfg := LLMConfigFromEnv("AI_RULES", defaults)

	// Prefixed vars should win
	if cfg.Provider != "gemini" {
		t.Errorf("Expected provider 'gemini', got '%s'", cfg.Provider)
	}
	if cfg.Model != "gemini-pro" {
		t.Errorf("Expected model 'gemini-pro', got '%s'", cfg.Model)
	}
	if cfg.APIKey != "ai-rules-key" {
		t.Errorf("Expected API key 'ai-rules-key', got '%s'", cfg.APIKey)
	}
}

// TestLLMConfigFromEnv_NoEnvVars tests behavior with no env vars set
func TestLLMConfigFromEnv_NoEnvVars(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	defaults := LLMDefaults{
		Retries:     5,
		Timeout:     300,
		MaxTokens:   16384,
		Temperature: 0.7,
	}

	cfg := LLMConfigFromEnv("TEST", defaults)

	// Should have empty strings for env-loaded values
	if cfg.Provider != "" {
		t.Errorf("Expected empty provider, got '%s'", cfg.Provider)
	}
	if cfg.Model != "" {
		t.Errorf("Expected empty model, got '%s'", cfg.Model)
	}
	if cfg.APIKey != "" {
		t.Errorf("Expected empty API key, got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		t.Errorf("Expected empty base URL, got '%s'", cfg.BaseURL)
	}

	// Defaults should be applied
	if cfg.Retries != 5 {
		t.Errorf("Expected retries 5, got %d", cfg.Retries)
	}
	if cfg.Timeout != 300 {
		t.Errorf("Expected timeout 300, got %d", cfg.Timeout)
	}
	if cfg.MaxTokens != 16384 {
		t.Errorf("Expected max tokens 16384, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", cfg.Temperature)
	}
}

// TestLLMConfigFromEnv_PartialFallback tests partial fallback (some vars from prefix, some from ANALYZER_*)
func TestLLMConfigFromEnv_PartialFallback(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	// Set only provider from prefix, rest from ANALYZER_*
	os.Setenv("DOCUMENTER_LLM_PROVIDER", "gemini")
	os.Setenv("ANALYZER_LLM_MODEL", "gpt-4")
	os.Setenv("ANALYZER_LLM_API_KEY", "analyzer-key")

	defaults := LLMDefaults{}

	cfg := LLMConfigFromEnv("DOCUMENTER", defaults)

	if cfg.Provider != "gemini" {
		t.Errorf("Expected provider 'gemini' from prefix, got '%s'", cfg.Provider)
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4' from fallback, got '%s'", cfg.Model)
	}
	if cfg.APIKey != "analyzer-key" {
		t.Errorf("Expected API key 'analyzer-key' from fallback, got '%s'", cfg.APIKey)
	}
}

// TestLLMConfigFromEnv_DifferentPrefixes tests various prefix values
func TestLLMConfigFromEnv_DifferentPrefixes(t *testing.T) {
	testCases := []struct {
		prefix   string
		envKey   string
		expected string
	}{
		{"DOCUMENTER", "DOCUMENTER_LLM_PROVIDER", "provider1"},
		{"AI_RULES", "AI_RULES_LLM_PROVIDER", "provider2"},
		{"ANALYZER", "ANALYZER_LLM_PROVIDER", "provider3"},
		{"CUSTOM", "CUSTOM_LLM_PROVIDER", "provider4"},
	}

	for _, tc := range testCases {
		t.Run(tc.prefix, func(t *testing.T) {
			os.Clearenv()
			os.Setenv(tc.envKey, tc.expected)

			cfg := LLMConfigFromEnv(tc.prefix, LLMDefaults{})

			if cfg.Provider != tc.expected {
				t.Errorf("Expected provider '%s', got '%s'", tc.expected, cfg.Provider)
			}
		})
	}
}

// TestHandleCommandError_NilError tests that nil error returns nil
func TestHandleCommandError_NilError(t *testing.T) {
	result := HandleCommandError(nil, nil, false)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

// TestHandleCommandError_NilErrorWithProgress tests nil error with progress
func TestHandleCommandError_NilErrorWithProgress(t *testing.T) {
	progress := tui.NewSimpleProgress("Test")
	result := HandleCommandError(nil, progress, true)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

// TestHandleCommandError_AIDocGenError_NoProgress tests AIDocGenError without progress UI
func TestHandleCommandError_AIDocGenError_NoProgress(t *testing.T) {
	// Create an AIDocGenError
	docErr := appErrors.NewError("test error message", appErrors.ExitGeneralError)

	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	result := HandleCommandError(docErr, nil, false)

	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should return the same error
	if result != docErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}

	// Should print to stderr
	if len(output) == 0 {
		t.Error("Expected error message to be printed to stderr")
	}
}

// TestHandleCommandError_AIDocGenError_WithProgress tests AIDocGenError with progress UI
func TestHandleCommandError_AIDocGenError_WithProgress(t *testing.T) {
	docErr := appErrors.NewError("user-facing error", appErrors.ExitGeneralError)
	progress := tui.NewSimpleProgress("Test")

	// We can't easily capture progress output, but we can verify the function completes
	// without panicking and returns the correct error
	result := HandleCommandError(docErr, progress, true)

	if result != docErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}
}

// TestHandleCommandError_RegularError_NoProgress tests regular error without progress UI
func TestHandleCommandError_RegularError_NoProgress(t *testing.T) {
	regularErr := errors.New("regular error")

	// With showProgress=false and no progress, error should just be returned
	result := HandleCommandError(regularErr, nil, false)

	if result != regularErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}
}

// TestHandleCommandError_RegularError_WithProgress tests regular error with progress UI
func TestHandleCommandError_RegularError_WithProgress(t *testing.T) {
	regularErr := errors.New("regular error with progress")
	progress := tui.NewSimpleProgress("Test")

	result := HandleCommandError(regularErr, progress, true)

	if result != regularErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}
}

// TestHandleCommandError_ShowProgressFalse_WithNilProgress tests showProgress=false with nil progress
func TestHandleCommandError_ShowProgressFalse_WithNilProgress(t *testing.T) {
	regularErr := errors.New("some error")

	// Should not panic even with nil progress when showProgress is false
	result := HandleCommandError(regularErr, nil, false)

	if result != regularErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}
}

// TestHandleCommandError_ShowProgressTrue_WithNilProgress tests showProgress=true with nil progress
func TestHandleCommandError_ShowProgressTrue_WithNilProgress(t *testing.T) {
	regularErr := errors.New("some error")

	// Should not panic even with nil progress - the function should handle this
	result := HandleCommandError(regularErr, nil, true)

	if result != regularErr {
		t.Errorf("Expected same error to be returned, got %v", result)
	}
}

// TestCommandContext_Fields tests CommandContext struct fields
func TestCommandContext_Fields(t *testing.T) {
	ctx := CommandContext{
		Logger:       nil,
		ShowProgress: true,
		RepoPath:     "/test/path",
	}

	if !ctx.ShowProgress {
		t.Error("Expected ShowProgress to be true")
	}
	if ctx.RepoPath != "/test/path" {
		t.Errorf("Expected RepoPath '/test/path', got '%s'", ctx.RepoPath)
	}
}

// TestLLMDefaults_ZeroValues tests LLMDefaults with zero values
func TestLLMDefaults_ZeroValues(t *testing.T) {
	defaults := LLMDefaults{}

	if defaults.Retries != 0 {
		t.Errorf("Expected Retries 0, got %d", defaults.Retries)
	}
	if defaults.Timeout != 0 {
		t.Errorf("Expected Timeout 0, got %d", defaults.Timeout)
	}
	if defaults.MaxTokens != 0 {
		t.Errorf("Expected MaxTokens 0, got %d", defaults.MaxTokens)
	}
	if defaults.Temperature != 0.0 {
		t.Errorf("Expected Temperature 0.0, got %f", defaults.Temperature)
	}
}

// TestLLMDefaults_TypicalValues tests LLMDefaults with typical values
func TestLLMDefaults_TypicalValues(t *testing.T) {
	defaults := LLMDefaults{
		Retries:     2,
		Timeout:     180,
		MaxTokens:   8192,
		Temperature: 0.0,
	}

	if defaults.Retries != 2 {
		t.Errorf("Expected Retries 2, got %d", defaults.Retries)
	}
	if defaults.Timeout != 180 {
		t.Errorf("Expected Timeout 180, got %d", defaults.Timeout)
	}
	if defaults.MaxTokens != 8192 {
		t.Errorf("Expected MaxTokens 8192, got %d", defaults.MaxTokens)
	}
	if defaults.Temperature != 0.0 {
		t.Errorf("Expected Temperature 0.0, got %f", defaults.Temperature)
	}
}
