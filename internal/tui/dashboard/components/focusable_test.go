package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFocusableSlice_Navigation(t *testing.T) {
	tf1 := NewTextField("Field1")
	tf2 := NewTextField("Field2")
	tf3 := NewTextField("Field3")

	fs := NewFocusableSlice(
		WrapTextField(&tf1),
		WrapTextField(&tf2),
		WrapTextField(&tf3),
	)

	if fs.Len() != 3 {
		t.Errorf("Len() = %d, want 3", fs.Len())
	}

	if fs.Index() != 0 {
		t.Errorf("initial Index() = %d, want 0", fs.Index())
	}

	fs.FocusFirst()
	if !tf1.Focused() {
		t.Error("FocusFirst(): tf1 should be focused")
	}

	fs.FocusNext()
	if tf1.Focused() {
		t.Error("FocusNext(): tf1 should be blurred")
	}
	if !tf2.Focused() {
		t.Error("FocusNext(): tf2 should be focused")
	}
	if fs.Index() != 1 {
		t.Errorf("Index() = %d, want 1", fs.Index())
	}

	fs.FocusNext()
	fs.FocusNext()
	if fs.Index() != 0 {
		t.Errorf("Index() after wrap = %d, want 0", fs.Index())
	}
	if !tf1.Focused() {
		t.Error("FocusNext() wrap: tf1 should be focused")
	}

	fs.FocusPrev()
	if fs.Index() != 2 {
		t.Errorf("FocusPrev() wrap Index() = %d, want 2", fs.Index())
	}
	if !tf3.Focused() {
		t.Error("FocusPrev() wrap: tf3 should be focused")
	}

	fs.FocusLast()
	if fs.Index() != 2 {
		t.Errorf("FocusLast() Index() = %d, want 2", fs.Index())
	}
}

func TestFocusableSlice_BlurAll(t *testing.T) {
	tf1 := NewTextField("Field1")
	tf2 := NewTextField("Field2")

	fs := NewFocusableSlice(
		WrapTextField(&tf1),
		WrapTextField(&tf2),
	)

	fs.FocusFirst()
	fs.FocusNext()
	fs.BlurAll()

	if tf1.Focused() || tf2.Focused() {
		t.Error("BlurAll(): all fields should be blurred")
	}
}

func TestFocusableSlice_UpdateCurrent(t *testing.T) {
	tf := NewTextField("Test")
	fs := NewFocusableSlice(WrapTextField(&tf))

	fs.FocusFirst()
	fs.UpdateCurrent(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if tf.Value() != "a" {
		t.Errorf("Value() = %q, want 'a'", tf.Value())
	}
}

func TestFocusableSlice_IsDirty(t *testing.T) {
	tf := NewTextField("Test")
	fs := NewFocusableSlice(WrapTextField(&tf))

	if fs.IsDirty() {
		t.Error("IsDirty() should be false initially")
	}

	tf.SetValue("original")
	fs.FocusFirst()
	fs.UpdateCurrent(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if !fs.IsDirty() {
		t.Error("IsDirty() should be true after modification")
	}
}

func TestFocusableSlice_Empty(t *testing.T) {
	fs := NewFocusableSlice()

	if fs.Len() != 0 {
		t.Errorf("Len() = %d, want 0", fs.Len())
	}
	if fs.Current() != nil {
		t.Error("Current() should be nil for empty slice")
	}
	if fs.FocusNext() != nil {
		t.Error("FocusNext() should return nil for empty slice")
	}
	if fs.FocusPrev() != nil {
		t.Error("FocusPrev() should return nil for empty slice")
	}
}

func TestFocusableSlice_Get(t *testing.T) {
	tf := NewTextField("Test")
	fs := NewFocusableSlice(WrapTextField(&tf))

	if fs.Get(0) == nil {
		t.Error("Get(0) should not be nil")
	}
	if fs.Get(1) != nil {
		t.Error("Get(1) should be nil for out of bounds")
	}
	if fs.Get(-1) != nil {
		t.Error("Get(-1) should be nil for negative index")
	}
}

func TestWrappers_ImplementFocusableUpdater(t *testing.T) {
	var _ FocusableUpdater = WrapTextField(&TextFieldModel{})
	var _ FocusableUpdater = WrapDropdown(&DropdownModel{})
	var _ FocusableUpdater = WrapMaskedInput(&MaskedInputModel{})
	var _ FocusableUpdater = WrapButton(&ButtonModel{})
	var _ FocusableUpdater = WrapToggle(&ToggleModel{})
}

func TestDropdownWrapper_Update(t *testing.T) {
	opts := []DropdownOption{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	}
	dd := NewDropdown("Test", opts, "help")
	w := WrapDropdown(&dd)

	w.Focus()
	w.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !dd.expanded {
		t.Error("Dropdown should be expanded after Enter")
	}

	w.Update(tea.KeyMsg{Type: tea.KeyDown})
	w.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if dd.Value() != "b" {
		t.Errorf("Value() = %q, want 'b'", dd.Value())
	}
}

func TestButtonWrapper_Update(t *testing.T) {
	pressed := false
	btn := NewButton("Test", func() tea.Msg {
		pressed = true
		return nil
	})
	w := WrapButton(&btn)

	w.Focus()
	cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("Button should return command on Enter")
	}

	cmd()
	if !pressed {
		t.Error("Button onPress should have been called")
	}
}

func TestToggleWrapper_Update(t *testing.T) {
	tests := []struct {
		name         string
		initialValue bool
		keyType      tea.KeyType
		keyRunes     []rune
		wantValue    bool
	}{
		{
			name:         "space toggles false to true",
			initialValue: false,
			keyType:      tea.KeySpace,
			wantValue:    true,
		},
		{
			name:         "space toggles true to false",
			initialValue: true,
			keyType:      tea.KeySpace,
			wantValue:    false,
		},
		{
			name:         "enter toggles false to true",
			initialValue: false,
			keyType:      tea.KeyEnter,
			wantValue:    true,
		},
		{
			name:         "enter toggles true to false",
			initialValue: true,
			keyType:      tea.KeyEnter,
			wantValue:    false,
		},
		{
			name:         "tab key does not toggle",
			initialValue: false,
			keyType:      tea.KeyTab,
			wantValue:    false,
		},
		{
			name:         "letter key does not toggle",
			initialValue: true,
			keyType:      tea.KeyRunes,
			keyRunes:     []rune{'x'},
			wantValue:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toggle := NewToggle("Test", "help text")
			toggle.SetValue(tt.initialValue)
			wrapper := WrapToggle(&toggle)

			wrapper.Focus()
			msg := tea.KeyMsg{Type: tt.keyType, Runes: tt.keyRunes}
			wrapper.Update(msg)

			if toggle.Value() != tt.wantValue {
				t.Errorf("Value() = %v, want %v", toggle.Value(), tt.wantValue)
			}
		})
	}
}

