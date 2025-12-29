package sections

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/types"
	"github.com/user/gendocs/internal/tui/dashboard/validation"
)

type LLMSectionModel struct {
	provider       components.DropdownModel
	model          components.TextFieldModel
	apiKey         components.MaskedInputModel
	baseURL        components.TextFieldModel
	temperature    components.TextFieldModel
	maxTokens      components.TextFieldModel
	timeout        components.TextFieldModel
	retries        components.TextFieldModel
	testConnButton components.ButtonModel

	focusIndex int
	testing    bool
}

type TestConnectionResultMsg struct {
	Success bool
	Message string
}

func NewLLMSection() *LLMSectionModel {
	providerOpts := []components.DropdownOption{
		{Value: "openai", Label: "OpenAI (GPT-4o, GPT-4)"},
		{Value: "anthropic", Label: "Anthropic (Claude)"},
		{Value: "gemini", Label: "Google (Gemini)"},
	}

	m := &LLMSectionModel{
		provider: components.NewDropdown("Provider", providerOpts, "Select your LLM provider"),
		model: components.NewTextField("Model",
			components.WithPlaceholder("e.g., gpt-4o, claude-3-5-sonnet"),
			components.WithRequired(),
			components.WithHelp("Model name for the selected provider")),
		apiKey: components.NewMaskedInput("API Key", "Your provider API key"),
		baseURL: components.NewTextField("Base URL",
			components.WithPlaceholder("https://api.openai.com/v1"),
			components.WithValidator(validation.ValidateURL()),
			components.WithHelp("Optional: Override API endpoint")),
		temperature: components.NewTextField("Temperature",
			components.WithPlaceholder("0.0"),
			components.WithValidator(validation.ValidateFloatRange(0.0, 2.0)),
			components.WithHelp("0.0 = deterministic, 2.0 = creative")),
		maxTokens: components.NewTextField("Max Tokens",
			components.WithPlaceholder("8192"),
			components.WithValidator(validation.ValidateIntRange(100, 128000)),
			components.WithHelp("Maximum response length")),
		timeout: components.NewTextField("Timeout (seconds)",
			components.WithPlaceholder("180"),
			components.WithValidator(validation.ValidateIntRange(1, 600)),
			components.WithHelp("Request timeout in seconds")),
		retries: components.NewTextField("Retries",
			components.WithPlaceholder("2"),
			components.WithValidator(validation.ValidateIntRange(0, 10)),
			components.WithHelp("Number of retry attempts on failure")),
	}

	m.testConnButton = components.NewButton(
		"Test Connection",
		m.testConnection,
		components.WithButtonHelp("Press Enter to test LLM connection"),
	)

	return m
}

func (m *LLMSectionModel) Title() string { return "LLM Provider Settings" }
func (m *LLMSectionModel) Icon() string  { return "🤖" }
func (m *LLMSectionModel) Description() string {
	return "Configure your AI model provider and parameters"
}

func (m *LLMSectionModel) Init() tea.Cmd {
	return nil
}

func (m *LLMSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TestConnectionResultMsg:
		m.testing = false
		m.testConnButton.SetLoading(false)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 9
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 8
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.provider, _ = m.provider.Update(msg)
	case 1:
		m.model, _ = m.model.Update(msg)
	case 2:
		m.apiKey, _ = m.apiKey.Update(msg)
	case 3:
		m.baseURL, _ = m.baseURL.Update(msg)
	case 4:
		m.temperature, _ = m.temperature.Update(msg)
	case 5:
		m.maxTokens, _ = m.maxTokens.Update(msg)
	case 6:
		m.timeout, _ = m.timeout.Update(msg)
	case 7:
		m.retries, _ = m.retries.Update(msg)
	case 8:
		var cmd tea.Cmd
		m.testConnButton, cmd = m.testConnButton.Update(msg)
		if cmd != nil {
			m.testing = true
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *LLMSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.provider.Blur()
	case 1:
		m.model.Blur()
	case 2:
		m.apiKey.Blur()
	case 3:
		m.baseURL.Blur()
	case 4:
		m.temperature.Blur()
	case 5:
		m.maxTokens.Blur()
	case 6:
		m.timeout.Blur()
	case 7:
		m.retries.Blur()
	case 8:
		m.testConnButton.Blur()
	}
}

