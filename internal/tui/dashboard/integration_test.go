package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/tui/dashboard/sections"
	"gopkg.in/yaml.v3"
)

func TestIntegration_ConfigRoundTrip_GlobalScope(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gendocs.yaml")

	originalConfig := &config.GlobalConfig{
		Version: 1,
		Analyzer: config.AnalyzerConfig{
			LLM: config.LLMConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet",
				APIKey:      "sk-test-key-12345",
				Temperature: 0.7,
				MaxTokens:   8192,
				Timeout:     180,
				Retries:     3,
			},
			MaxWorkers:     4,
			MaxHashWorkers: 2,
		},
	}

	data, err := yaml.Marshal(originalConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	model, _ := d.Update(ConfigLoadedMsg{Config: originalConfig, Err: nil})
	d = model.(DashboardModel)

	if d.cfg == nil {
		t.Fatal("Config should be loaded")
	}

	if d.cfg.Analyzer.LLM.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %q", d.cfg.Analyzer.LLM.Provider)
	}
	if d.cfg.Analyzer.LLM.Model != "claude-3-5-sonnet" {
		t.Errorf("Expected model 'claude-3-5-sonnet', got %q", d.cfg.Analyzer.LLM.Model)
	}
}

func TestIntegration_ConfigRoundTrip_SectionValues(t *testing.T) {
	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	cfg := &config.GlobalConfig{
		Version: 1,
		Analyzer: config.AnalyzerConfig{
			LLM: config.LLMConfig{
				Provider:    "openai",
				Model:       "gpt-4o",
				APIKey:      "sk-openai-key",
				Temperature: 0.5,
				MaxTokens:   4096,
				Timeout:     120,
				Retries:     2,
			},
		},
	}

	model, _ := d.Update(ConfigLoadedMsg{Config: cfg, Err: nil})
	d = model.(DashboardModel)

	section := d.sections["llm"]
	values := section.GetValues()

	if values["provider"] != "openai" {
		t.Errorf("Expected provider 'openai', got %v", values["provider"])
	}
	if values["model"] != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got %v", values["model"])
	}
	if values["api_key"] != "sk-openai-key" {
		t.Errorf("Expected api_key 'sk-openai-key', got %v", values["api_key"])
	}
}

func TestIntegration_ScopeToggle_ChangesScope(t *testing.T) {
	d := NewDashboard()

	if d.statusbar.Scope() != ScopeGlobal {
		t.Error("Should start with global scope")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	d = model.(DashboardModel)

	if d.statusbar.Scope() != ScopeProject {
		t.Error("Ctrl+T should toggle to project scope")
	}

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	d = model.(DashboardModel)

	if d.statusbar.Scope() != ScopeGlobal {
		t.Error("Ctrl+T should toggle back to global scope")
	}
}

func TestIntegration_Navigation_SidebarToContent(t *testing.T) {
	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	if d.focusPane != FocusSidebar {
		t.Error("Should start with sidebar focus")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyTab})
	d = model.(DashboardModel)

	if d.focusPane != FocusContent {
		t.Error("Tab should move focus to content")
	}

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d = model.(DashboardModel)

	if d.focusPane != FocusSidebar {
		t.Error("Esc should move focus back to sidebar")
	}
}

func TestIntegration_Navigation_SidebarItems(t *testing.T) {
	d := NewDashboard()
	d.sidebar.SetFocused(true)
	initialItem := d.sidebar.ActiveItem()

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d = model.(DashboardModel)
	nextItem := d.sidebar.ActiveItem()

	if initialItem.ID == nextItem.ID && len(d.sidebar.items) > 1 {
		t.Error("Down arrow should navigate to next sidebar item")
	}

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyUp})
	d = model.(DashboardModel)
	backItem := d.sidebar.ActiveItem()

	if backItem.ID != initialItem.ID {
		t.Error("Up arrow should navigate back to previous item")
	}
}

func TestIntegration_UnsavedChanges_WarnsOnQuit(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: true})

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	d = model.(DashboardModel)

	if d.quitting {
		t.Error("Should not quit when there are unsaved changes")
	}
}

func TestIntegration_NoUnsavedChanges_QuitsImmediately(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: false})

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	d = model.(DashboardModel)

	if !d.quitting {
		t.Error("Should quit when there are no unsaved changes")
	}
}

func TestIntegration_CtrlC_AlwaysQuits(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: true})

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	d = model.(DashboardModel)

	if !d.quitting {
		t.Error("Ctrl+C should always quit even with unsaved changes")
	}
}

