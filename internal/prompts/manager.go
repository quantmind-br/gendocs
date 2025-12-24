package prompts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	textTemplate "text/template"

	"gopkg.in/yaml.v3"
)

// Manager handles loading and rendering prompt templates
type Manager struct {
	prompts map[string]string
	sources map[string]string // Track which file provided each prompt (for debugging)
}

// PromptTemplate represents a prompt with system and user components
type PromptTemplate struct {
	SystemPrompt string `yaml:"system_prompt"`
	UserPrompt   string `yaml:"user_prompt"`
}

// NewManager creates a new prompt manager by loading prompts from a directory
func NewManager(promptsDir string) (*Manager, error) {
	pm := &Manager{
		prompts: make(map[string]string),
		sources: make(map[string]string),
	}

	// Load from single directory (backward compatibility)
	if err := pm.loadDirectory(promptsDir, "system"); err != nil {
		return nil, err
	}

	return pm, nil
}

// NewManagerWithOverrides creates manager with system + project overrides
func NewManagerWithOverrides(systemDir, projectDir string) (*Manager, error) {
	pm := &Manager{
		prompts: make(map[string]string),
		sources: make(map[string]string),
	}

	// 1. Load system prompts first (baseline)
	if err := pm.loadDirectory(systemDir, "system"); err != nil {
		return nil, fmt.Errorf("failed to load system prompts: %w", err)
	}

	// 2. Load project prompts (overrides) if directory exists
	if projectDir != "" {
		if _, err := os.Stat(projectDir); err == nil {
			if err := pm.loadDirectory(projectDir, "project"); err != nil {
				return nil, fmt.Errorf("failed to load project prompts: %w", err)
			}
		}
		// If project dir doesn't exist, that's OK - no overrides
	}

	// 3. Validate required prompts exist
	if err := pm.validateRequiredPrompts(); err != nil {
		return nil, err
	}

	return pm, nil
}

// loadDirectory loads all YAML files from a directory
func (pm *Manager) loadDirectory(dir, source string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		var prompts map[string]string
		if err := yaml.Unmarshal(data, &prompts); err != nil {
			return fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		// Merge into main map (later loads override earlier)
		for key, value := range prompts {
			pm.prompts[key] = value
			pm.sources[key] = fmt.Sprintf("%s:%s", source, entry.Name())
		}
	}

	return nil
}

// validateRequiredPrompts ensures critical prompts exist
func (pm *Manager) validateRequiredPrompts() error {
	required := []string{
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

	var missing []string
	for _, key := range required {
		if _, ok := pm.prompts[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required prompts: %v", missing)
	}

	return nil
}

// NewManagerFromMap creates a prompt manager from a map (useful for testing)
func NewManagerFromMap(prompts map[string]string) *Manager {
	sources := make(map[string]string)
	for key := range prompts {
		sources[key] = "test:map"
	}
	return &Manager{
		prompts: prompts,
		sources: sources,
	}
}

// Get returns a raw prompt by name
func (pm *Manager) Get(name string) (string, error) {
	prompt, ok := pm.prompts[name]
	if !ok {
		return "", fmt.Errorf("prompt '%s' not found (available: %v)", name, pm.getAvailableNames())
	}
	return prompt, nil
}

// Render renders a prompt template with the given variables
func (pm *Manager) Render(name string, vars map[string]interface{}) (string, error) {
	promptTemplate, err := pm.Get(name)
	if err != nil {
		return "", err
	}

	// Parse and execute template
	tmpl, err := textTemplate.New(name).Option("missingkey=error").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template '%s': %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template '%s': %w", name, err)
	}

	return buf.String(), nil
}

// GetPromptTemplate returns a PromptTemplate with system and user prompts
func (pm *Manager) GetPromptTemplate(name string) (*PromptTemplate, error) {
	systemPrompt, err := pm.Render(name+"_system_prompt", nil)
	if err != nil {
		// Try without suffix
		systemPrompt, err = pm.Get(name + "_system")
		if err != nil {
			return nil, fmt.Errorf("system prompt '%s' not found", name)
		}
	}

	userPrompt, err := pm.Get(name + "_user_prompt")
	if err != nil {
		return nil, fmt.Errorf("user prompt '%s' not found", name)
	}

	return &PromptTemplate{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}, nil
}

// RenderTemplate renders both system and user prompts with variables
func (pm *Manager) RenderTemplate(name string, vars map[string]interface{}) (*PromptTemplate, error) {
	template, err := pm.GetPromptTemplate(name)
	if err != nil {
		return nil, err
	}

	// Render system prompt with variables
	systemTmpl, err := textTemplate.New("system").Parse(template.SystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system prompt: %w", err)
	}

	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, vars); err != nil {
		return nil, fmt.Errorf("failed to render system prompt: %w", err)
	}

	// Render user prompt with variables
	userTmpl, err := textTemplate.New("user").Parse(template.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user prompt: %w", err)
	}

	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, vars); err != nil {
		return nil, fmt.Errorf("failed to render user prompt: %w", err)
	}

	return &PromptTemplate{
		SystemPrompt: systemBuf.String(),
		UserPrompt:   userBuf.String(),
	}, nil
}

// getAvailableNames returns a list of available prompt names
func (pm *Manager) getAvailableNames() []string {
	names := make([]string, 0, len(pm.prompts))
	for name := range pm.prompts {
		names = append(names, name)
	}
	return names
}

// HasPrompt checks if a prompt exists
func (pm *Manager) HasPrompt(name string) bool {
	_, ok := pm.prompts[name]
	return ok
}

// GetSource returns which file provided a prompt (for debugging)
func (pm *Manager) GetSource(name string) string {
	if source, ok := pm.sources[name]; ok {
		return source
	}
	return "unknown"
}

// ListOverrides returns all prompts that were overridden from project
func (pm *Manager) ListOverrides() []string {
	var overrides []string
	for key, source := range pm.sources {
		if len(source) > 0 && source[0:7] == "project" {
			overrides = append(overrides, key)
		}
	}
	return overrides
}

// CountPrompts returns the total number of loaded prompts
func (pm *Manager) CountPrompts() int {
	return len(pm.prompts)
}
