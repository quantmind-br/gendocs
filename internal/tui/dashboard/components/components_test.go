package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextField_NewTextField_DefaultValues(t *testing.T) {
	tf := NewTextField("Test Label")

	if tf.label != "Test Label" {
		t.Errorf("Expected label 'Test Label', got '%s'", tf.label)
	}
	if tf.Value() != "" {
		t.Errorf("Expected empty value, got '%s'", tf.Value())
	}
	if tf.IsDirty() {
		t.Error("Expected new field to not be dirty")
	}
	if tf.required {
		t.Error("Expected default required to be false")
	}
}

func TestTextField_WithOptions(t *testing.T) {
	tf := NewTextField("Label",
		WithPlaceholder("placeholder"),
		WithRequired(),
		WithHelp("help text"),
		WithCharLimit(100),
	)

	if tf.input.Placeholder != "placeholder" {
		t.Errorf("Expected placeholder 'placeholder', got '%s'", tf.input.Placeholder)
	}
	if !tf.required {
		t.Error("Expected required to be true")
	}
	if tf.helpText != "help text" {
		t.Errorf("Expected help 'help text', got '%s'", tf.helpText)
	}
	if tf.input.CharLimit != 100 {
		t.Errorf("Expected char limit 100, got %d", tf.input.CharLimit)
	}
}

func TestTextField_SetValue_UpdatesValue(t *testing.T) {
	tf := NewTextField("Label")
	tf.SetValue("test value")

	if tf.Value() != "test value" {
		t.Errorf("Expected value 'test value', got '%s'", tf.Value())
	}
	if tf.IsDirty() {
		t.Error("SetValue should not mark as dirty")
	}
}

func TestTextField_IsDirty_TracksChanges(t *testing.T) {
	tf := NewTextField("Label")
	tf.SetValue("original")

	if tf.IsDirty() {
		t.Error("Should not be dirty after SetValue")
	}

	tf.input.SetValue("modified")
	tf.dirty = tf.input.Value() != tf.originalVal

	if !tf.IsDirty() {
		t.Error("Should be dirty after modifying value")
	}
}

func TestTextField_IsValid_RequiredField(t *testing.T) {
	tf := NewTextField("Label", WithRequired())

	if tf.IsValid() {
		t.Error("Empty required field should not be valid")
	}

	tf.SetValue("some value")
	if !tf.IsValid() {
		t.Error("Required field with value should be valid")
	}
}

func TestTextField_WithValidator_RunsValidation(t *testing.T) {
	validatorCalled := false
	tf := NewTextField("Label", WithValidator(func(s string) error {
		validatorCalled = true
		return nil
	}))

	tf.Focus()
	tf.input.SetValue("test")
	tf, _ = tf.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if !validatorCalled {
		t.Error("Validator should have been called on Update")
	}
}

func TestTextField_Focus_Blur(t *testing.T) {
	tf := NewTextField("Label")

	tf.Focus()
	if !tf.Focused() {
		t.Error("Expected field to be focused after Focus()")
	}

	tf.Blur()
	if tf.Focused() {
		t.Error("Expected field to not be focused after Blur()")
	}
}

func TestTextField_View_ContainsLabel(t *testing.T) {
	tf := NewTextField("My Label")
	view := tf.View()

	if !strings.Contains(view, "My Label") {
		t.Error("View should contain the label")
	}
}

func TestTextField_View_ShowsRequiredIndicator(t *testing.T) {
	tf := NewTextField("Label", WithRequired())
	view := tf.View()

	if !strings.Contains(view, "*") {
		t.Error("View should show required indicator")
	}
}

func TestToggle_NewToggle_DefaultValues(t *testing.T) {
	toggle := NewToggle("Test Toggle", "Help text")

	if toggle.label != "Test Toggle" {
		t.Errorf("Expected label 'Test Toggle', got '%s'", toggle.label)
	}
	if toggle.helpText != "Help text" {
		t.Errorf("Expected helpText 'Help text', got '%s'", toggle.helpText)
	}
	if toggle.Value() {
		t.Error("Expected default value to be false")
	}
	if toggle.IsDirty() {
		t.Error("Expected new toggle to not be dirty")
	}
}

func TestToggle_SetValue_UpdatesValue(t *testing.T) {
	toggle := NewToggle("Label", "")
	toggle.SetValue(true)

	if !toggle.Value() {
		t.Error("Expected value to be true after SetValue(true)")
	}
	if toggle.IsDirty() {
		t.Error("SetValue should not mark as dirty")
	}
}

