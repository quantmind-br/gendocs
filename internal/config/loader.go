package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
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

	for key, value := range cliOverrides {
		if value != nil {
			setNested(sectionConfig, key, value)
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

	cfg := &AnalyzerConfig{}
	decoderConfig := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           cfg,
		TagName:          "mapstructure",
		Squash:           true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config decoder: %w", err)
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode analyzer config: %w", err)
	}

	applyAnalyzerDefaults(cfg)
	applyAnalyzerEnvOverrides(cfg)

	if err := validateLLMConfig(&cfg.LLM, "ANALYZER"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyAnalyzerDefaults(cfg *AnalyzerConfig) {
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "openai"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o"
	}
	if cfg.LLM.Retries == 0 {
		cfg.LLM.Retries = 2
	}
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 180
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 8192
	}
}

func applyLLMEnvOverrides(llm *LLMConfig, prefix string, fallbackPrefix string, defaults LLMDefaults) {
	if llm.Provider == defaults.Provider {
		if fallbackPrefix != "" {
			if env := getEnvWithFallback(prefix+"_LLM_PROVIDER", fallbackPrefix+"_LLM_PROVIDER", ""); env != "" {
				llm.Provider = env
			}
		} else if env := os.Getenv(prefix + "_LLM_PROVIDER"); env != "" {
			llm.Provider = env
		}
	}
	if llm.Model == defaults.Model {
		if fallbackPrefix != "" {
			if env := getEnvWithFallback(prefix+"_LLM_MODEL", fallbackPrefix+"_LLM_MODEL", ""); env != "" {
				llm.Model = env
			}
		} else if env := os.Getenv(prefix + "_LLM_MODEL"); env != "" {
			llm.Model = env
		}
	}
	if llm.APIKey == "" {
		if fallbackPrefix != "" {
			llm.APIKey = getEnvWithFallback(prefix+"_LLM_API_KEY", fallbackPrefix+"_LLM_API_KEY", "")
		} else if env := os.Getenv(prefix + "_LLM_API_KEY"); env != "" {
			llm.APIKey = env
		}
	}
	if llm.BaseURL == "" {
		if env := os.Getenv(prefix + "_LLM_BASE_URL"); env != "" {
			llm.BaseURL = env
		}
	}
	if llm.Retries == defaults.Retries {
		llm.Retries = getEnvIntOrDefault(prefix+"_AGENT_RETRIES", defaults.Retries)
	}
	if llm.Timeout == defaults.Timeout {
		llm.Timeout = getEnvIntOrDefault(prefix+"_LLM_TIMEOUT", defaults.Timeout)
	}
	if llm.MaxTokens == defaults.MaxTokens {
		llm.MaxTokens = getEnvIntOrDefault(prefix+"_LLM_MAX_TOKENS", defaults.MaxTokens)
	}
	if llm.Temperature == defaults.Temperature {
		llm.Temperature = getEnvFloatOrDefault(prefix+"_LLM_TEMPERATURE", defaults.Temperature)
	}
}

type LLMDefaults struct {
	Provider    string
	Model       string
	Retries     int
	Timeout     int
	MaxTokens   int
	Temperature float64
}

var defaultLLMConfig = LLMDefaults{
	Provider:    "openai",
	Model:       "gpt-4o",
	Retries:     2,
	Timeout:     180,
	MaxTokens:   8192,
	Temperature: 0.0,
}

var aiRulesLLMDefaults = LLMDefaults{
	Provider:    "openai",
	Model:       "gpt-4o",
	Retries:     2,
	Timeout:     240, // AIRules uses longer timeout
	MaxTokens:   8192,
	Temperature: 0.0,
}

func applyAnalyzerEnvOverrides(cfg *AnalyzerConfig) {
	applyLLMEnvOverrides(&cfg.LLM, "ANALYZER", "", defaultLLMConfig)
}

