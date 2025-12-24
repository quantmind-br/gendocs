package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("##FAFAFA")).
			Background(lipgloss.Color("##7D56F4")).
			Padding(0, 1)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)
)

// Step represents a configuration step
type Step int

const (
	StepProvider Step = iota
	StepAPIKey
	StepModel
	StepBaseURL
	StepConfirm
	StepSave
	StepComplete
)

func (s Step) String() string {
	switch s {
	case StepProvider:
		return "Provider Selection"
	case StepAPIKey:
		return "API Key"
	case StepModel:
		return "Model"
	case StepBaseURL:
		return "Base URL (Optional)"
	case StepConfirm:
		return "Confirm"
	case StepSave:
		return "Save"
	case StepComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// Model holds the TUI state
type Model struct {
	Step            Step
	Provider        string
	APIKey          string
	Model           string
	BaseURL         string
	Quitting        bool
	ConfigPath      string
	SavedConfig     bool
	Err             error
	detectedEnvVars map[string]string // Environment variables detected at startup
	// Text inputs for user input (exported fields)
	APIKeyInput  textinput.Model
	ModelInput   textinput.Model
	BaseURLInput textinput.Model
}

// ConfigResult holds the final configuration result
type ConfigResult struct {
	Saved bool
	Path  string
	Error error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	m.detectedEnvVars = detectEnvironmentVariables()
	return textinput.Blink
}

// detectEnvironmentVariables scans for common config environment variables
func detectEnvironmentVariables() map[string]string {
	detected := make(map[string]string)

	envVars := []string{
		"ANALYZER_LLM_PROVIDER",
		"ANALYZER_LLM_MODEL",
		"ANALYZER_LLM_API_KEY",
		"DOCUMENTER_LLM_PROVIDER",
		"DOCUMENTER_LLM_MODEL",
		"DOCUMENTER_LLM_API_KEY",
		"GITLAB_OAUTH_TOKEN",
	}

	for _, key := range envVars {
		if val := os.Getenv(key); val != "" {
			// Mask API keys for display
			if strings.Contains(key, "API_KEY") || strings.Contains(key, "TOKEN") {
				detected[key] = maskSecret(val)
			} else {
				detected[key] = val
			}
		}
	}

	return detected
}

// maskSecret masks sensitive strings for display
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "enter":
			// Handle Enter key based on current step
			switch m.Step {
			case StepProvider:
				// Only advance if a provider is selected
				if m.Provider != "" {
					m.Step = StepAPIKey
					m.APIKeyInput.Focus()
					m.ModelInput.Blur()
					m.BaseURLInput.Blur()
				}
			case StepAPIKey:
				m.APIKey = m.APIKeyInput.Value()
				if m.APIKey != "" {
					m.Step = StepModel
					m.APIKeyInput.Blur()
					m.ModelInput.Focus()
					m.BaseURLInput.Blur()
				}
			case StepModel:
				inputModel := m.ModelInput.Value()
				if inputModel != "" {
					m.Model = inputModel
				} else if m.Model == "" {
					// Set default model based on provider
					switch m.Provider {
					case "openai":
						m.Model = "gpt-4o"
					case "anthropic":
						m.Model = "claude-3-5-sonnet-20241022"
					case "gemini":
						m.Model = "gemini-1.5-pro"
					}
				}
				m.Step = StepBaseURL
				m.APIKeyInput.Blur()
				m.ModelInput.Blur()
				m.BaseURLInput.Focus()
			case StepBaseURL:
				m.BaseURL = m.BaseURLInput.Value()
				m.Step = StepConfirm
				m.APIKeyInput.Blur()
				m.ModelInput.Blur()
				m.BaseURLInput.Blur()
			case StepConfirm:
				// Enter on confirm step is handled by y/n
			}
		case "1":
			if m.Step == StepProvider {
				m.Provider = "openai"
				m.Model = "gpt-4o"
			}
		case "2":
			if m.Step == StepProvider {
				m.Provider = "anthropic"
				m.Model = "claude-3-5-sonnet-20241022"
			}
		case "3":
			if m.Step == StepProvider {
				m.Provider = "gemini"
				m.Model = "gemini-1.5-pro"
			}
		case "y", "Y":
			if m.Step == StepConfirm {
				m.saveConfig()
				m.Step = StepComplete
			}
		case "n", "N":
			if m.Step == StepConfirm {
				m.Step = StepProvider
			}
		case "esc":
			if m.Step == StepModel {
				m.Step = StepAPIKey
				m.APIKeyInput.Focus()
				m.ModelInput.Blur()
				m.BaseURLInput.Blur()
			} else if m.Step == StepBaseURL {
				m.Step = StepModel
				m.APIKeyInput.Blur()
				m.ModelInput.Focus()
				m.BaseURLInput.Blur()
			}
		}
	}

	// Update text inputs based on current step
	var cmd tea.Cmd
	switch m.Step {
	case StepAPIKey:
		m.APIKeyInput, cmd = m.APIKeyInput.Update(msg)
	case StepModel:
		m.ModelInput, cmd = m.ModelInput.Update(msg)
	case StepBaseURL:
		m.BaseURLInput, cmd = m.BaseURLInput.Update(msg)
	}

	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	if m.Quitting {
		return "Exiting...\n"
	}

	if m.Step == StepComplete {
		if m.Err != nil {
			return fmt.Sprintf("\n%s\n\nError saving configuration: %v\n\nPress any key to exit...",
				errorStyle.Render("Configuration Failed"), m.Err)
		}
		return fmt.Sprintf("\n%s\n\nConfiguration saved to: %s\n\nPress any key to exit...",
			successStyle.Render("Configuration Saved Successfully!"), m.ConfigPath)
	}

	var s string

	// Title
	s += titleStyle.Render(" Gendocs Configuration Wizard ") + "\n\n"

	// Current step indicator
	stepNum := int(m.Step) + 1
	s += fmt.Sprintf("Step %d/5: %s\n\n", stepNum, m.Step.String())

	// Render current step content
	switch m.Step {
	case StepProvider:
		s += m.renderProviderSelection()
	case StepAPIKey:
		s += m.renderAPIKeyInput()
	case StepModel:
		s += m.renderModelInput()
	case StepBaseURL:
		s += m.renderBaseURLInput()
	case StepConfirm:
		s += m.renderConfirm()
	}

	// Help text
	s += "\n\n"
	if m.Step == StepProvider {
		s += "1-3: Select provider  |  Enter: Continue  |  q: Quit"
	} else if m.Step == StepConfirm {
		s += "y: Yes (save)  |  n: No (go back)  |  q: Quit"
	} else {
		s += "Type input  |  Enter: Continue  |  Esc: Go back  |  q: Quit"
	}

	return s + "\n"
}

