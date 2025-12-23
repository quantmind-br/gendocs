package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
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
	Step        Step
	Provider    string
	APIKey      string
	Model       string
	BaseURL     string
	Quitting    bool
	ConfigPath  string
	SavedConfig bool
	Err         error
}

// ConfigResult holds the final configuration result
type ConfigResult struct {
	Saved bool
	Path  string
	Error error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyMsg := msg.String()
		switch keyMsg {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "enter", " ":
			m.advanceStep()
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
				m.Step = StepSave
			}
		case "n", "N":
			if m.Step == StepConfirm {
				m.Step = StepProvider
			}
		case "esc":
			if m.Step > StepProvider && m.Step < StepConfirm {
				m.Step--
			}
		}
	}

	return m, nil
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
	s += fmt.Sprintf("Step %d/5: %s\n\n", m.Step, m.Step.String())

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
	case StepSave:
		s += m.renderSaving()
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

	return s
}

func (m Model) renderAPIKeyInput() string {
	return fmt.Sprintf("Enter your API key for %s:\n\n%s\n\n(Press Enter when done)",
		highlightStyle.Render(m.Provider),
		highlightStyle.Render("••••••••••••••••"))
}

func (m Model) renderModelInput() string {
	defaultModel := m.Model
	if defaultModel == "" {
		defaultModel = "<default>"
	}
	return fmt.Sprintf("Enter model name (or press Enter for default %s):\n\n%s",
		highlightStyle.Render(defaultModel),
		highlightStyle.Render(m.Model))
}

func (m Model) renderBaseURLInput() string {
	defaultURL := "<provider default>"
	if m.BaseURL != "" {
		defaultURL = m.BaseURL
	}
	return fmt.Sprintf("Enter base URL (optional, press Enter to skip):\n\n%s\n\nLeave empty for provider default.",
		highlightStyle.Render(defaultURL))
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

func (m Model) renderSaving() string {
	return "Saving configuration..."
}

func (m Model) advanceStep() {
	m.Step++
	
	// Auto-advance model selection based on provider
	if m.Step == StepModel && m.Model == "" {
		switch m.Provider {
		case "openai":
			m.Model = "gpt-4o"
		case "anthropic":
			m.Model = "claude-3-5-sonnet-20241022"
		case "gemini":
			m.Model = "gemini-1.5-pro"
		}
	}

	// Save configuration on save step
	if m.Step == StepSave {
		m.saveConfig()
		m.Step = StepComplete
	}
}

func (m Model) saveConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.Err = err
		return
	}

	m.ConfigPath = filepath.Join(homeDir, ".gendocs.yaml")

	// Create YAML configuration
	configYAML := fmt.Sprintf("# Gendocs Global Configuration\nanalyzer:\n  llm:\n    provider: %s\n    model: %s\n",
		m.Provider, m.Model)

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
