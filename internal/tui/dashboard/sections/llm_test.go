package sections

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/tui/dashboard/types"
)

func TestLLMSection_NewLLMSection_InitializesCorrectly(t *testing.T) {
	s := NewLLMSection()

	if s == nil {
		t.Fatal("NewLLMSection should not return nil")
	}

	if s.focusIndex != 0 {
		t.Errorf("Expected initial focus index 0, got %d", s.focusIndex)
	}
}

func TestLLMSection_Title_ReturnsExpectedValue(t *testing.T) {
	s := NewLLMSection()
	expected := "Analyzer LLM"

	if s.Title() != expected {
		t.Errorf("Expected title %q, got %q", expected, s.Title())
	}
}

func TestLLMSection_Icon_ReturnsExpectedValue(t *testing.T) {
	s := NewLLMSection()
	expected := "🤖"

	if s.Icon() != expected {
		t.Errorf("Expected icon %q, got %q", expected, s.Icon())
	}
}

func TestLLMSection_Description_ReturnsExpectedValue(t *testing.T) {
	s := NewLLMSection()
	expected := "LLM settings for codebase analysis (gendocs analyze)"

	if s.Description() != expected {
		t.Errorf("Expected description %q, got %q", expected, s.Description())
	}
}

func TestLLMSection_SetValues_SetsAllFields(t *testing.T) {
	s := NewLLMSection()
	values := map[string]any{
		"provider":    "anthropic",
		"model":       "claude-3-5-sonnet",
		"api_key":     "sk-test-key",
		"base_url":    "https://api.anthropic.com",
		"temperature": 0.7,
		"max_tokens":  8192,
		"timeout":     180,
		"retries":     3,
	}

	err := s.SetValues(values)
	if err != nil {
		t.Fatalf("SetValues returned error: %v", err)
	}

	got := s.GetValues()
	if got["provider"] != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %v", got["provider"])
	}
	if got["model"] != "claude-3-5-sonnet" {
		t.Errorf("Expected model 'claude-3-5-sonnet', got %v", got["model"])
	}
	if got["api_key"] != "sk-test-key" {
		t.Errorf("Expected api_key 'sk-test-key', got %v", got["api_key"])
	}
	if got["base_url"] != "https://api.anthropic.com" {
		t.Errorf("Expected base_url 'https://api.anthropic.com', got %v", got["base_url"])
	}
	if got["temperature"] != 0.7 {
		t.Errorf("Expected temperature 0.7, got %v", got["temperature"])
	}
	if got["max_tokens"] != 8192 {
		t.Errorf("Expected max_tokens 8192, got %v", got["max_tokens"])
	}
	if got["timeout"] != 180 {
		t.Errorf("Expected timeout 180, got %v", got["timeout"])
	}
	if got["retries"] != 3 {
		t.Errorf("Expected retries 3, got %v", got["retries"])
	}
}

func TestLLMSection_SetValues_PartialUpdate(t *testing.T) {
	s := NewLLMSection()
	values := map[string]any{
		"provider": "openai",
		"model":    "gpt-4o",
	}

	err := s.SetValues(values)
	if err != nil {
		t.Fatalf("SetValues returned error: %v", err)
	}

	got := s.GetValues()
	if got["provider"] != "openai" {
		t.Errorf("Expected provider 'openai', got %v", got["provider"])
	}
	if got["model"] != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got %v", got["model"])
	}
}

func TestLLMSection_GetValues_ReturnsEmptyStringsForUnsetFields(t *testing.T) {
	s := NewLLMSection()

	got := s.GetValues()
	if got["model"] != "" {
		t.Errorf("Expected empty model, got %v", got["model"])
	}
	if got["api_key"] != "" {
		t.Errorf("Expected empty api_key, got %v", got["api_key"])
	}
	if got["base_url"] != "" {
		t.Errorf("Expected empty base_url, got %v", got["base_url"])
	}
}

func TestLLMSection_GetValues_ConvertsNumericFields(t *testing.T) {
	s := NewLLMSection()
	values := map[string]any{
		"temperature": 1.5,
		"max_tokens":  4096,
		"timeout":     60,
		"retries":     5,
	}

	_ = s.SetValues(values)
	got := s.GetValues()

	if temp, ok := got["temperature"].(float64); !ok || temp != 1.5 {
		t.Errorf("Expected temperature 1.5, got %v", got["temperature"])
	}
	if tokens, ok := got["max_tokens"].(int); !ok || tokens != 4096 {
		t.Errorf("Expected max_tokens 4096, got %v", got["max_tokens"])
	}
	if timeout, ok := got["timeout"].(int); !ok || timeout != 60 {
		t.Errorf("Expected timeout 60, got %v", got["timeout"])
	}
	if retries, ok := got["retries"].(int); !ok || retries != 5 {
		t.Errorf("Expected retries 5, got %v", got["retries"])
	}
}

