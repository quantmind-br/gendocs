package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManagerFromMap_Get(t *testing.T) {
	prompts := map[string]string{
		"test_system": "You are a test assistant",
		"test_user":   "Analyze the code",
	}

	mgr := NewManagerFromMap(prompts)

	// Test Get
	prompt, err := mgr.Get("test_system")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if prompt != "You are a test assistant" {
		t.Errorf("Expected 'You are a test assistant', got '%s'", prompt)
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"exists": "value",
	})

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent prompt, got nil")
	}
}

func TestManager_Render_WithVariables(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"template": "Path: {{.RepoPath}}, Workers: {{.Workers}}",
	})

	result, err := mgr.Render("template", map[string]interface{}{
		"RepoPath": "/test/path",
		"Workers":  4,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "Path: /test/path, Workers: 4"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestManager_Render_MissingVariable(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"template": "Path: {{.RepoPath}}",
	})

	// Render with empty vars - should error on missing variable
	_, err := mgr.Render("template", map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error for missing variable, got nil")
	}
}

func TestManager_Render_ComplexTemplate(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"complex": `Repository: {{.Repo}}
{{if .Debug}}Debug mode enabled{{end}}
Files: {{range .Files}}
  - {{.}}{{end}}`,
	})

	result, err := mgr.Render("complex", map[string]interface{}{
		"Repo":  "test-repo",
		"Debug": true,
		"Files": []string{"file1.go", "file2.go"},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !containsString(result, "Repository: test-repo") {
		t.Error("Expected 'Repository: test-repo' in result")
	}

	if !containsString(result, "Debug mode enabled") {
		t.Error("Expected 'Debug mode enabled' in result")
	}

	if !containsString(result, "file1.go") {
		t.Error("Expected 'file1.go' in result")
	}
}

func TestManager_HasPrompt(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"exists": "value",
	})

	if !mgr.HasPrompt("exists") {
		t.Error("Expected HasPrompt to return true for existing prompt")
	}

	if mgr.HasPrompt("nonexistent") {
		t.Error("Expected HasPrompt to return false for non-existent prompt")
	}
}

func TestNewManager_LoadFromDirectory(t *testing.T) {
	// Create temp directory with YAML files
	tmpDir := t.TempDir()

	// Create analyzer.yaml
	analyzerYAML := `
test_system: "System prompt"
test_user: "User prompt"
`
	_ = os.WriteFile(filepath.Join(tmpDir, "analyzer.yaml"), []byte(analyzerYAML), 0644)

	// Create documenter.yml
	documenterYML := `
doc_system: "Doc system prompt"
doc_user: "Doc user prompt"
`
	_ = os.WriteFile(filepath.Join(tmpDir, "documenter.yml"), []byte(documenterYML), 0644)

	// Load manager
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error loading manager, got %v", err)
	}

	// Verify all prompts loaded
	prompts := []string{"test_system", "test_user", "doc_system", "doc_user"}
	for _, p := range prompts {
		if !mgr.HasPrompt(p) {
			t.Errorf("Expected prompt '%s' to be loaded", p)
		}
	}
}

func TestNewManager_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid YAML
	invalidYAML := `
test: this is not
  valid: yaml: structure
`
	_ = os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0644)

	_, err := NewManager(tmpDir)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestNewManager_DirectoryNotExists(t *testing.T) {
	_, err := NewManager("/nonexistent/directory")
	if err == nil {
		t.Fatal("Expected error for non-existent directory, got nil")
	}
}

func TestNewManager_IgnoresNonYAMLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create YAML file
	yamlContent := `
prompt1: "value1"
`
	_ = os.WriteFile(filepath.Join(tmpDir, "prompts.yaml"), []byte(yamlContent), 0644)

	// Create non-YAML files (should be ignored)
	_ = os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "script.sh"), []byte("#!/bin/bash"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0644)

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only have prompts from YAML file
	if !mgr.HasPrompt("prompt1") {
		t.Error("Expected 'prompt1' to be loaded from YAML")
	}

	// Other files should not create prompts
	if mgr.HasPrompt("readme.md") {
		t.Error("Non-YAML file should not be loaded as prompt")
	}
}

