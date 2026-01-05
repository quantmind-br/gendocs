package config

import (
	"time"
)

// BaseConfig holds common configuration for all handlers
type BaseConfig struct {
	RepoPath string `mapstructure:"repo_path" yaml:"repo_path"`
	Debug    bool   `mapstructure:"debug" yaml:"debug"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider    string         `mapstructure:"provider" yaml:"provider"` // openai, anthropic, gemini, ollama, lmstudio
	Model       string         `mapstructure:"model" yaml:"model"`
	APIKey      string         `mapstructure:"api_key" yaml:"api_key"`
	BaseURL     string         `mapstructure:"base_url" yaml:"base_url"` // Optional, for OpenAI-compatible APIs
	Retries     int            `mapstructure:"retries" yaml:"retries"`
	Timeout     int            `mapstructure:"timeout" yaml:"timeout"` // Timeout in seconds
	MaxTokens   int            `mapstructure:"max_tokens" yaml:"max_tokens"`
	Temperature float64        `mapstructure:"temperature" yaml:"temperature"`
	Cache       LLMCacheConfig `mapstructure:"cache" yaml:"cache"` // Cache configuration
}

// LLMCacheConfig holds LLM response cache configuration
type LLMCacheConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`       // Enable/disable caching
	MaxSize   int    `mapstructure:"max_size" yaml:"max_size"`     // Maximum number of entries in memory cache
	TTL       int    `mapstructure:"ttl" yaml:"ttl"`               // Time-to-live for cache entries in days
	CachePath string `mapstructure:"cache_path" yaml:"cache_path"` // Path to disk cache file
}

// GeminiConfig holds Gemini-specific configuration
type GeminiConfig struct {
	UseVertexAI bool   `mapstructure:"use_vertex_ai" yaml:"use_vertex_ai"`
	ProjectID   string `mapstructure:"project_id" yaml:"project_id"`
	Location    string `mapstructure:"location" yaml:"location"`
}

// RetryConfig holds HTTP retry configuration
type RetryConfig struct {
	MaxAttempts       int `mapstructure:"max_attempts" yaml:"max_attempts"`                 // Default: 5
	Multiplier        int `mapstructure:"multiplier" yaml:"multiplier"`                     // Default: 1
	MaxWaitPerAttempt int `mapstructure:"max_wait_per_attempt" yaml:"max_wait_per_attempt"` // Default: 60 seconds
	MaxTotalWait      int `mapstructure:"max_total_wait" yaml:"max_total_wait"`             // Default: 300 seconds
}

// AnalyzerConfig holds configuration for the analyze command
type AnalyzerConfig struct {
	BaseConfig       `yaml:",inline"`
	LLM              LLMConfig   `mapstructure:"llm" yaml:"llm"`
	ExcludeStructure bool        `mapstructure:"exclude_code_structure" yaml:"exclude_code_structure"`
	ExcludeDataFlow  bool        `mapstructure:"exclude_data_flow" yaml:"exclude_data_flow"`
	ExcludeDeps      bool        `mapstructure:"exclude_dependencies" yaml:"exclude_dependencies"`
	ExcludeReqFlow   bool        `mapstructure:"exclude_request_flow" yaml:"exclude_request_flow"`
	ExcludeAPI       bool        `mapstructure:"exclude_api_analysis" yaml:"exclude_api_analysis"`
	MaxWorkers       int         `mapstructure:"max_workers" yaml:"max_workers"`
	MaxHashWorkers   int         `mapstructure:"max_hash_workers" yaml:"max_hash_workers"`
	RetryConfig      RetryConfig `mapstructure:"retry" yaml:"retry"`
	Force            bool        `mapstructure:"force" yaml:"force"`             // Force full re-analysis, ignore cache
	Incremental      bool        `mapstructure:"incremental" yaml:"incremental"` // Enable incremental analysis (default: true)
}

// DocumenterConfig holds configuration for readme generation
type DocumenterConfig struct {
	BaseConfig  `yaml:",inline"`
	LLM         LLMConfig   `mapstructure:"llm" yaml:"llm"`
	RetryConfig RetryConfig `mapstructure:"retry" yaml:"retry"`
}

// AIRulesConfig holds configuration for AI rules generation
type AIRulesConfig struct {
	BaseConfig        `yaml:",inline"`
	LLM               LLMConfig   `mapstructure:"llm" yaml:"llm"`
	RetryConfig       RetryConfig `mapstructure:"retry" yaml:"retry"`
	MaxTokensMarkdown int         `mapstructure:"max_tokens_markdown" yaml:"max_tokens_markdown"`
	MaxTokensCursor   int         `mapstructure:"max_tokens_cursor" yaml:"max_tokens_cursor"`
}