func (m Model) renderProviderSelection() string {
	s := "Select your LLM provider:\n\n"

	providers := []struct {
		key   string
		name  string
		model string
	}{
		{"1", "OpenAI", "gpt-4o, gpt-4o-mini, etc."},
		{"2", "Anthropic Claude", "claude-3-5-sonnet, claude-3-haiku, etc."},
		{"3", "Google Gemini", "gemini-1.5-pro, gemini-pro, etc."},
	}

	for _, p := range providers {
		prefix := " "
		if m.Provider == getProviderFromKey(p.key) {
			prefix = highlightStyle.Render("✓")
		}
		s += fmt.Sprintf("%s %s. %s (%s)\n", prefix, p.key, p.name, p.model)
	}

	// Show detected environment provider if available
	envKey := "ANALYZER_LLM_PROVIDER"
	if detected, ok := m.detectedEnvVars[envKey]; ok {
		s += fmt.Sprintf("\n💡 Found %s=%s in environment\n", envKey, detected)
	}

	return s
}

func (m Model) renderAPIKeyInput() string {
	s := fmt.Sprintf("Enter your API key for %s:\n\n%s\n\n",
		highlightStyle.Render(m.Provider),
		m.APIKeyInput.View())

	// Show detected environment variable if available
	envKey := "ANALYZER_LLM_API_KEY"
	if detected, ok := m.detectedEnvVars[envKey]; ok {
		s += fmt.Sprintf("\n💡 Found %s=%s in environment\n", envKey, detected)
		s += "   (Will be used if you leave this empty)\n"
	}

	s += "\n(Press Enter when done)"
	return s
}

func (m Model) renderModelInput() string {
	defaultModel := m.Model
	if defaultModel == "" {
		switch m.Provider {
		case "openai":
			defaultModel = "gpt-4o"
		case "anthropic":
			defaultModel = "claude-3-5-sonnet-20241022"
		case "gemini":
			defaultModel = "gemini-1.5-pro"
		default:
			defaultModel = "<default>"
		}
	}

	s := fmt.Sprintf("Enter model name (or press Enter for default %s):\n\n%s",
		highlightStyle.Render(defaultModel),
		m.ModelInput.View())

	// Show detected environment variable if available
	envKey := "ANALYZER_LLM_MODEL"
	if detected, ok := m.detectedEnvVars[envKey]; ok {
		s += fmt.Sprintf("\n\n💡 Found %s=%s in environment", envKey, detected)
	}

	return s
}

func (m Model) renderBaseURLInput() string {
	return fmt.Sprintf("Enter base URL (optional, press Enter to skip):\n\n%s\n\nLeave empty for provider default.",
		m.BaseURLInput.View())
}

func (m Model) renderConfirm() string {
	s := "Review your configuration:\n\n"
	s += fmt.Sprintf("  Provider:   %s\n", highlightStyle.Render(m.Provider))
	s += fmt.Sprintf("  Model:      %s\n", highlightStyle.Render(m.Model))
	if m.BaseURL != "" {
		s += fmt.Sprintf("  Base URL:   %s\n", highlightStyle.Render(m.BaseURL))
	}
	s += "\nSave this configuration?"
	return s
}

func (m Model) saveConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.Err = err
		return
	}

	m.ConfigPath = filepath.Join(homeDir, ".gendocs.yaml")

	// Create YAML configuration
	configYAML := fmt.Sprintf("# Gendocs Global Configuration\nanalyzer:\n  llm:\n    provider: %s\n    api_key: %s\n    model: %s\n",
		m.Provider, m.APIKey, m.Model)

	if m.BaseURL != "" {
		configYAML += fmt.Sprintf("    base_url: %s\n", m.BaseURL)
	}

	if err := os.WriteFile(m.ConfigPath, []byte(configYAML), 0600); err != nil {
		m.Err = err
		return
	}

	m.SavedConfig = true
}

func getProviderFromKey(key string) string {
	switch key {
	case "1":
		return "openai"
	case "2":
		return "anthropic"
	case "3":
		return "gemini"
	default:
		return ""
	}
}

// GetConfigPath returns the path where config was saved
func (m Model) GetConfigPath() string {
	return m.ConfigPath
}
