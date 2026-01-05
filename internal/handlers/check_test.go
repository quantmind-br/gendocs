package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/gendocs/internal/cache"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/logging"
	testHelpers "github.com/user/gendocs/internal/testing"
)

func TestCheckHandler_Handle_FirstRun(t *testing.T) {
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}",
		"go.mod":  "module example\n\ngo 1.21",
	})

	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: repoPath,
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.IsFirstRun {
		t.Error("expected IsFirstRun to be true")
	}

	if !report.HasDrift {
		t.Error("expected HasDrift to be true for first run")
	}

	if report.Severity != DriftSeverityMajor {
		t.Errorf("expected severity %s, got %s", DriftSeverityMajor, report.Severity)
	}
}

func TestCheckHandler_Handle_NoDrift(t *testing.T) {
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}",
		"go.mod":  "module example\n\ngo 1.21",
	})

	analysisCache := cache.NewCache()

	files, err := cache.ScanFiles(repoPath, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("failed to scan files: %v", err)
	}

	agentResults := map[string]bool{
		"structure_analyzer":    true,
		"dependency_analyzer":   true,
		"data_flow_analyzer":    true,
		"request_flow_analyzer": true,
		"api_analyzer":          true,
	}
	analysisCache.UpdateAfterAnalysis(repoPath, files, agentResults)

	if err := analysisCache.Save(repoPath); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: repoPath,
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.IsFirstRun {
		t.Error("expected IsFirstRun to be false")
	}

	if report.HasDrift {
		t.Errorf("expected HasDrift to be false, got new=%d, modified=%d, deleted=%d",
			len(report.NewFiles), len(report.ModifiedFiles), len(report.DeletedFiles))
	}

	if report.Severity != DriftSeverityNone {
		t.Errorf("expected severity %s, got %s", DriftSeverityNone, report.Severity)
	}
}

func TestCheckHandler_Handle_WithDrift(t *testing.T) {
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}",
		"go.mod":  "module example\n\ngo 1.21",
	})

	analysisCache := cache.NewCache()
	files, err := cache.ScanFiles(repoPath, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("failed to scan files: %v", err)
	}
	agentResults := map[string]bool{
		"structure_analyzer":    true,
		"dependency_analyzer":   true,
		"data_flow_analyzer":    true,
		"request_flow_analyzer": true,
		"api_analyzer":          true,
	}
	analysisCache.UpdateAfterAnalysis(repoPath, files, agentResults)
	if err := analysisCache.Save(repoPath); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	newFilePath := filepath.Join(repoPath, "utils.go")
	if err := os.WriteFile(newFilePath, []byte("package main\nfunc helper() {}"), 0644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: repoPath,
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.HasDrift {
		t.Error("expected HasDrift to be true")
	}

	if len(report.NewFiles) == 0 {
		t.Error("expected at least one new file")
	}

	foundUtilsGo := false
	for _, f := range report.NewFiles {
		if f == "utils.go" {
			foundUtilsGo = true
			break
		}
	}
	if !foundUtilsGo {
		t.Errorf("expected utils.go in new files, got %v", report.NewFiles)
	}
}

func TestCheckHandler_Handle_ModifiedFile(t *testing.T) {
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}",
		"go.mod":  "module example\n\ngo 1.21",
	})

	analysisCache := cache.NewCache()
	files, err := cache.ScanFiles(repoPath, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("failed to scan files: %v", err)
	}
	agentResults := map[string]bool{
		"structure_analyzer": true,
	}
	analysisCache.UpdateAfterAnalysis(repoPath, files, agentResults)
	if err := analysisCache.Save(repoPath); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	mainPath := filepath.Join(repoPath, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() { println(\"modified\") }"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: repoPath,
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.HasDrift {
		t.Error("expected HasDrift to be true")
	}

	if len(report.ModifiedFiles) == 0 {
		t.Error("expected at least one modified file")
	}
}

func TestCheckHandler_Handle_DeletedFile(t *testing.T) {
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"main.go":  "package main\nfunc main() {}",
		"utils.go": "package main\nfunc helper() {}",
		"go.mod":   "module example\n\ngo 1.21",
	})

	analysisCache := cache.NewCache()
	files, err := cache.ScanFiles(repoPath, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("failed to scan files: %v", err)
	}
	agentResults := map[string]bool{
		"structure_analyzer": true,
	}
	analysisCache.UpdateAfterAnalysis(repoPath, files, agentResults)
	if err := analysisCache.Save(repoPath); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	utilsPath := filepath.Join(repoPath, "utils.go")
	if err := os.Remove(utilsPath); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: repoPath,
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.HasDrift {
		t.Error("expected HasDrift to be true")
	}

	if len(report.DeletedFiles) == 0 {
		t.Error("expected at least one deleted file")
	}

	foundUtilsGo := false
	for _, f := range report.DeletedFiles {
		if f == "utils.go" {
			foundUtilsGo = true
			break
		}
	}
	if !foundUtilsGo {
		t.Errorf("expected utils.go in deleted files, got %v", report.DeletedFiles)
	}
}

