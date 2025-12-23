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
	}

	// Load all YAML files from the prompts directory
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" {
			continue
		}

		filePath := filepath.Join(promptsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file %s: %w", filePath, err)
		}

		// Parse YAML - could be simple string -> string or nested
		var prompts map[string]string
		if err := yaml.Unmarshal(data, &prompts); err != nil {
			return nil, fmt.Errorf("failed to parse prompts from %s: %w", filePath, err)
		}

		// Merge into main prompts map
		for key, value := range prompts {
			pm.prompts[key] = value
		}
	}

	return pm, nil
}

// NewManagerFromMap creates a prompt manager from a map (useful for testing)
func NewManagerFromMap(prompts map[string]string) *Manager {
	return &Manager{
		prompts: prompts,
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
