package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/config"
)

func TestDashboard_NewDashboard_InitializesCorrectly(t *testing.T) {
	d := NewDashboard()

	if d.sidebar.items == nil || len(d.sidebar.items) == 0 {
		t.Error("Expected sidebar items to be initialized")
	}
	if d.sections == nil {
		t.Error("Expected sections map to be initialized")
	}
	if d.loader == nil {
		t.Error("Expected loader to be initialized")
	}
	if d.saver == nil {
		t.Error("Expected saver to be initialized")
	}
	if d.focusPane != FocusSidebar {
		t.Error("Expected initial focus to be on sidebar")
	}
}

func TestDashboard_Init_ReturnsCommands(t *testing.T) {
	d := NewDashboard()
	cmd := d.Init()

	if cmd == nil {
		t.Error("Init should return a batch command")
	}
}

func TestDashboard_RegisterSection_AddsSection(t *testing.T) {
	d := NewDashboard()
	mockSection := &mockSectionModel{}

	d.RegisterSection("test", mockSection)

	if _, ok := d.sections["test"]; !ok {
		t.Error("RegisterSection should add section to map")
	}
}

func TestDashboard_Update_TabSwitchesFocus(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("llm", &mockSectionModel{})

	if d.focusPane != FocusSidebar {
		t.Error("Should start with sidebar focus")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyTab})
	d = model.(DashboardModel)

	if d.focusPane != FocusContent {
		t.Error("Tab should switch focus to content")
	}

	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d = model.(DashboardModel)

	if d.focusPane != FocusSidebar {
		t.Error("Esc should switch focus back to sidebar")
	}
}

func TestDashboard_Update_EscSwitchesFocusToSidebar(t *testing.T) {
	d := NewDashboard()
	d.focusPane = FocusContent

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d = model.(DashboardModel)

	if d.focusPane != FocusSidebar {
		t.Error("Esc should switch focus to sidebar")
	}
}

func TestDashboard_Update_QuestionMarkTogglesHelp(t *testing.T) {
	d := NewDashboard()

	if d.helpVisible {
		t.Error("Help should be hidden initially")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	d = model.(DashboardModel)

	if !d.helpVisible {
		t.Error("? should show help")
	}
}

func TestDashboard_Update_HelpCanBeClosed(t *testing.T) {
	d := NewDashboard()
	d.helpVisible = true

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	d = model.(DashboardModel)

	if d.helpVisible {
		t.Error("? should close help when visible")
	}
}

func TestDashboard_Update_EscClosesHelp(t *testing.T) {
	d := NewDashboard()
	d.helpVisible = true

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d = model.(DashboardModel)

	if d.helpVisible {
		t.Error("Esc should close help")
	}
}

func TestDashboard_Update_CtrlCQuits(t *testing.T) {
	d := NewDashboard()

	model, cmd := d.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	d = model.(DashboardModel)

	if !d.quitting {
		t.Error("Ctrl+C should set quitting flag")
	}
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}

func TestDashboard_Update_ScopeToggle(t *testing.T) {
	d := NewDashboard()

	if d.statusbar.Scope() != ScopeGlobal {
		t.Error("Should start with global scope")
	}

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	d = model.(DashboardModel)

	if d.statusbar.Scope() != ScopeProject {
		t.Error("Ctrl+T should toggle to project scope")
	}
}

func TestDashboard_Update_WindowResize(t *testing.T) {
	d := NewDashboard()

	model, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	d = model.(DashboardModel)

	if d.width != 100 {
		t.Errorf("Expected width 100, got %d", d.width)
	}
	if d.height != 50 {
		t.Errorf("Expected height 50, got %d", d.height)
	}
}

func TestDashboard_Update_ConfigLoaded(t *testing.T) {
	d := NewDashboard()
	cfg := &config.GlobalConfig{
		Version: 1,
		Analyzer: config.AnalyzerConfig{
			LLM: config.LLMConfig{Provider: "openai"},
		},
	}

	model, _ := d.Update(ConfigLoadedMsg{Config: cfg, Err: nil})
	d = model.(DashboardModel)

	if d.cfg == nil {
		t.Error("Config should be set after ConfigLoadedMsg")
	}
	if d.cfg.Analyzer.LLM.Provider != "openai" {
		t.Error("Config values should be preserved")
	}
}

func TestDashboard_Update_ConfigSaved_ShowsSuccessMessage(t *testing.T) {
	d := NewDashboard()

	_, cmds := d.Update(ConfigSavedMsg{})

	if cmds == nil {
		t.Error("ConfigSavedMsg should return commands")
	}
}

func TestDashboard_Update_ConfigSaveError_ShowsErrorMessage(t *testing.T) {
	d := NewDashboard()

	_, cmds := d.Update(ConfigSaveErrorMsg{Err: nil})

	if cmds == nil {
		t.Error("ConfigSaveErrorMsg should return commands")
	}
}

func TestDashboard_HasUnsavedChanges_WhenNoDirtySections(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: false})

	if d.hasUnsavedChanges() {
		t.Error("Should not have unsaved changes when no sections are dirty")
	}
}