func (m *LLMSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.provider.Focus()
	case 1:
		return m.model.Focus()
	case 2:
		return m.apiKey.Focus()
	case 3:
		return m.baseURL.Focus()
	case 4:
		return m.temperature.Focus()
	case 5:
		return m.maxTokens.Focus()
	case 6:
		return m.timeout.Focus()
	case 7:
		return m.retries.Focus()
	case 8:
		return m.testConnButton.Focus()
	}
	return nil
}

func (m *LLMSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		m.temperature.View(),
		"    ",
		m.maxTokens.View(),
	)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top,
		m.timeout.View(),
		"    ",
		m.retries.View(),
	)

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.provider.View(),
		"",
		m.model.View(),
		"",
		m.apiKey.View(),
		"",
		m.baseURL.View(),
		"",
		row1,
		"",
		row2,
		"",
		m.testConnButton.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *LLMSectionModel) Validate() []types.ValidationError {
	var errors []types.ValidationError

	if m.apiKey.Value() == "" {
		errors = append(errors, types.ValidationError{
			Field:    "API Key",
			Message:  "API Key is required",
			Severity: types.SeverityError,
		})
	}

	if m.model.Value() == "" {
		errors = append(errors, types.ValidationError{
			Field:    "Model",
			Message:  "Model name is required",
			Severity: types.SeverityError,
		})
	}

	return errors
}

func (m *LLMSectionModel) IsDirty() bool {
	return m.provider.IsDirty() || m.model.IsDirty() || m.apiKey.IsDirty() ||
		m.baseURL.IsDirty() || m.temperature.IsDirty() || m.maxTokens.IsDirty() ||
		m.timeout.IsDirty() || m.retries.IsDirty()
}

func (m *LLMSectionModel) GetValues() map[string]any {
	values := map[string]any{
		"provider": m.provider.Value(),
		"model":    m.model.Value(),
		"api_key":  m.apiKey.Value(),
		"base_url": m.baseURL.Value(),
	}

	if v := m.temperature.Value(); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			values["temperature"] = f
		}
	}
	if v := m.maxTokens.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_tokens"] = i
		}
	}
	if v := m.timeout.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["timeout"] = i
		}
	}
	if v := m.retries.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["retries"] = i
		}
	}

	return values
}

func (m *LLMSectionModel) SetValues(values map[string]any) error {
	if v, ok := values["provider"].(string); ok {
		m.provider.SetValue(v)
	}
	if v, ok := values["model"].(string); ok {
		m.model.SetValue(v)
	}
	if v, ok := values["api_key"].(string); ok {
		m.apiKey.SetValue(v)
	}
	if v, ok := values["base_url"].(string); ok {
		m.baseURL.SetValue(v)
	}
	if v, ok := values["temperature"].(float64); ok {
		m.temperature.SetValue(fmt.Sprintf("%.1f", v))
	}
	if v, ok := values["max_tokens"].(int); ok {
		m.maxTokens.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["timeout"].(int); ok {
		m.timeout.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["retries"].(int); ok {
		m.retries.SetValue(strconv.Itoa(v))
	}
	return nil
}

func (m *LLMSectionModel) FocusFirst() tea.Cmd {
	m.focusIndex = 0
	return m.provider.Focus()
}

func (m *LLMSectionModel) FocusLast() tea.Cmd {
	m.focusIndex = 8
	return m.testConnButton.Focus()
}

func (m *LLMSectionModel) testConnection() tea.Msg {
	provider := m.provider.Value()
	modelName := m.model.Value()
	apiKey := m.apiKey.Value()
	baseURL := m.baseURL.Value()

	if apiKey == "" {
		return TestConnectionResultMsg{
			Success: false,
			Message: "API Key is required to test connection",
		}
	}

	if modelName == "" {
		return TestConnectionResultMsg{
			Success: false,
			Message: "Model name is required to test connection",
		}
	}

	if provider == "" {
		provider = "openai"
	}

	cfg := config.LLMConfig{
		Provider: provider,
		Model:    modelName,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Timeout:  30,
	}

	retryClient := llm.NewRetryClient(nil)
	factory := llm.NewFactory(retryClient, nil, nil, false, 0)

	client, err := factory.CreateClient(cfg)
	if err != nil {
		return TestConnectionResultMsg{
			Success: false,
			Message: fmt.Sprintf("Failed to create client: %v", err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Say 'ok' to confirm connection works."},
		},
		MaxTokens:   10,
		Temperature: 0,
	}

	_, err = client.GenerateCompletion(ctx, req)
	if err != nil {
		return TestConnectionResultMsg{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}
	}

	return TestConnectionResultMsg{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to %s (%s)", provider, modelName),
	}
}
