package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManagerWithOverrides_SystemOnly(t *testing.T) {
	// Create system prompts directory
	systemDir := t.TempDir()

	systemYAML := `
structure_analyzer_system: "System structure prompt"
structure_analyzer_user: "User structure prompt"
dependency_analyzer_system: "System dependency prompt"
dependency_analyzer_user: "User dependency prompt"
data_flow_analyzer_system: "System data flow prompt"
data_flow_analyzer_user: "User data flow prompt"
request_flow_analyzer_system: "System request flow prompt"
request_flow_analyzer_user: "User request flow prompt"
api_analyzer_system: "System API prompt"
api_analyzer_user: "User API prompt"
documenter_system_prompt: "System doc prompt"
documenter_user_prompt: "User doc prompt"
ai_rules_system_prompt: "System rules prompt"
ai_rules_user_prompt: "User rules prompt"
`
	os.WriteFile(filepath.Join(systemDir, "analyzer.yaml"), []byte(systemYAML), 0644)

	// Create manager with no project overrides
	mgr, err := NewManagerWithOverrides(systemDir, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify system prompts loaded
	prompt, err := mgr.Get("structure_analyzer_system")
	if err != nil {
		t.Fatalf("Expected prompt to exist, got error: %v", err)
	}

	if prompt != "System structure prompt" {
		t.Errorf("Expected 'System structure prompt', got '%s'", prompt)
	}

	// Verify source tracking
	source := mgr.GetSource("structure_analyzer_system")
	if !strings.Contains(source, "system") {
		t.Errorf("Expected source to contain 'system', got '%s'", source)
	}
}

func TestNewManagerWithOverrides_WithProjectOverrides(t *testing.T) {
	// Create system prompts directory
	systemDir := t.TempDir()

	systemYAML := `
structure_analyzer_system: "System structure prompt"
structure_analyzer_user: "User structure prompt"
dependency_analyzer_system: "System dependency prompt"
dependency_analyzer_user: "User dependency prompt"
data_flow_analyzer_system: "System data flow prompt"
data_flow_analyzer_user: "User data flow prompt"
request_flow_analyzer_system: "System request flow prompt"
request_flow_analyzer_user: "User request flow prompt"
api_analyzer_system: "System API prompt"
api_analyzer_user: "User API prompt"
documenter_system_prompt: "System doc prompt"
documenter_user_prompt: "User doc prompt"
ai_rules_system_prompt: "System rules prompt"
ai_rules_user_prompt: "User rules prompt"
test_only_system: "Only in system"
`
	os.WriteFile(filepath.Join(systemDir, "analyzer.yaml"), []byte(systemYAML), 0644)

	// Create project prompts directory with overrides
	projectDir := t.TempDir()

	projectYAML := `
structure_analyzer_system: "CUSTOM structure prompt"
documenter_system_prompt: "CUSTOM doc prompt"
test_only_project: "Only in project"
`
	os.WriteFile(filepath.Join(projectDir, "custom.yaml"), []byte(projectYAML), 0644)

	// Create manager with overrides
	mgr, err := NewManagerWithOverrides(systemDir, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify overridden prompts
	prompt, err := mgr.Get("structure_analyzer_system")
	if err != nil {
		t.Fatalf("Expected prompt to exist, got error: %v", err)
	}

	if prompt != "CUSTOM structure prompt" {
		t.Errorf("Expected 'CUSTOM structure prompt', got '%s'", prompt)
	}

	// Verify override source tracking
	source := mgr.GetSource("structure_analyzer_system")
	if !strings.Contains(source, "project") {
		t.Errorf("Expected source to contain 'project' for overridden prompt, got '%s'", source)
	}

	// Verify non-overridden prompts still work
	prompt2, err := mgr.Get("dependency_analyzer_system")
	if err != nil {
		t.Fatalf("Expected prompt to exist, got error: %v", err)
	}

	if prompt2 != "System dependency prompt" {
		t.Errorf("Expected 'System dependency prompt', got '%s'", prompt2)
	}

	source2 := mgr.GetSource("dependency_analyzer_system")
	if !strings.Contains(source2, "system") {
		t.Errorf("Expected source to contain 'system' for non-overridden prompt, got '%s'", source2)
	}

	// Verify project-only prompts exist
	prompt3, err := mgr.Get("test_only_project")
	if err != nil {
		t.Fatalf("Expected project-only prompt to exist, got error: %v", err)
	}

	if prompt3 != "Only in project" {
		t.Errorf("Expected 'Only in project', got '%s'", prompt3)
	}

	// Verify system-only prompts exist
	prompt4, err := mgr.Get("test_only_system")
	if err != nil {
		t.Fatalf("Expected system-only prompt to exist, got error: %v", err)
	}

	if prompt4 != "Only in system" {
		t.Errorf("Expected 'Only in system', got '%s'", prompt4)
	}
}

func TestNewManagerWithOverrides_ListOverrides(t *testing.T) {
	systemDir := t.TempDir()

	systemYAML := `
structure_analyzer_system: "System"
dependency_analyzer_system: "System"
data_flow_analyzer_system: "System"
request_flow_analyzer_system: "System"
api_analyzer_system: "System"
structure_analyzer_user: "System"
dependency_analyzer_user: "System"
data_flow_analyzer_user: "System"
request_flow_analyzer_user: "System"
api_analyzer_user: "System"
documenter_system_prompt: "System"
documenter_user_prompt: "System"
ai_rules_system_prompt: "System"
ai_rules_user_prompt: "System"
`
	os.WriteFile(filepath.Join(systemDir, "analyzer.yaml"), []byte(systemYAML), 0644)

	projectDir := t.TempDir()
	projectYAML := `
structure_analyzer_system: "Custom"
documenter_system_prompt: "Custom"
`
	os.WriteFile(filepath.Join(projectDir, "custom.yaml"), []byte(projectYAML), 0644)

	mgr, err := NewManagerWithOverrides(systemDir, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	overrides := mgr.ListOverrides()
	if len(overrides) != 2 {
		t.Errorf("Expected 2 overrides, got %d: %v", len(overrides), overrides)
	}

	// Check that overrides contain the expected prompts
	hasStructure := false
	hasDocumenter := false
	for _, override := range overrides {
		if override == "structure_analyzer_system" {
			hasStructure = true
		}
		if override == "documenter_system_prompt" {
			hasDocumenter = true
		}
	}

	if !hasStructure {
		t.Error("Expected overrides to contain 'structure_analyzer_system'")
	}

	if !hasDocumenter {
		t.Error("Expected overrides to contain 'documenter_system_prompt'")
	}
}

func TestNewManagerWithOverrides_NonexistentProjectDir(t *testing.T) {
	systemDir := t.TempDir()

	systemYAML := `
structure_analyzer_system: "System"
dependency_analyzer_system: "System"
data_flow_analyzer_system: "System"
request_flow_analyzer_system: "System"
api_analyzer_system: "System"
structure_analyzer_user: "System"
dependency_analyzer_user: "System"
data_flow_analyzer_user: "System"
request_flow_analyzer_user: "System"
api_analyzer_user: "System"
documenter_system_prompt: "System"
documenter_user_prompt: "System"
ai_rules_system_prompt: "System"
ai_rules_user_prompt: "System"
`
	os.WriteFile(filepath.Join(systemDir, "analyzer.yaml"), []byte(systemYAML), 0644)

	// Use nonexistent project directory
	projectDir := filepath.Join(t.TempDir(), "nonexistent")

	mgr, err := NewManagerWithOverrides(systemDir, projectDir)
	if err != nil {
		t.Fatalf("Expected no error with nonexistent project dir, got %v", err)
	}

	// Should work fine, just no overrides
	overrides := mgr.ListOverrides()
	if len(overrides) != 0 {
		t.Errorf("Expected 0 overrides with nonexistent project dir, got %d", len(overrides))
	}

	// System prompts should still be loaded
	if !mgr.HasPrompt("structure_analyzer_system") {
		t.Error("Expected system prompt to be loaded even without project dir")
	}
}

func TestNewManagerWithOverrides_MissingRequiredPrompt(t *testing.T) {
	systemDir := t.TempDir()

	// Create incomplete system prompts (missing required prompts)
	incompleteYAML := `
structure_analyzer_system: "System"
# Missing other required prompts
`
	os.WriteFile(filepath.Join(systemDir, "incomplete.yaml"), []byte(incompleteYAML), 0644)

	_, err := NewManagerWithOverrides(systemDir, "")
	if err == nil {
		t.Fatal("Expected error for missing required prompts, got nil")
	}

	if !strings.Contains(err.Error(), "missing required prompts") {
		t.Errorf("Expected error message to mention missing prompts, got: %v", err)
	}
}

func TestManager_CountPrompts(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"prompt1": "value1",
		"prompt2": "value2",
		"prompt3": "value3",
	})

	count := mgr.CountPrompts()
	if count != 3 {
		t.Errorf("Expected count of 3, got %d", count)
	}
}

func TestManager_GetSource(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"test": "value",
	})

	source := mgr.GetSource("test")
	if source != "test:map" {
		t.Errorf("Expected 'test:map', got '%s'", source)
	}

	// Test unknown prompt
	unknownSource := mgr.GetSource("nonexistent")
	if unknownSource != "unknown" {
		t.Errorf("Expected 'unknown' for nonexistent prompt, got '%s'", unknownSource)
	}
}
