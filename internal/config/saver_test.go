package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaver_SaveGlobalConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", originalHome) })
	os.Setenv("HOME", tmpDir)

	saver := NewSaver()
	cfg := &GlobalConfig{
		Version: CurrentConfigVersion,
		Analyzer: AnalyzerConfig{
			BaseConfig: BaseConfig{RepoPath: "/test/path"},
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4o",
				APIKey:   "test-key",
			},
		},
	}

	err := saver.SaveGlobalConfig(cfg)
	if err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".gendocs.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", expectedPath)
	}

	info, _ := os.Stat(expectedPath)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", perm)
	}
}

func TestSaver_SaveProjectConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewSaver()
	cfg := &GlobalConfig{
		Version: CurrentConfigVersion,
		Analyzer: AnalyzerConfig{
			LLM: LLMConfig{Provider: "anthropic", Model: "claude-3"},
		},
	}

	err := saver.SaveProjectConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".ai", "config.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", expectedPath)
	}
}

func TestSaver_SaveProjectConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nested", "deep")
	saver := NewSaver()

	cfg := &GlobalConfig{Version: CurrentConfigVersion}
	err := saver.SaveProjectConfig(nonExistentDir, cfg)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed to create nested directory: %v", err)
	}

	expectedDir := filepath.Join(nonExistentDir, ".ai")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Fatalf("Directory was not created at %s", expectedDir)
	}
}

func TestSaver_RoundTrip_PreservesValues(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewSaver()

	original := &GlobalConfig{
		Version: CurrentConfigVersion,
		Analyzer: AnalyzerConfig{
			BaseConfig:       BaseConfig{RepoPath: "/analyzer/path", Debug: true},
			ExcludeStructure: true,
			ExcludeDataFlow:  false,
			ExcludeDeps:      true,
			MaxWorkers:       4,
			MaxHashWorkers:   2,
			Force:            true,
			Incremental:      false,
			LLM: LLMConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet",
				APIKey:      "sk-test-key-123",
				BaseURL:     "https://api.anthropic.com",
				Retries:     3,
				Timeout:     180,
				MaxTokens:   8192,
				Temperature: 0.1,
				Cache: LLMCacheConfig{
					Enabled:   true,
					MaxSize:   500,
					TTL:       14,
					CachePath: ".ai/cache.json",
				},
			},
			RetryConfig: RetryConfig{
				MaxAttempts:       5,
				Multiplier:        2,
				MaxWaitPerAttempt: 60,
				MaxTotalWait:      300,
			},
		},
		Documenter: DocumenterConfig{
			BaseConfig: BaseConfig{RepoPath: "/doc/path"},
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4o",
			},
		},
		AIRules: AIRulesConfig{
			MaxTokensMarkdown: 32000,
			MaxTokensCursor:   16000,
		},
		Cronjob: CronjobConfig{
			MaxDaysSinceLastCommit: 30,
			WorkingPath:            "/tmp/test",
			GroupProjectID:         12345,
		},
		GitLab: GitLabConfig{
			APIURL:       "https://gitlab.example.com",
			UserName:     "Test User",
			UserUsername: "testuser",
			UserEmail:    "test@example.com",
			OAuthToken:   "gitlab-token",
		},
		Gemini: GeminiConfig{
			UseVertexAI: true,
			ProjectID:   "test-project",
			Location:    "us-central1",
		},
		Logging: LoggingConfig{
			LogDir:       ".ai/logs",
			FileLevel:    "debug",
			ConsoleLevel: "info",
		},
	}

	err := saver.SaveProjectConfig(tmpDir, original)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".ai", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded GlobalConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", loaded.Version, original.Version)
	}
	if loaded.Analyzer.LLM.Provider != original.Analyzer.LLM.Provider {
		t.Errorf("LLM Provider mismatch: got %s, want %s", loaded.Analyzer.LLM.Provider, original.Analyzer.LLM.Provider)
	}
	if loaded.Analyzer.LLM.APIKey != original.Analyzer.LLM.APIKey {
		t.Errorf("LLM APIKey mismatch: got %s, want %s", loaded.Analyzer.LLM.APIKey, original.Analyzer.LLM.APIKey)
	}
	if loaded.Analyzer.LLM.Cache.Enabled != original.Analyzer.LLM.Cache.Enabled {
		t.Errorf("LLM Cache Enabled mismatch: got %v, want %v", loaded.Analyzer.LLM.Cache.Enabled, original.Analyzer.LLM.Cache.Enabled)
	}
	if loaded.Analyzer.MaxWorkers != original.Analyzer.MaxWorkers {
		t.Errorf("MaxWorkers mismatch: got %d, want %d", loaded.Analyzer.MaxWorkers, original.Analyzer.MaxWorkers)
	}
	if loaded.Cronjob.MaxDaysSinceLastCommit != original.Cronjob.MaxDaysSinceLastCommit {
		t.Errorf("Cronjob MaxDays mismatch: got %d, want %d", loaded.Cronjob.MaxDaysSinceLastCommit, original.Cronjob.MaxDaysSinceLastCommit)
	}
	if loaded.GitLab.APIURL != original.GitLab.APIURL {
		t.Errorf("GitLab APIURL mismatch: got %s, want %s", loaded.GitLab.APIURL, original.GitLab.APIURL)
	}
	if loaded.Gemini.UseVertexAI != original.Gemini.UseVertexAI {
		t.Errorf("Gemini UseVertexAI mismatch: got %v, want %v", loaded.Gemini.UseVertexAI, original.Gemini.UseVertexAI)
	}
	if loaded.Logging.FileLevel != original.Logging.FileLevel {
		t.Errorf("Logging FileLevel mismatch: got %s, want %s", loaded.Logging.FileLevel, original.Logging.FileLevel)
	}
}

