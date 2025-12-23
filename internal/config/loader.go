package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/user/gendocs/internal/errors"
)

// Loader handles loading configuration from multiple sources
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	// Load .env file if exists
	_ = godotenv.Load()

	v := viper.New()
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvPrefix("GENDOCS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return &Loader{v: v}
}

// LoadForAgent loads configuration for a specific agent section
// Precedence: CLI > .ai/config.yaml > ~/.gendocs.yaml > Environment > Defaults
func (l *Loader) LoadForAgent(repoPath, section string, cliOverrides map[string]interface{}) (*viper.Viper, error) {
	// 1. Load defaults (set via struct defaults)

	// 2. Load from ~/.gendocs.yaml (global user config)
	if err := l.loadGlobalConfig(); err != nil {
		return nil, err
	}

	// 3. Load from .ai/config.yaml (project-specific config)
	if err := l.loadProjectConfig(repoPath); err != nil {
		return nil, err
	}

	// 4. Apply CLI overrides
	if err := l.applyCLIOverrides(cliOverrides); err != nil {
		return nil, err
	}

	return l.v, nil
}

// loadGlobalConfig loads configuration from ~/.gendocs.yaml
func (l *Loader) loadGlobalConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Not a fatal error
	}

	globalConfig := filepath.Join(homeDir, ".gendocs.yaml")
	if _, err := os.Stat(globalConfig); err != nil {
		return nil // File doesn't exist, skip
	}

	l.v.SetConfigFile(globalConfig)
	if err := l.v.ReadInConfig(); err != nil {
		return errors.NewConfigFileError(globalConfig, err)
	}

	return nil
}

// loadProjectConfig loads configuration from .ai/config.yaml
func (l *Loader) loadProjectConfig(repoPath string) error {
	if repoPath == "" {
		repoPath = "."
	}

	configPath := filepath.Join(repoPath, ".ai", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return nil // File doesn't exist, skip
	}

	l.v.SetConfigFile(configPath)
	if err := l.v.MergeInConfig(); err != nil {
		return errors.NewConfigFileError(configPath, err)
	}

	return nil
}

// applyCLIOverrides applies CLI flag overrides
func (l *Loader) applyCLIOverrides(overrides map[string]interface{}) error {
	for key, value := range overrides {
		// Only set if value is not nil/zero
		if value != nil {
			l.v.Set(key, value)
		}
	}
	return nil
}

// GetEnvVar gets an environment variable, returning an error if not set
func GetEnvVar(name, description string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", errors.NewMissingEnvVarError(name, description)
	}
	return value, nil
}