func TestToggleWrapper_FocusBlur(t *testing.T) {
	toggle := NewToggle("Test", "help")
	wrapper := WrapToggle(&toggle)

	if toggle.Focused() {
		t.Error("Toggle should not be focused initially")
	}

	wrapper.Focus()
	if !toggle.Focused() {
		t.Error("Toggle should be focused after Focus()")
	}

	wrapper.Blur()
	if toggle.Focused() {
		t.Error("Toggle should not be focused after Blur()")
	}
}

func TestToggleWrapper_IsDirty(t *testing.T) {
	toggle := NewToggle("Test", "help")
	toggle.SetValue(false)
	wrapper := WrapToggle(&toggle)

	if wrapper.IsDirty() {
		t.Error("IsDirty() should be false initially")
	}

	wrapper.Focus()
	wrapper.Update(tea.KeyMsg{Type: tea.KeySpace})

	if !wrapper.IsDirty() {
		t.Error("IsDirty() should be true after toggle")
	}
}

func TestMaskedInputWrapper_Update(t *testing.T) {
	mi := NewMaskedInput("API Key", "Enter your API key")
	wrapper := WrapMaskedInput(&mi)

	wrapper.Focus()

	wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})

	if mi.Value() != "sk-" {
		t.Errorf("Value() = %q, want %q", mi.Value(), "sk-")
	}

	wrapper.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if mi.Value() != "sk" {
		t.Errorf("After backspace, Value() = %q, want %q", mi.Value(), "sk")
	}
}

func TestMaskedInputWrapper_FocusBlur(t *testing.T) {
	mi := NewMaskedInput("API Key", "help")
	wrapper := WrapMaskedInput(&mi)

	if mi.Focused() {
		t.Error("MaskedInput should not be focused initially")
	}

	wrapper.Focus()
	if !mi.Focused() {
		t.Error("MaskedInput should be focused after Focus()")
	}

	wrapper.Blur()
	if mi.Focused() {
		t.Error("MaskedInput should not be focused after Blur()")
	}
}

func TestMaskedInputWrapper_IsDirty(t *testing.T) {
	mi := NewMaskedInput("API Key", "help")
	wrapper := WrapMaskedInput(&mi)

	if wrapper.IsDirty() {
		t.Error("IsDirty() should be false initially")
	}

	wrapper.Focus()
	wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if !wrapper.IsDirty() {
		t.Error("IsDirty() should be true after input")
	}
}

func TestFocusableSlice_Current_AfterNavigation(t *testing.T) {
	tf1 := NewTextField("Field1")
	tf2 := NewTextField("Field2")

	fs := NewFocusableSlice(
		WrapTextField(&tf1),
		WrapTextField(&tf2),
	)

	fs.FocusFirst()
	fs.FocusNext()

	current := fs.Current()
	if current == nil {
		t.Error("Current() should not be nil after navigation")
	}
}

func TestFocusableSlice_UpdateCurrent_NilSafe(t *testing.T) {
	fs := NewFocusableSlice()

	cmd := fs.UpdateCurrent(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("UpdateCurrent on empty slice should return nil")
	}
}

func TestFocusableSlice_FocusFirst_NonEmpty(t *testing.T) {
	tf := NewTextField("Test")
	fs := NewFocusableSlice(WrapTextField(&tf))

	fs.FocusLast()
	cmd := fs.FocusFirst()

	if fs.Index() != 0 {
		t.Errorf("Index() = %d, want 0", fs.Index())
	}
	if cmd == nil {
		t.Error("FocusFirst should return a command")
	}
}

func TestFocusableSlice_FocusLast_NonEmpty(t *testing.T) {
	tf1 := NewTextField("Field1")
	tf2 := NewTextField("Field2")
	tf3 := NewTextField("Field3")

	fs := NewFocusableSlice(
		WrapTextField(&tf1),
		WrapTextField(&tf2),
		WrapTextField(&tf3),
	)

	fs.FocusFirst()
	cmd := fs.FocusLast()

	if fs.Index() != 2 {
		t.Errorf("Index() = %d, want 2", fs.Index())
	}
	if !tf3.Focused() {
		t.Error("tf3 should be focused after FocusLast")
	}
	if cmd == nil {
		t.Error("FocusLast should return a command")
	}
}