// CronjobConfig holds configuration for cronjob command
type CronjobConfig struct {
	MaxDaysSinceLastCommit int    `mapstructure:"max_days_since_last_commit" yaml:"max_days_since_last_commit"`
	WorkingPath            string `mapstructure:"working_path" yaml:"working_path"`
	GroupProjectID         int    `mapstructure:"group_project_id" yaml:"group_project_id"`
}

// GitLabConfig holds GitLab integration configuration
type GitLabConfig struct {
	APIURL       string `mapstructure:"api_url" yaml:"api_url"`
	UserName     string `mapstructure:"user_name" yaml:"user_name"`
	UserUsername string `mapstructure:"user_username" yaml:"user_username"`
	UserEmail    string `mapstructure:"user_email" yaml:"user_email"`
	OAuthToken   string `mapstructure:"oauth_token" yaml:"oauth_token"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	LogDir       string `mapstructure:"log_dir" yaml:"log_dir"`
	FileLevel    string `mapstructure:"file_level" yaml:"file_level"`       // debug, info, warn, error
	ConsoleLevel string `mapstructure:"console_level" yaml:"console_level"` // debug, info, warn, error
}

// CurrentConfigVersion is the current schema version for config files
const CurrentConfigVersion = 1

// GlobalConfig holds top-level configuration from .ai/config.yaml
type GlobalConfig struct {
	Version    int              `mapstructure:"version" yaml:"version"`
	Analyzer   AnalyzerConfig   `mapstructure:"analyzer" yaml:"analyzer"`
	Documenter DocumenterConfig `mapstructure:"documenter" yaml:"documenter"`
	AIRules    AIRulesConfig    `mapstructure:"ai_rules" yaml:"ai_rules"`
	Cronjob    CronjobConfig    `mapstructure:"cronjob" yaml:"cronjob"`
	GitLab     GitLabConfig     `mapstructure:"gitlab" yaml:"gitlab"`
	Gemini     GeminiConfig     `mapstructure:"gemini" yaml:"gemini"`
	Logging    LoggingConfig    `mapstructure:"logging" yaml:"logging"`
}

// GetTimeout returns the timeout as a time.Duration
func (c *LLMConfig) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return 180 * time.Second // Default timeout
	}
	return time.Duration(c.Timeout) * time.Second
}

// GetMaxTokens returns the max tokens with a default
func (c *LLMConfig) GetMaxTokens() int {
	if c.MaxTokens == 0 {
		return 8192 // Default max tokens
	}
	return c.MaxTokens
}

// GetTemperature returns the temperature with a default
func (c *LLMConfig) GetTemperature() float64 {
	if c.Temperature == 0 {
		return 0.0 // Default temperature for deterministic output
	}
	return c.Temperature
}

// GetRetries returns the retry count with a default
func (c *LLMConfig) GetRetries() int {
	if c.Retries == 0 {
		return 2 // Default retries
	}
	return c.Retries
}

// IsEnabled returns whether caching is enabled
func (c *LLMCacheConfig) IsEnabled() bool {
	return c.Enabled
}

// GetMaxSize returns the maximum cache size with a default
func (c *LLMCacheConfig) GetMaxSize() int {
	if c.MaxSize == 0 {
		return 1000 // Default max entries
	}
	return c.MaxSize
}

// GetTTL returns the TTL as a time.Duration with a default
func (c *LLMCacheConfig) GetTTL() time.Duration {
	if c.TTL == 0 {
		return 7 * 24 * time.Hour // Default 7 days
	}
	return time.Duration(c.TTL) * 24 * time.Hour
}

// GetCachePath returns the cache file path with a default
func (c *LLMCacheConfig) GetCachePath() string {
	if c.CachePath == "" {
		return ".ai/llm_cache.json" // Default cache path
	}
	return c.CachePath
}

// GetMaxHashWorkers returns the max hash workers with a default (0 = use CPU count with max of 8)
func (c *AnalyzerConfig) GetMaxHashWorkers() int {
	return c.MaxHashWorkers
}

// CheckConfig holds configuration for the check command (drift detection)
type CheckConfig struct {
	BaseConfig     `yaml:",inline"`
	MaxHashWorkers int    `mapstructure:"max_hash_workers" yaml:"max_hash_workers"`
	OutputFormat   string `mapstructure:"output_format" yaml:"output_format"` // text, json
	Verbose        bool   `mapstructure:"verbose" yaml:"verbose"`
}

// GetMaxHashWorkers returns the max hash workers with a default (0 = use CPU count with max of 8)
func (c *CheckConfig) GetMaxHashWorkers() int {
	return c.MaxHashWorkers
}