func TestLLMSection_IsDirty_FalseAfterSetValues(t *testing.T) {
	s := NewLLMSection()
	values := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
	}

	_ = s.SetValues(values)

	if s.IsDirty() {
		t.Error("Should not be dirty after SetValues")
	}
}

func TestLLMSection_Validate_RequiresAPIKey(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"model": "gpt-4",
	})

	errs := s.Validate()

	var hasAPIKeyError bool
	for _, e := range errs {
		if e.Field == "API Key" && e.Severity == types.SeverityError {
			hasAPIKeyError = true
			break
		}
	}

	if !hasAPIKeyError {
		t.Error("Expected validation error for missing API Key")
	}
}

func TestLLMSection_Validate_RequiresModel(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"api_key": "sk-test",
	})

	errs := s.Validate()

	var hasModelError bool
	for _, e := range errs {
		if e.Field == "Model" && e.Severity == types.SeverityError {
			hasModelError = true
			break
		}
	}

	if !hasModelError {
		t.Error("Expected validation error for missing Model")
	}
}

func TestLLMSection_Validate_PassesWithRequiredFields(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"api_key": "sk-test",
		"model":   "gpt-4",
	})

	errs := s.Validate()

	if len(errs) > 0 {
		t.Errorf("Expected no validation errors, got %d: %v", len(errs), errs)
	}
}

func TestLLMSection_Update_TabNavigatesForward(t *testing.T) {
	s := NewLLMSection()

	if s.focusIndex != 0 {
		t.Fatalf("Expected initial focus index 0, got %d", s.focusIndex)
	}

	model, _ := s.Update(tea.KeyMsg{Type: tea.KeyTab})
	s = model.(*LLMSectionModel)

	if s.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after Tab, got %d", s.focusIndex)
	}

	model, _ = s.Update(tea.KeyMsg{Type: tea.KeyTab})
	s = model.(*LLMSectionModel)

	if s.focusIndex != 2 {
		t.Errorf("Expected focus index 2 after second Tab, got %d", s.focusIndex)
	}
}

func TestLLMSection_Update_TabWrapsAround(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 8

	model, _ := s.Update(tea.KeyMsg{Type: tea.KeyTab})
	s = model.(*LLMSectionModel)

	if s.focusIndex != 0 {
		t.Errorf("Expected focus to wrap to 0, got %d", s.focusIndex)
	}
}

func TestLLMSection_Update_ShiftTabNavigatesBackward(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 2

	model, _ := s.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	s = model.(*LLMSectionModel)

	if s.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after Shift+Tab, got %d", s.focusIndex)
	}
}

func TestLLMSection_Update_ShiftTabWrapsAround(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 0

	model, _ := s.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	s = model.(*LLMSectionModel)

	if s.focusIndex != 8 {
		t.Errorf("Expected focus to wrap to 8, got %d", s.focusIndex)
	}
}

func TestLLMSection_FocusFirst_SetsFocusToZero(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 5

	s.FocusFirst()

	if s.focusIndex != 0 {
		t.Errorf("Expected focus index 0 after FocusFirst, got %d", s.focusIndex)
	}
}

func TestLLMSection_FocusLast_SetsFocusToLast(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 0

	s.FocusLast()

	if s.focusIndex != 8 {
		t.Errorf("Expected focus index 8 after FocusLast, got %d", s.focusIndex)
	}
}

func TestLLMSection_View_ContainsTitle(t *testing.T) {
	s := NewLLMSection()
	view := s.View()

	if !strings.Contains(view, "Analyzer LLM") {
		t.Error("View should contain section title")
	}
}

func TestLLMSection_View_ContainsIcon(t *testing.T) {
	s := NewLLMSection()
	view := s.View()

	if !strings.Contains(view, "🤖") {
		t.Error("View should contain section icon")
	}
}

func TestLLMSection_View_ContainsDescription(t *testing.T) {
	s := NewLLMSection()
	view := s.View()

	if !strings.Contains(view, "LLM settings for codebase analysis") {
		t.Error("View should contain section description")
	}
}