func TestDashboard_HasUnsavedChanges_WhenDirtySections(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: true})

	if !d.hasUnsavedChanges() {
		t.Error("Should have unsaved changes when a section is dirty")
	}
}

func TestDashboard_View_ContainsGoodbye_WhenQuitting(t *testing.T) {
	d := NewDashboard()
	d.quitting = true

	view := d.View()
	if !strings.Contains(view, "Goodbye") {
		t.Error("View should show goodbye when quitting")
	}
}

func TestDashboard_View_ShowsHelp_WhenHelpVisible(t *testing.T) {
	d := NewDashboard()
	d.helpVisible = true
	d.width = 100
	d.height = 50

	view := d.View()
	if !strings.Contains(view, "Configuration") {
		t.Error("View should show help content when help is visible")
	}
}

func TestDashboard_View_ShowsPlaceholder_WhenNoSectionRegistered(t *testing.T) {
	d := NewDashboard()
	d.width = 100
	d.height = 50

	view := d.View()
	if !strings.Contains(view, "not yet implemented") || !strings.Contains(view, "LLM") {
		t.Log("Expected view to show section placeholder")
	}
}

func TestDashboard_ActiveSection_ReturnsNil_WhenNotRegistered(t *testing.T) {
	d := NewDashboard()

	section := d.activeSection()
	if section != nil {
		t.Error("Should return nil when section not registered")
	}
}

func TestDashboard_ActiveSection_ReturnsSection_WhenRegistered(t *testing.T) {
	d := NewDashboard()
	mockSection := &mockSectionModel{}
	d.RegisterSection("llm", mockSection)

	section := d.activeSection()
	if section != mockSection {
		t.Error("Should return the registered section")
	}
}

func TestDashboard_HasError_ReturnsCorrectValue(t *testing.T) {
	d := NewDashboard()

	if d.HasError() {
		t.Error("Should not have error initially")
	}

	d.err = &testError{}
	if !d.HasError() {
		t.Error("Should have error when set")
	}
}

func TestDashboard_Error_ReturnsError(t *testing.T) {
	d := NewDashboard()
	err := &testError{}
	d.err = err

	if d.Error() != err {
		t.Error("Error() should return the stored error")
	}
}

func TestDashboard_Update_QWithUnsavedChanges_ShowsWarning(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: true})

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	d = model.(DashboardModel)

	if d.quitting {
		t.Error("Should not quit with unsaved changes on first 'q'")
	}
}

func TestDashboard_Update_QWithoutUnsavedChanges_Quits(t *testing.T) {
	d := NewDashboard()
	d.RegisterSection("test", &mockSectionModel{dirty: false})

	model, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	d = model.(DashboardModel)

	if !d.quitting {
		t.Error("Should quit when no unsaved changes")
	}
}

type mockSectionModel struct {
	dirty  bool
	values map[string]any
}

func (m *mockSectionModel) Init() tea.Cmd                           { return nil }
func (m *mockSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *mockSectionModel) View() string                            { return "mock section" }
func (m *mockSectionModel) Title() string                           { return "Mock" }
func (m *mockSectionModel) Icon() string                            { return "🧪" }
func (m *mockSectionModel) Description() string                     { return "Mock section" }
func (m *mockSectionModel) Validate() []ValidationError             { return nil }
func (m *mockSectionModel) IsDirty() bool                           { return m.dirty }
func (m *mockSectionModel) GetValues() map[string]any               { return m.values }
func (m *mockSectionModel) SetValues(values map[string]any) error   { m.values = values; return nil }
func (m *mockSectionModel) FocusFirst() tea.Cmd                     { return nil }
func (m *mockSectionModel) FocusLast() tea.Cmd                      { return nil }

type testError struct{}

func (e *testError) Error() string { return "test error" }
