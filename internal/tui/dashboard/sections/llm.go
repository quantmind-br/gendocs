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

type LLMTarget int

const (
	LLMTargetAnalyzer LLMTarget = iota
	LLMTargetDocumenter
	LLMTargetAIRules
)

// Default BaseURLs for local LLM providers
const (
	OllamaDefaultBaseURL   = "http://localhost:11434/v1"
	LMStudioDefaultBaseURL = "http://localhost:1234/v1"
)

// isLocalProvider returns true if the provider is a local LLM (Ollama, LM Studio)
func isLocalProvider(provider string) bool {
	return provider == "ollama" || provider == "lmstudio"
}

// getDefaultBaseURL returns the default BaseURL for a provider
func getDefaultBaseURL(provider string) string {
	switch provider {
	case "ollama":
		return OllamaDefaultBaseURL
	case "lmstudio":
		return LMStudioDefaultBaseURL
	default:
		return ""
	}
}

// getModelPlaceholder returns an appropriate placeholder for the model field
func getModelPlaceholder(provider string) string {
	switch provider {
	case "openai":
		return "e.g., gpt-4o, gpt-4-turbo, gpt-3.5-turbo"
	case "anthropic":
		return "e.g., claude-3-5-sonnet-20241022, claude-3-opus"
	case "gemini":
		return "e.g., gemini-1.5-pro, gemini-1.5-flash"
	case "ollama":
		return "e.g., llama3, codellama, mistral, deepseek-coder"
	case "lmstudio":
		return "e.g., llama3, codellama, mistral (check LM Studio)"
	default:
		return "e.g., gpt-4o, claude-3-5-sonnet"
	}
}

type LLMSectionDescriptor struct {
	ID          string
	Title       string
	Icon        string
	Description string
	KeyPrefix   string
}

var LLMTargetDescriptors = map[LLMTarget]LLMSectionDescriptor{
	LLMTargetAnalyzer: {
		ID:          "llm",
		Title:       "Analyzer LLM",
		Icon:        "🤖",
		Description: "LLM settings for codebase analysis (gendocs analyze)",
		KeyPrefix:   "",
	},
	LLMTargetDocumenter: {
		ID:          "documenter_llm",
		Title:       "Documenter LLM",
		Icon:        "📝",
		Description: "LLM settings for README generation (gendocs generate readme)",
		KeyPrefix:   "documenter_",
	},
	LLMTargetAIRules: {
		ID:          "ai_rules_llm",
		Title:       "AI Rules LLM",
		Icon:        "📋",
		Description: "LLM settings for AI rules generation (gendocs generate ai-rules)",
		KeyPrefix:   "ai_rules_",
	},
}

type LLMSectionModel struct {
	target     LLMTarget
	descriptor LLMSectionDescriptor

	provider       components.DropdownModel
	model          components.TextFieldModel
	apiKey         components.MaskedInputModel
	baseURL        components.TextFieldModel
	temperature    components.TextFieldModel
	maxTokens      components.TextFieldModel
	timeout        components.TextFieldModel
	retries        components.TextFieldModel
	testConnButton components.ButtonModel

	inputs       *components.FocusableSlice
	testing      bool
	prevProvider string
}

type TestConnectionResultMsg struct {
	Success bool
	Message string
}

func NewLLMSection() *LLMSectionModel {
	return NewLLMSectionWithTarget(LLMTargetAnalyzer)
}

func NewLLMSectionWithTarget(target LLMTarget) *LLMSectionModel {
	descriptor := LLMTargetDescriptors[target]

	providerOpts := []components.DropdownOption{
		{Value: "openai", Label: "OpenAI (GPT-4o, GPT-4)"},
		{Value: "anthropic", Label: "Anthropic (Claude)"},
		{Value: "gemini", Label: "Google (Gemini)"},
		{Value: "ollama", Label: "Ollama (Local)"},
		{Value: "lmstudio", Label: "LM Studio (Local)"},
	}

	defaultProvider := "openai"

	m := &LLMSectionModel{
		target:       target,
		descriptor:   descriptor,
		prevProvider: defaultProvider,
		provider:     components.NewDropdown("Provider", providerOpts, "Select your LLM provider"),
		model: components.NewTextField("Model",
			components.WithPlaceholder(getModelPlaceholder(defaultProvider)),
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

	m.inputs = components.NewFocusableSlice(
		components.WrapDropdown(&m.provider),
		components.WrapTextField(&m.model),
		components.WrapMaskedInput(&m.apiKey),
		components.WrapTextField(&m.baseURL),
		components.WrapTextField(&m.temperature),
		components.WrapTextField(&m.maxTokens),
		components.WrapTextField(&m.timeout),
		components.WrapTextField(&m.retries),
		components.WrapButton(&m.testConnButton),
	)

	return m
}

func (m *LLMSectionModel) Title() string       { return m.descriptor.Title }
func (m *LLMSectionModel) Icon() string        { return m.descriptor.Icon }
func (m *LLMSectionModel) Description() string { return m.descriptor.Description }
func (m *LLMSectionModel) ID() string          { return m.descriptor.ID }
func (m *LLMSectionModel) Target() LLMTarget   { return m.target }

func (m *LLMSectionModel) Init() tea.Cmd {
	return nil
}

func (m *LLMSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TestConnectionResultMsg:
		m.testing = false
		m.testConnButton.SetLoading(false)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return m, m.inputs.FocusNext()
		case "shift+tab":
			return m, m.inputs.FocusPrev()
		}
	}

	cmd := m.inputs.UpdateCurrent(msg)

	currentProvider := m.provider.Value()
	if currentProvider != m.prevProvider {
		m.onProviderChange(currentProvider, m.prevProvider)
		m.prevProvider = currentProvider
	}

	if cmd != nil && m.inputs.Index() == m.inputs.Len()-1 {
		m.testing = true
	}

	return m, cmd
}

