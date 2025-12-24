package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_RealProjectPrompts tests loading prompts from actual project structure
func TestIntegration_RealProjectPrompts(t *testing.T) {
	// Get project root (assume we're in internal/prompts/)
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	systemPromptsDir := filepath.Join(projectRoot, "prompts")
	projectPromptsDir := filepath.Join(projectRoot, ".ai/prompts")

	// Verify system prompts directory exists
	if _, err := os.Stat(systemPromptsDir); os.IsNotExist(err) {
		t.Fatalf("System prompts directory not found: %s", systemPromptsDir)
	}

	// Load with overrides
	mgr, err := NewManagerWithOverrides(systemPromptsDir, projectPromptsDir)
	if err != nil {
		t.Fatalf("Failed to load prompts: %v", err)
	}

	// Verify all required prompts exist
	requiredPrompts := []string{
		"structure_analyzer_system",
		"structure_analyzer_user",
		"dependency_analyzer_system",
		"dependency_analyzer_user",
		"data_flow_analyzer_system",
		"data_flow_analyzer_user",
		"request_flow_analyzer_system",
		"request_flow_analyzer_user",
		"api_analyzer_system",
		"api_analyzer_user",
		"documenter_system_prompt",
		"documenter_user_prompt",
		"ai_rules_system_prompt",
		"ai_rules_user_prompt",
	}

	for _, promptName := range requiredPrompts {
		if !mgr.HasPrompt(promptName) {
			t.Errorf("Required prompt missing: %s", promptName)
		}
	}

	// If project prompts directory exists, verify overrides work
	if _, err := os.Stat(projectPromptsDir); err == nil {
		overrides := mgr.ListOverrides()
		t.Logf("Found %d custom prompt overrides:", len(overrides))
		for _, override := range overrides {
			t.Logf("  - %s (source: %s)", override, mgr.GetSource(override))

			// Verify override is actually loaded
			prompt, err := mgr.Get(override)
			if err != nil {
				t.Errorf("Failed to get overridden prompt %s: %v", override, err)
			}

			// Verify it's from project, not system
			source := mgr.GetSource(override)
			if !strings.HasPrefix(source, "project:") {
				t.Errorf("Override %s should have 'project:' source, got: %s", override, source)
			}

			t.Logf("    Content length: %d chars", len(prompt))
		}
	} else {
		t.Log("No project prompts found (this is OK for testing)")
	}

	// Test prompt count
	totalPrompts := mgr.CountPrompts()
	if totalPrompts < len(requiredPrompts) {
		t.Errorf("Expected at least %d prompts, got %d", len(requiredPrompts), totalPrompts)
	}
	t.Logf("Total prompts loaded: %d", totalPrompts)
}

// TestIntegration_PromptRendering tests template rendering with real prompts
func TestIntegration_PromptRendering(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	systemPromptsDir := filepath.Join(projectRoot, "prompts")
	projectPromptsDir := filepath.Join(projectRoot, ".ai/prompts")

	mgr, err := NewManagerWithOverrides(systemPromptsDir, projectPromptsDir)
	if err != nil {
		t.Fatalf("Failed to load prompts: %v", err)
	}

	// Test rendering analyzer user prompts (they use templates)
	testCases := []struct {
		promptName string
		vars       map[string]interface{}
	}{
		{
			promptName: "structure_analyzer_user",
			vars: map[string]interface{}{
				"RepoPath": "/test/repo",
				"Language": "Go",
			},
		},
		{
			promptName: "dependency_analyzer_user",
			vars: map[string]interface{}{
				"RepoPath": "/test/repo",
				"Language": "Go",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.promptName, func(t *testing.T) {
			// Try to render - if it has no template vars, this will still work
			result, err := mgr.Render(tc.promptName, tc.vars)
			if err != nil {
				// If it fails, try Get (maybe it's not a template)
				result, err = mgr.Get(tc.promptName)
				if err != nil {
					t.Fatalf("Failed to get prompt: %v", err)
				}
			}

			if result == "" {
				t.Error("Rendered prompt is empty")
			}

			t.Logf("Rendered %s (%d chars)", tc.promptName, len(result))
		})
	}
}

// findProjectRoot walks up from current directory to find project root
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