func TestToggle_Update_SpaceToggles(t *testing.T) {
	toggle := NewToggle("Label", "")
	toggle.Focus()

	if toggle.Value() {
		t.Error("Initial value should be false")
	}

	toggle, _ = toggle.Update(tea.KeyMsg{Type: tea.KeySpace})

	if !toggle.Value() {
		t.Error("Space should toggle value to true")
	}
	if !toggle.IsDirty() {
		t.Error("Toggle should be dirty after change")
	}
}

func TestToggle_Update_EnterToggles(t *testing.T) {
	toggle := NewToggle("Label", "")
	toggle.Focus()
	toggle.SetValue(true)

	toggle, _ = toggle.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if toggle.Value() {
		t.Error("Enter should toggle value to false")
	}
}

func TestToggle_Update_YSetsTrue(t *testing.T) {
	toggle := NewToggle("Label", "")
	toggle.Focus()

	toggle, _ = toggle.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if !toggle.Value() {
		t.Error("Y should set value to true")
	}
}

func TestToggle_Update_NSetsFalse(t *testing.T) {
	toggle := NewToggle("Label", "")
	toggle.Focus()
	toggle.SetValue(true)

	toggle, _ = toggle.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if toggle.Value() {
		t.Error("N should set value to false")
	}
}

func TestToggle_Update_IgnoresWhenUnfocused(t *testing.T) {
	toggle := NewToggle("Label", "")

	toggle, _ = toggle.Update(tea.KeyMsg{Type: tea.KeySpace})

	if toggle.Value() {
		t.Error("Should not change value when unfocused")
	}
}

func TestToggle_Focus_Blur(t *testing.T) {
	toggle := NewToggle("Label", "")

	toggle.Focus()
	if !toggle.Focused() {
		t.Error("Expected toggle to be focused")
	}

	toggle.Blur()
	if toggle.Focused() {
		t.Error("Expected toggle to not be focused")
	}
}

func TestToggle_View_ShowsState(t *testing.T) {
	toggle := NewToggle("Label", "")

	view := toggle.View()
	if !strings.Contains(view, "Disabled") {
		t.Error("View should show Disabled when false")
	}

	toggle.SetValue(true)
	view = toggle.View()
	if !strings.Contains(view, "Enabled") {
		t.Error("View should show Enabled when true")
	}
}

func TestDropdown_NewDropdown_DefaultValues(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	}
	dd := NewDropdown("Test", opts, "Help")

	if dd.label != "Test" {
		t.Errorf("Expected label 'Test', got '%s'", dd.label)
	}
	if len(dd.options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(dd.options))
	}
	if dd.Value() != "a" {
		t.Errorf("Expected first option value 'a', got '%s'", dd.Value())
	}
	if dd.IsDirty() {
		t.Error("Expected new dropdown to not be dirty")
	}
}

func TestDropdown_SetValue_SelectsCorrectOption(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	}
	dd := NewDropdown("Test", opts, "")
	dd.SetValue("b")

	if dd.Value() != "b" {
		t.Errorf("Expected value 'b', got '%s'", dd.Value())
	}
	if dd.selected != 1 {
		t.Errorf("Expected selected index 1, got %d", dd.selected)
	}
}