func LoadDocumenterConfig(repoPath string, cliOverrides map[string]interface{}) (*DocumenterConfig, error) {
	configMap, err := MergeConfigs(repoPath, "documenter", &DocumenterConfig{}, cliOverrides)
	if err != nil {
		return nil, err
	}

	cfg := &DocumenterConfig{}
	decoderConfig := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           cfg,
		TagName:          "mapstructure",
		Squash:           true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config decoder: %w", err)
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode documenter config: %w", err)
	}

	applyDocumenterDefaults(cfg)
	applyDocumenterEnvOverrides(cfg)

	if err := validateLLMConfig(&cfg.LLM, "DOCUMENTER"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyDocumenterDefaults(cfg *DocumenterConfig) {
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "openai"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o"
	}
	if cfg.LLM.Retries == 0 {
		cfg.LLM.Retries = 2
	}
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 180
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 8192
	}
}

func applyDocumenterEnvOverrides(cfg *DocumenterConfig) {
	applyLLMEnvOverrides(&cfg.LLM, "DOCUMENTER", "ANALYZER", defaultLLMConfig)
}

func LoadAIRulesConfig(repoPath string, cliOverrides map[string]interface{}) (*AIRulesConfig, error) {
	configMap, err := MergeConfigs(repoPath, "ai_rules", &AIRulesConfig{}, cliOverrides)
	if err != nil {
		return nil, err
	}

	cfg := &AIRulesConfig{}
	decoderConfig := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           cfg,
		TagName:          "mapstructure",
		Squash:           true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config decoder: %w", err)
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode ai_rules config: %w", err)
	}

	applyAIRulesDefaults(cfg)
	applyAIRulesEnvOverrides(cfg)

	if err := validateLLMConfig(&cfg.LLM, "AI_RULES"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyAIRulesDefaults(cfg *AIRulesConfig) {
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "openai"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o"
	}
	if cfg.LLM.Retries == 0 {
		cfg.LLM.Retries = 2
	}
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 240
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 8192
	}
}

func applyAIRulesEnvOverrides(cfg *AIRulesConfig) {
	applyLLMEnvOverrides(&cfg.LLM, "AI_RULES", "ANALYZER", aiRulesLLMDefaults)
}

func setNested(m map[string]interface{}, dottedKey string, value interface{}) {
	parts := strings.Split(dottedKey, ".")
	if len(parts) == 1 {
		m[dottedKey] = value
		return
	}

	current := m
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
	current[parts[len(parts)-1]] = value
}

func getEnvWithFallback(primaryKey, fallbackKey, defaultValue string) string {
	if val := os.Getenv(primaryKey); val != "" {
		return val
	}
	if val := os.Getenv(fallbackKey); val != "" {
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

// LoadGlobalConfig loads and returns the full GlobalConfig from ~/.gendocs.yaml
func (l *Loader) LoadGlobalConfig() (*GlobalConfig, error) {
	if err := l.loadGlobalConfig(); err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{}
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}

	return cfg, nil
}

// LoadProjectConfig loads project-specific config from .ai/config.yaml
func (l *Loader) LoadProjectConfig(repoPath string) (*GlobalConfig, error) {
	if err := l.loadProjectConfig(repoPath); err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{}
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project config: %w", err)
	}

	return cfg, nil
}

// LoadMergedConfig loads config with proper precedence (project > global > env > defaults)
func (l *Loader) LoadMergedConfig(repoPath string) (*GlobalConfig, error) {
	if err := l.loadGlobalConfig(); err != nil {
		return nil, err
	}

	if err := l.loadProjectConfig(repoPath); err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{}
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged config: %w", err)
	}

	return cfg, nil
}

func LoadCheckConfig(repoPath string, cliOverrides map[string]interface{}) (*CheckConfig, error) {
	configMap, err := MergeConfigs(repoPath, "check", &CheckConfig{}, cliOverrides)
	if err != nil {
		return nil, err
	}

	cfg := &CheckConfig{}
	decoderConfig := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           cfg,
		TagName:          "mapstructure",
		Squash:           true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config decoder: %w", err)
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode check config: %w", err)
	}

	applyCheckDefaults(cfg)

	return cfg, nil
}

func applyCheckDefaults(cfg *CheckConfig) {
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "text"
	}
}