func TestIntegration_HelpToggle(t *testing.T) {
	d := NewDashboard()

	if d.helpVisible {
		t.Error("Help should be hidden initially")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	d = model.(DashboardModel)

	if !d.helpVisible {
		t.Error("? should show help")
	}

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d = model.(DashboardModel)

	if d.helpVisible {
		t.Error("Esc should close help")
	}
}

func TestIntegration_WindowResize_UpdatesDimensions(t *testing.T) {
	d := NewDashboard()

	model, _ := d.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	d = model.(DashboardModel)

	if d.width != 120 {
		t.Errorf("Expected width 120, got %d", d.width)
	}
	if d.height != 40 {
		t.Errorf("Expected height 40, got %d", d.height)
	}
}

func TestIntegration_ConfigLoadedMsg_PopulatesSections(t *testing.T) {
	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	cfg := &config.GlobalConfig{
		Version: 1,
		Analyzer: config.AnalyzerConfig{
			LLM: config.LLMConfig{
				Provider:    "gemini",
				Model:       "gemini-pro",
				APIKey:      "gemini-key",
				Temperature: 1.0,
				MaxTokens:   16384,
				Timeout:     300,
				Retries:     5,
			},
		},
	}

	model, _ := d.Update(ConfigLoadedMsg{Config: cfg, Err: nil})
	d = model.(DashboardModel)

	section := d.sections["llm"]
	values := section.GetValues()

	if values["provider"] != "gemini" {
		t.Errorf("Expected provider 'gemini', got %v", values["provider"])
	}
	if values["model"] != "gemini-pro" {
		t.Errorf("Expected model 'gemini-pro', got %v", values["model"])
	}
	if temp, ok := values["temperature"].(float64); !ok || temp != 1.0 {
		t.Errorf("Expected temperature 1.0, got %v", values["temperature"])
	}
}

func TestIntegration_ApplyValuesToConfig_UpdatesConfig(t *testing.T) {
	d := NewDashboard()
	d.cfg = &config.GlobalConfig{
		Version:  1,
		Analyzer: config.AnalyzerConfig{},
	}

	values := map[string]any{
		"provider":    "anthropic",
		"model":       "claude-3-opus",
		"api_key":     "sk-new-key",
		"temperature": 0.8,
		"max_tokens":  16000,
	}

	d.applyValuesToConfig(values)

	if d.cfg.Analyzer.LLM.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %q", d.cfg.Analyzer.LLM.Provider)
	}
	if d.cfg.Analyzer.LLM.Model != "claude-3-opus" {
		t.Errorf("Expected model 'claude-3-opus', got %q", d.cfg.Analyzer.LLM.Model)
	}
	if d.cfg.Analyzer.LLM.APIKey != "sk-new-key" {
		t.Errorf("Expected api_key 'sk-new-key', got %q", d.cfg.Analyzer.LLM.APIKey)
	}
	if d.cfg.Analyzer.LLM.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got %f", d.cfg.Analyzer.LLM.Temperature)
	}
	if d.cfg.Analyzer.LLM.MaxTokens != 16000 {
		t.Errorf("Expected max_tokens 16000, got %d", d.cfg.Analyzer.LLM.MaxTokens)
	}
}

func TestIntegration_MultipleSections_AllRegistered(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("llm", sections.NewLLMSection())

	if _, ok := d.sections["llm"]; !ok {
		t.Error("LLM section should be registered")
	}

	if len(d.sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(d.sections))
	}
}

func TestIntegration_ContentFocus_UpdatesSection(t *testing.T) {
	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	cfg := &config.GlobalConfig{
		Version:  1,
		Analyzer: config.AnalyzerConfig{},
	}
	model, _ := d.Update(ConfigLoadedMsg{Config: cfg, Err: nil})
	d = model.(DashboardModel)

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})
	d = model.(DashboardModel)

	if d.focusPane != FocusContent {
		t.Error("Should be focused on content after Tab")
	}
}

func TestIntegration_ActiveSection_ReturnsCorrectSection(t *testing.T) {
	d := NewDashboard()
	llmSection := sections.NewLLMSection()
	d.RegisterSection("llm", llmSection)

	section := d.activeSection()
	if section == nil {
		t.Error("activeSection should return the LLM section when it matches sidebar selection")
	}
}

func TestIntegration_Sidebar_DefaultNavItems(t *testing.T) {
	d := NewDashboard()

	if len(d.sidebar.items) == 0 {
		t.Error("Sidebar should have default nav items")
	}

	foundLLM := false
	for _, item := range d.sidebar.items {
		if item.ID == "llm" {
			foundLLM = true
			break
		}
	}

	if !foundLLM {
		t.Error("Sidebar should have LLM nav item")
	}
}

func TestIntegration_ConfigSavedMsg_TriggersSuccessMessage(t *testing.T) {
	d := NewDashboard()

	_, cmds := d.Update(ConfigSavedMsg{})

	if cmds == nil {
		t.Error("ConfigSavedMsg should return commands for success message")
	}
}

func TestIntegration_ConfigSaveErrorMsg_TriggersErrorMessage(t *testing.T) {
	d := NewDashboard()

	_, cmds := d.Update(ConfigSaveErrorMsg{Err: nil})

	if cmds == nil {
		t.Error("ConfigSaveErrorMsg should return commands for error message")
	}
}

func TestIntegration_View_ContainsSidebar(t *testing.T) {
	d := NewDashboard()
	d.width = 100
	d.height = 50

	view := d.View()

	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestIntegration_View_QuittingShowsGoodbye(t *testing.T) {
	d := NewDashboard()
	d.quitting = true

	view := d.View()

	if view != "Goodbye!\n" {
		t.Errorf("Expected 'Goodbye!', got %q", view)
	}
}

func TestIntegration_Init_ReturnsBatchCommand(t *testing.T) {
	d := NewDashboard()
	cmd := d.Init()

	if cmd == nil {
		t.Error("Init should return a batch command")
	}
}