func (m *LLMSectionModel) onProviderChange(newProvider, oldProvider string) {
	if isLocalProvider(newProvider) {
		m.baseURL.SetValue(getDefaultBaseURL(newProvider))
	} else if isLocalProvider(oldProvider) {
		oldDefault := getDefaultBaseURL(oldProvider)
		if m.baseURL.Value() == oldDefault {
			m.baseURL.SetValue("")
		}
	}

	m.apiKey.SetRequired(!isLocalProvider(newProvider))
	m.model.SetPlaceholder(getModelPlaceholder(newProvider))
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

	if !isLocalProvider(m.provider.Value()) && m.apiKey.Value() == "" {
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

	if isLocalProvider(m.provider.Value()) && m.baseURL.Value() == "" {
		errors = append(errors, types.ValidationError{
			Field:    "Base URL",
			Message:  "Base URL is required for local providers",
			Severity: types.SeverityError,
		})
	}

	return errors
}

func (m *LLMSectionModel) IsDirty() bool {
	return m.inputs.IsDirty()
}

func (m *LLMSectionModel) GetValues() map[string]any {
	p := m.descriptor.KeyPrefix
	values := map[string]any{
		p + KeyProvider: m.provider.Value(),
		p + KeyModel:    m.model.Value(),
		p + KeyAPIKey:   m.apiKey.Value(),
		p + KeyBaseURL:  m.baseURL.Value(),
	}

	if v := m.temperature.Value(); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			values[p+KeyTemperature] = f
		}
	}
	if v := m.maxTokens.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[p+KeyMaxTokens] = i
		}
	}
	if v := m.timeout.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[p+KeyTimeout] = i
		}
	}
	if v := m.retries.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[p+KeyRetries] = i
		}
	}

	return values
}

func (m *LLMSectionModel) SetValues(values map[string]any) error {
	p := m.descriptor.KeyPrefix
	if v, ok := values[p+KeyProvider].(string); ok {
		m.provider.SetValue(v)
	}
	if v, ok := values[p+KeyModel].(string); ok {
		m.model.SetValue(v)
	}
	if v, ok := values[p+KeyAPIKey].(string); ok {
		m.apiKey.SetValue(v)
	}
	if v, ok := values[p+KeyBaseURL].(string); ok {
		m.baseURL.SetValue(v)
	}
	if v, ok := values[p+KeyTemperature].(float64); ok {
		m.temperature.SetValue(fmt.Sprintf("%.1f", v))
	}
	if v, ok := values[p+KeyMaxTokens].(int); ok {
		m.maxTokens.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[p+KeyTimeout].(int); ok {
		m.timeout.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[p+KeyRetries].(int); ok {
		m.retries.SetValue(strconv.Itoa(v))
	}
	return nil
}

func (m *LLMSectionModel) FocusFirst() tea.Cmd {
	return m.inputs.FocusFirst()
}

func (m *LLMSectionModel) FocusLast() tea.Cmd {
	return m.inputs.FocusLast()
}

func (m *LLMSectionModel) testConnection() tea.Msg {
	provider := m.provider.Value()
	modelName := m.model.Value()
	apiKey := m.apiKey.Value()
	baseURL := m.baseURL.Value()

	if !isLocalProvider(provider) && apiKey == "" {
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

	if isLocalProvider(provider) && baseURL == "" {
		return TestConnectionResultMsg{
			Success: false,
			Message: "Base URL is required for local providers",
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