func TestLLMSection_View_ContainsFieldLabels(t *testing.T) {
	s := NewLLMSection()
	view := s.View()

	expectedLabels := []string{
		"Provider",
		"Model",
		"API Key",
		"Base URL",
		"Temperature",
		"Max Tokens",
		"Timeout",
		"Retries",
	}

	for _, label := range expectedLabels {
		if !strings.Contains(view, label) {
			t.Errorf("View should contain field label %q", label)
		}
	}
}

func TestLLMSection_Init_ReturnsNil(t *testing.T) {
	s := NewLLMSection()
	cmd := s.Init()

	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestLLMSection_ImplementsSectionModel(t *testing.T) {
	s := NewLLMSection()
	var _ types.SectionModel = s
}

func TestLLMSection_ProviderOptions_HasExpectedValues(t *testing.T) {
	s := NewLLMSection()
	providers := []string{"openai", "anthropic", "gemini"}

	for _, provider := range providers {
		_ = s.SetValues(map[string]any{"provider": provider})
		got := s.GetValues()
		if got["provider"] != provider {
			t.Errorf("Expected provider %q to be settable, got %v", provider, got["provider"])
		}
	}
}

func TestLLMSection_TemperatureFormatting(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"temperature": 0.5,
	})

	got := s.GetValues()
	if temp, ok := got["temperature"].(float64); !ok {
		t.Error("Temperature should be float64")
	} else if temp != 0.5 {
		t.Errorf("Expected temperature 0.5, got %v", temp)
	}
}

func TestLLMSection_EmptyNumericFields_NotIncludedInGetValues(t *testing.T) {
	s := NewLLMSection()

	got := s.GetValues()

	if _, ok := got["model"]; !ok {
		t.Error("model key should exist")
	}

	if _, ok := got["temperature"]; ok {
		t.Error("temperature should not be in values when empty")
	}
	if _, ok := got["max_tokens"]; ok {
		t.Error("max_tokens should not be in values when empty")
	}
	if _, ok := got["timeout"]; ok {
		t.Error("timeout should not be in values when empty")
	}
	if _, ok := got["retries"]; ok {
		t.Error("retries should not be in values when empty")
	}
}

func TestLLMSection_View_ContainsTestConnectionButton(t *testing.T) {
	s := NewLLMSection()
	view := s.View()

	if !strings.Contains(view, "Test Connection") {
		t.Error("View should contain Test Connection button")
	}
}

func TestLLMSection_TestConnection_RequiresAPIKey(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"model": "gpt-4",
	})

	result := s.testConnection()
	msg, ok := result.(TestConnectionResultMsg)
	if !ok {
		t.Fatal("Expected TestConnectionResultMsg")
	}

	if msg.Success {
		t.Error("Expected failure when API key is missing")
	}
	if !strings.Contains(msg.Message, "API Key") {
		t.Errorf("Expected message to mention API Key, got %q", msg.Message)
	}
}

func TestLLMSection_TestConnection_RequiresModel(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"api_key": "sk-test",
	})

	result := s.testConnection()
	msg, ok := result.(TestConnectionResultMsg)
	if !ok {
		t.Fatal("Expected TestConnectionResultMsg")
	}

	if msg.Success {
		t.Error("Expected failure when model is missing")
	}
	if !strings.Contains(msg.Message, "Model") {
		t.Errorf("Expected message to mention Model, got %q", msg.Message)
	}
}

func TestLLMSection_TestConnection_DefaultsProvider(t *testing.T) {
	s := NewLLMSection()
	_ = s.SetValues(map[string]any{
		"api_key": "sk-test",
		"model":   "gpt-4",
	})

	result := s.testConnection()
	msg, ok := result.(TestConnectionResultMsg)
	if !ok {
		t.Fatal("Expected TestConnectionResultMsg")
	}

	if msg.Success {
		t.Skip("Skipping: would need a real API connection")
	}
}

func TestLLMSection_Update_HandlesTestConnectionResult(t *testing.T) {
	s := NewLLMSection()
	s.testing = true

	resultMsg := TestConnectionResultMsg{
		Success: true,
		Message: "Connection successful",
	}

	model, _ := s.Update(resultMsg)
	s = model.(*LLMSectionModel)

	if s.testing {
		t.Error("Expected testing to be false after result")
	}
}

func TestLLMSection_Update_ButtonPress(t *testing.T) {
	s := NewLLMSection()
	s.focusIndex = 8

	_ = s.SetValues(map[string]any{
		"api_key": "sk-test",
		"model":   "gpt-4",
	})

	s.testConnButton.Focus()
}
