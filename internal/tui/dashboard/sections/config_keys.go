// Package sections provides TUI dashboard section components.
// This file contains constants for configuration keys used in GetValues/SetValues
// to prevent typos and enable compile-time safety.
package sections

// LLM configuration keys (used with optional prefix for documenter_/ai_rules_)
const (
	KeyProvider    = "provider"
	KeyModel       = "model"
	KeyAPIKey      = "api_key"
	KeyBaseURL     = "base_url"
	KeyTemperature = "temperature"
	KeyMaxTokens   = "max_tokens"
	KeyTimeout     = "timeout"
	KeyRetries     = "retries"
)

// Analysis configuration keys
const (
	KeyExcludeCodeStructure = "exclude_code_structure"
	KeyExcludeDataFlow      = "exclude_data_flow"
	KeyExcludeDependencies  = "exclude_dependencies"
	KeyExcludeRequestFlow   = "exclude_request_flow"
	KeyExcludeAPIAnalysis   = "exclude_api_analysis"
	KeyMaxWorkers           = "max_workers"
	KeyMaxHashWorkers       = "max_hash_workers"
	KeyForce                = "force"
	KeyIncremental          = "incremental"
)

// Cache configuration keys
const (
	KeyCacheEnabled = "cache_enabled"
	KeyCachePath    = "cache_path"
	KeyCacheMaxSize = "cache_max_size"
	KeyCacheTTL     = "cache_ttl"
)

// Retry configuration keys
const (
	KeyMaxAttempts       = "max_attempts"
	KeyMultiplier        = "multiplier"
	KeyMaxWaitPerAttempt = "max_wait_per_attempt"
	KeyMaxTotalWait      = "max_total_wait"
)

// Cronjob configuration keys
const (
	KeyMaxDaysSinceLastCommit = "max_days_since_last_commit"
	KeyGroupProjectID         = "group_project_id"
	KeyWorkingPath            = "working_path"
)

// Gemini configuration keys
const (
	KeyUseVertexAI = "use_vertex_ai"
	KeyProjectID   = "project_id"
	KeyLocation    = "location"
)

// GitLab configuration keys
const (
	KeyGitLabAPIURL       = "gitlab_api_url"
	KeyGitLabUserName     = "gitlab_user_name"
	KeyGitLabUserUsername = "gitlab_user_username"
	KeyGitLabUserEmail    = "gitlab_user_email"
	KeyGitLabOAuthToken   = "gitlab_oauth_token"
)

// Logging configuration keys
const (
	KeyLogDir       = "log_dir"
	KeyFileLevel    = "file_level"
	KeyConsoleLevel = "console_level"
)