func TestDropdown_Update_EnterExpandsAndSelects(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	}
	dd := NewDropdown("Test", opts, "")
	dd.Focus()

	if dd.expanded {
		t.Error("Should not be expanded initially")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !dd.expanded {
		t.Error("Enter should expand dropdown")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if dd.expanded {
		t.Error("Enter should close dropdown when expanded")
	}
}

func TestDropdown_Update_NavigatesOptions(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
		{Value: "c", Label: "Option C"},
	}
	dd := NewDropdown("Test", opts, "")
	dd.Focus()
	dd.expanded = true

	if dd.selected != 0 {
		t.Error("Should start at first option")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	if dd.selected != 1 {
		t.Error("Down should move to next option")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	if dd.selected != 2 {
		t.Error("Down should move to next option")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	if dd.selected != 2 {
		t.Error("Should not go past last option")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyUp})
	if dd.selected != 1 {
		t.Error("Up should move to previous option")
	}
}

func TestDropdown_Update_EscCloses(t *testing.T) {
	opts := []DropdownOption{{Value: "a", Label: "A"}}
	dd := NewDropdown("Test", opts, "")
	dd.Focus()
	dd.expanded = true

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if dd.expanded {
		t.Error("Esc should close dropdown")
	}
}

func TestDropdown_IsDirty_TracksChanges(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B"},
	}
	dd := NewDropdown("Test", opts, "")
	dd.SetValue("a")

	if dd.IsDirty() {
		t.Error("Should not be dirty after SetValue")
	}

	dd.Focus()
	dd.expanded = true
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !dd.IsDirty() {
		t.Error("Should be dirty after changing selection")
	}
}

func TestDropdown_Blur_ClosesExpanded(t *testing.T) {
	opts := []DropdownOption{{Value: "a", Label: "A"}}
	dd := NewDropdown("Test", opts, "")
	dd.Focus()
	dd.expanded = true

	dd.Blur()

	if dd.expanded {
		t.Error("Blur should close expanded dropdown")
	}
	if dd.Focused() {
		t.Error("Should not be focused after blur")
	}
}

func TestDropdown_View_ShowsSelectedOption(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	}
	dd := NewDropdown("Test", opts, "")
	dd.SetValue("b")

	view := dd.View()
	if !strings.Contains(view, "Option B") {
		t.Error("View should show selected option label")
	}
}

func TestMaskedInput_NewMaskedInput_DefaultValues(t *testing.T) {
	mi := NewMaskedInput("API Key", "Help")

	if mi.label != "API Key" {
		t.Errorf("Expected label 'API Key', got '%s'", mi.label)
	}
	if mi.Value() != "" {
		t.Errorf("Expected empty value, got '%s'", mi.Value())
	}
	if mi.revealed {
		t.Error("Expected masked by default")
	}
	if mi.IsDirty() {
		t.Error("Expected new input to not be dirty")
	}
}

func TestMaskedInput_SetValue_UpdatesValue(t *testing.T) {
	mi := NewMaskedInput("Label", "")
	mi.SetValue("secret-key")

	if mi.Value() != "secret-key" {
		t.Errorf("Expected value 'secret-key', got '%s'", mi.Value())
	}
	if mi.IsDirty() {
		t.Error("SetValue should not mark as dirty")
	}
}

func TestMaskedInput_Update_CtrlUTogglesReveal(t *testing.T) {
	mi := NewMaskedInput("Label", "")
	mi.Focus()

	if mi.revealed {
		t.Error("Should be masked initially")
	}

	mi, _ = mi.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	if !mi.revealed {
		t.Error("Ctrl+U should reveal the input")
	}

	mi, _ = mi.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	if mi.revealed {
		t.Error("Ctrl+U should mask the input again")
	}
}

func TestMaskedInput_IsDirty_TracksChanges(t *testing.T) {
	mi := NewMaskedInput("Label", "")
	mi.SetValue("original")

	if mi.IsDirty() {
		t.Error("Should not be dirty after SetValue")
	}

	mi.input.SetValue("modified")
	mi.dirty = mi.input.Value() != mi.originalVal

	if !mi.IsDirty() {
		t.Error("Should be dirty after change")
	}
}

func TestMaskedInput_Focus_Blur(t *testing.T) {
	mi := NewMaskedInput("Label", "")

	mi.Focus()
	if !mi.Focused() {
		t.Error("Expected input to be focused")
	}

	mi.Blur()
	if mi.Focused() {
		t.Error("Expected input to not be focused")
	}
}

func TestMaskedInput_View_ContainsLabel(t *testing.T) {
	mi := NewMaskedInput("My API Key", "")
	view := mi.View()

	if !strings.Contains(view, "My API Key") {
		t.Error("View should contain the label")
	}
}

func TestMaskedInput_View_ShowsRequiredIndicator(t *testing.T) {
	mi := NewMaskedInput("Label", "")
	view := mi.View()

	if !strings.Contains(view, "*") {
		t.Error("Masked input should always show required indicator")
	}
}

func TestMaskedInput_View_ShowsHint_WhenHasValue(t *testing.T) {
	mi := NewMaskedInput("Label", "")
	mi.SetValue("sk-1234567890abcdef")

	view := mi.View()
	if !strings.Contains(view, "cdef") {
		t.Error("View should show last 4 chars hint when unfocused with value")
	}
}