// GetEnvVarOrDefault gets an environment variable with a default value
func GetEnvVarOrDefault(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// MergeConfigs merges multiple configuration sources with precedence
// Precedence order (highest to lowest): cli, project, global, env, defaults
func MergeConfigs(repoPath string, section string, defaults interface{}, cliOverrides map[string]interface{}) (map[string]interface{}, error) {
	loader := NewLoader()

	// Load all config sources
	v, err := loader.LoadForAgent(repoPath, section, cliOverrides)
	if err != nil {
		return nil, err
	}

	// Get the section-specific config
	var sectionConfig map[string]interface{}
	if section != "" {
		sectionConfig = v.GetStringMap(section)
	} else {
		// Get all settings if no section specified
		sectionConfig = v.AllSettings()
	}

	// Apply CLI overrides (highest precedence)
	for key, value := range cliOverrides {
		if value != nil {
			// Convert key from snake_case to dot notation if needed
			sectionConfig[key] = value
		}
	}

	return sectionConfig, nil
}

// LoadAnalyzerConfig loads and validates analyzer configuration
func LoadAnalyzerConfig(repoPath string, cliOverrides map[string]interface{}) (*AnalyzerConfig, error) {
	configMap, err := MergeConfigs(repoPath, "analyzer", &AnalyzerConfig{}, cliOverrides)
	if err != nil {
		return nil, err
	}

	// Create config from map
	cfg := &AnalyzerConfig{
		BaseConfig: BaseConfig{
			RepoPath: getString(configMap, "repo_path", "."),
			Debug:    getBool(configMap, "debug", false),
		},
		MaxWorkers: getInt(configMap, "max_workers", 0),
	}

	// Load LLM config from environment or config
	cfg.LLM = LLMConfig{
		Provider:    getString(configMap, "llm.provider", getEnvOrDefault("ANALYZER_LLM_PROVIDER", "openai")),
		Model:       getString(configMap, "llm.model", getEnvOrDefault("ANALYZER_LLM_MODEL", "gpt-4o")),
		APIKey:      getString(configMap, "llm.api_key", getEnvOrDefault("ANALYZER_LLM_API_KEY", "")),
		BaseURL:     getString(configMap, "llm.base_url", getEnvOrDefault("ANALYZER_LLM_BASE_URL", "")),
		Retries:     getInt(configMap, "llm.retries", getEnvIntOrDefault("ANALYZER_AGENT_RETRIES", 2)),
		Timeout:     getInt(configMap, "llm.timeout", getEnvIntOrDefault("ANALYZER_LLM_TIMEOUT", 180)),
		MaxTokens:   getInt(configMap, "llm.max_tokens", getEnvIntOrDefault("ANALYZER_LLM_MAX_TOKENS", 8192)),
		Temperature: getFloat64(configMap, "llm.temperature", getEnvFloatOrDefault("ANALYZER_LLM_TEMPERATURE", 0.0)),
	}

	cfg.ExcludeStructure = getBool(configMap, "exclude_code_structure", false)
	cfg.ExcludeDataFlow = getBool(configMap, "exclude_data_flow", false)
	cfg.ExcludeDeps = getBool(configMap, "exclude_dependencies", false)
	cfg.ExcludeReqFlow = getBool(configMap, "exclude_request_flow", false)
	cfg.ExcludeAPI = getBool(configMap, "exclude_api_analysis", false)

	// Validate required fields
	if err := validateLLMConfig(&cfg.LLM, "ANALYZER"); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Helper functions for type-safe config access

func getString(m map[string]interface{}, key, defaultValue string) string {
	parts := strings.Split(key, ".")
	var val interface{} = m

	for _, part := range parts {
		if subMap, ok := val.(map[string]interface{}); ok {
			val = subMap[part]
		} else {
			return defaultValue
		}
	}

	if str, ok := val.(string); ok {
		return str
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	parts := strings.Split(key, ".")
	var val interface{} = m

	for _, part := range parts {
		if subMap, ok := val.(map[string]interface{}); ok {
			val = subMap[part]
		} else {
			return defaultValue
		}
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		// Try to parse string as int
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	parts := strings.Split(key, ".")
	var val interface{} = m

	for _, part := range parts {
		if subMap, ok := val.(map[string]interface{}); ok {
			val = subMap[part]
		} else {
			return defaultValue
		}
	}

	if b, ok := val.(bool); ok {
		return b
	}
	return defaultValue
}

func getFloat64(m map[string]interface{}, key string, defaultValue float64) float64 {
	parts := strings.Split(key, ".")
	var val interface{} = m

	for _, part := range parts {
		if subMap, ok := val.(map[string]interface{}); ok {
			val = subMap[part]
		} else {
			return defaultValue
		}
	}

	if f, ok := val.(float64); ok {
		return f
	}
	return defaultValue
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f
		}
	}
	return defaultValue
}

// validateLLMConfig validates LLM configuration
func validateLLMConfig(cfg *LLMConfig, prefix string) error {
	if cfg.APIKey == "" {
		return errors.NewMissingEnvVarError(prefix+"_LLM_API_KEY", "API key for LLM provider")
	}

	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"gemini":    true,
	}

	if !validProviders[cfg.Provider] {
		return errors.NewInvalidEnvVarError(prefix+"_LLM_PROVIDER", cfg.Provider, "Must be one of: openai, anthropic, gemini")
	}

	return nil
}