func TestSaver_VersionHandling_SetsDefaultVersion(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewSaver()

	cfg := &GlobalConfig{
		Version: 0,
		Analyzer: AnalyzerConfig{
			LLM: LLMConfig{Provider: "openai"},
		},
	}

	err := saver.SaveProjectConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".ai", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded GlobalConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if loaded.Version != CurrentConfigVersion {
		t.Errorf("Expected version %d, got %d", CurrentConfigVersion, loaded.Version)
	}
}

func TestSaver_SaveProjectConfig_DefaultRepoPath(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(originalWd) })
	_ = os.Chdir(tmpDir)

	saver := NewSaver()
	cfg := &GlobalConfig{Version: CurrentConfigVersion}

	err := saver.SaveProjectConfig("", cfg)
	if err != nil {
		t.Fatalf("SaveProjectConfig with empty path failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".ai", "config.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", expectedPath)
	}
}

func TestGlobalConfigPath_ReturnsCorrectPath(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", originalHome) })
	os.Setenv("HOME", tmpDir)

	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, ".gendocs.yaml")
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

func TestProjectConfigPath_ReturnsCorrectPath(t *testing.T) {
	path := ProjectConfigPath("/test/repo")
	expected := filepath.Join("/test/repo", ".ai", "config.yaml")
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}

	pathEmpty := ProjectConfigPath("")
	expectedEmpty := filepath.Join(".", ".ai", "config.yaml")
	if pathEmpty != expectedEmpty {
		t.Errorf("Expected path %s, got %s", expectedEmpty, pathEmpty)
	}
}

func TestConfigExists_ReturnsTrueForExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")
	_ = os.WriteFile(testFile, []byte("test"), 0644)

	if !ConfigExists(testFile) {
		t.Error("Expected ConfigExists to return true for existing file")
	}
}

func TestConfigExists_ReturnsFalseForNonExistingFile(t *testing.T) {
	if ConfigExists("/nonexistent/path/file.yaml") {
		t.Error("Expected ConfigExists to return false for non-existing file")
	}
}

func TestGlobalConfigExists_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", originalHome) })
	os.Setenv("HOME", tmpDir)

	if GlobalConfigExists() {
		t.Error("Expected GlobalConfigExists to return false initially")
	}

	saver := NewSaver()
	cfg := &GlobalConfig{Version: CurrentConfigVersion}
	_ = saver.SaveGlobalConfig(cfg)

	if !GlobalConfigExists() {
		t.Error("Expected GlobalConfigExists to return true after saving")
	}
}

func TestProjectConfigExists_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	if ProjectConfigExists(tmpDir) {
		t.Error("Expected ProjectConfigExists to return false initially")
	}

	saver := NewSaver()
	cfg := &GlobalConfig{Version: CurrentConfigVersion}
	_ = saver.SaveProjectConfig(tmpDir, cfg)

	if !ProjectConfigExists(tmpDir) {
		t.Error("Expected ProjectConfigExists to return true after saving")
	}
}