func TestCheckHandler_FormatTextReport(t *testing.T) {
	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: "/test/path",
		},
		Verbose: true,
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report := &DriftReport{
		HasDrift:      true,
		Severity:      DriftSeverityModerate,
		LastAnalysis:  time.Now().Add(-24 * time.Hour),
		NewFiles:      []string{"new_file.go"},
		ModifiedFiles: []string{"modified.go"},
		AgentStatus: []AgentDriftStatus{
			{
				Name:        "structure_analyzer",
				DisplayName: "Structure Analysis",
				NeedsRerun:  true,
				RerunReason: "1 affected file(s) changed",
			},
			{
				Name:         "dependency_analyzer",
				DisplayName:  "Dependency Analysis",
				OutputExists: true,
			},
		},
		Summary:        "2 file(s) changed; 1 agent(s) need re-run",
		Recommendation: "Run 'gendocs analyze' soon to update documentation",
	}

	output := handler.FormatTextReport(report)

	expectedStrings := []string{
		"Documentation Drift Report",
		"Status: Moderate",
		"File Changes",
		"New: 1",
		"Modified: 1",
		"new_file.go",
		"modified.go",
		"Structure Analysis",
		"needs re-run",
		"Dependency Analysis",
		"up to date",
	}

	for _, expected := range expectedStrings {
		if !containsSubstring(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func TestCheckHandler_FormatJSONReport(t *testing.T) {
	cfg := config.CheckConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: "/test/path",
		},
	}

	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	report := &DriftReport{
		HasDrift: true,
		Severity: DriftSeverityMinor,
		NewFiles: []string{"test.go"},
		Summary:  "1 file changed",
	}

	output, err := handler.FormatJSONReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed DriftReport
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.HasDrift != report.HasDrift {
		t.Errorf("expected HasDrift %v, got %v", report.HasDrift, parsed.HasDrift)
	}

	if parsed.Severity != report.Severity {
		t.Errorf("expected Severity %s, got %s", report.Severity, parsed.Severity)
	}

	if len(parsed.NewFiles) != 1 || parsed.NewFiles[0] != "test.go" {
		t.Errorf("expected NewFiles [test.go], got %v", parsed.NewFiles)
	}
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		name     string
		report   *DriftReport
		expected DriftSeverity
	}{
		{
			name:     "no drift",
			report:   &DriftReport{HasDrift: false},
			expected: DriftSeverityNone,
		},
		{
			name: "minor drift - few files",
			report: &DriftReport{
				HasDrift:    true,
				NewFiles:    []string{"a.go", "b.go"},
				AgentStatus: []AgentDriftStatus{{NeedsRerun: true}},
			},
			expected: DriftSeverityMinor,
		},
		{
			name: "moderate drift - more files",
			report: &DriftReport{
				HasDrift:    true,
				NewFiles:    make([]string, 15),
				AgentStatus: []AgentDriftStatus{{NeedsRerun: true}},
			},
			expected: DriftSeverityModerate,
		},
		{
			name: "major drift - many agents",
			report: &DriftReport{
				HasDrift: true,
				NewFiles: []string{"a.go"},
				AgentStatus: []AgentDriftStatus{
					{NeedsRerun: true},
					{NeedsRerun: true},
					{NeedsRerun: true},
					{NeedsRerun: true},
				},
			},
			expected: DriftSeverityMajor,
		},
	}

	cfg := config.CheckConfig{}
	logger := logging.NewNopLogger()
	handler := NewCheckHandler(cfg, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.calculateSeverity(tt.report)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		filename string
		pattern  string
		expected bool
	}{
		{"main.go", "*.go", true},
		{"main.py", "*.go", false},
		{"handler.go", "*handler*", true},
		{"api_handler.go", "*handler*", true},
		{"main.go", "*handler*", false},
		{"go.mod", "go.mod", true},
		{"go.sum", "go.mod", false},
		{"test_file.go", "*_test.go", false},
		{"file_test.go", "*_test.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename+"_"+tt.pattern, func(t *testing.T) {
			got := matchesPattern(tt.filename, tt.pattern)
			if got != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v",
					tt.filename, tt.pattern, got, tt.expected)
			}
		})
	}
}

func TestLimitSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		limit    int
		expected int
	}{
		{"under limit", []string{"a", "b"}, 5, 2},
		{"at limit", []string{"a", "b", "c"}, 3, 3},
		{"over limit", []string{"a", "b", "c", "d", "e"}, 3, 3},
		{"empty slice", []string{}, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limitSlice(tt.slice, tt.limit)
			if len(result) != tt.expected {
				t.Errorf("expected length %d, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"none", "None"},
		{"minor", "Minor"},
		{"", ""},
		{"MAJOR", "MAJOR"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toTitleCase(tt.input)
			if got != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