func TestNewManager_IgnoresSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create YAML in root
	_ = os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte("root_prompt: value\n"), 0644)

	// Create subdirectory with YAML (should be ignored by Walk behavior)
	subdir := filepath.Join(tmpDir, "subdir")
	_ = os.MkdirAll(subdir, 0755)
	_ = os.WriteFile(filepath.Join(subdir, "sub.yaml"), []byte("sub_prompt: value\n"), 0644)

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have root prompt
	if !mgr.HasPrompt("root_prompt") {
		t.Error("Expected 'root_prompt' from root directory")
	}

	// Current implementation uses ReadDir which doesn't recurse
	// So subdirectory prompts should NOT be loaded
	if mgr.HasPrompt("sub_prompt") {
		t.Error("Prompts from subdirectories should not be loaded")
	}
}

func TestManager_GetPromptTemplate(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"test_system_prompt": "System prompt content",
		"test_user_prompt":   "User prompt content",
	})

	template, err := mgr.GetPromptTemplate("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if template.SystemPrompt != "System prompt content" {
		t.Errorf("Expected system prompt, got '%s'", template.SystemPrompt)
	}

	if template.UserPrompt != "User prompt content" {
		t.Errorf("Expected user prompt, got '%s'", template.UserPrompt)
	}
}

func TestManager_GetPromptTemplate_NotFound(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"other_prompt": "value",
	})

	_, err := mgr.GetPromptTemplate("test")
	if err == nil {
		t.Fatal("Expected error for non-existent template, got nil")
	}
}

func TestManager_RenderTemplate(t *testing.T) {
	// RenderTemplate looks for name_system_prompt and name_user_prompt
	// But GetPromptTemplate tries to call Render on name_system_prompt first,
	// falling back to Get name_system. Let's provide both formats.
	mgr := NewManagerFromMap(map[string]string{
		"test_system":      "Analyze {{.Language}} code",
		"test_user_prompt": "Path: {{.RepoPath}}",
	})

	template, err := mgr.RenderTemplate("test", map[string]interface{}{
		"Language": "Go",
		"RepoPath": "/test",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if template.SystemPrompt != "Analyze Go code" {
		t.Errorf("Expected 'Analyze Go code', got '%s'", template.SystemPrompt)
	}

	if template.UserPrompt != "Path: /test" {
		t.Errorf("Expected 'Path: /test', got '%s'", template.UserPrompt)
	}
}

func TestManager_EmptyPrompt(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"empty": "",
	})

	prompt, err := mgr.Get("empty")
	if err != nil {
		t.Fatalf("Expected no error for empty prompt, got %v", err)
	}

	if prompt != "" {
		t.Errorf("Expected empty string, got '%s'", prompt)
	}
}

func TestManager_WhitespacePrompt(t *testing.T) {
	mgr := NewManagerFromMap(map[string]string{
		"whitespace": "   \n\t\n   ",
	})

	prompt, err := mgr.Get("whitespace")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should preserve whitespace exactly
	if prompt != "   \n\t\n   " {
		t.Errorf("Expected whitespace to be preserved, got '%s'", prompt)
	}
}

func TestManager_MultilinePrompt(t *testing.T) {
	multiline := `Line 1
Line 2
Line 3
`

	mgr := NewManagerFromMap(map[string]string{
		"multiline": multiline,
	})

	prompt, err := mgr.Get("multiline")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if prompt != multiline {
		t.Errorf("Expected multiline prompt to be preserved")
	}
}

func TestManager_SpecialCharacters(t *testing.T) {
	special := "Test with {{special}} {{.characters}} and $symbols @#%"

	mgr := NewManagerFromMap(map[string]string{
		"special": special,
	})

	prompt, err := mgr.Get("special")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if prompt != special {
		t.Errorf("Expected special characters to be preserved")
	}
}

// Helper function for string contains check
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle || len(needle) == 0 || findSubstring(haystack, needle))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
