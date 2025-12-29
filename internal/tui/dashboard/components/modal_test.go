package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModal_NewModal_CreatesWithTitle(t *testing.T) {
	m := NewModal("Test Title", "Test message")

	m.Show()
	view := m.View()

	if !strings.Contains(view, "Test Title") {
		t.Error("Modal view should contain the title")
	}
}

func TestModal_NewModal_CreatesWithMessage(t *testing.T) {
	m := NewModal("Title", "Test message content")

	m.Show()
	view := m.View()

	if !strings.Contains(view, "Test message content") {
		t.Error("Modal view should contain the message")
	}
}

func TestModal_NewModal_DefaultsWithCancelOnly(t *testing.T) {
	m := NewModal("Title", "Message")

	if m.showSave {
		t.Error("Default modal should not show Save button")
	}
	if m.showDiscard {
		t.Error("Default modal should not show Discard button")
	}
	if !m.showCancel {
		t.Error("Default modal should show Cancel button")
	}
}

func TestModal_NewConfirmModal_ShowsAllButtons(t *testing.T) {
	m := NewConfirmModal("Title", "Message")

	if !m.showSave {
		t.Error("Confirm modal should show Save button")
	}
	if !m.showDiscard {
		t.Error("Confirm modal should show Discard button")
	}
	if !m.showCancel {
		t.Error("Confirm modal should show Cancel button")
	}
}

func TestModal_Show_MakesVisible(t *testing.T) {
	m := NewModal("Title", "Message")

	if m.Visible() {
		t.Error("Modal should not be visible initially")
	}

	m.Show()

	if !m.Visible() {
		t.Error("Modal should be visible after Show()")
	}
}

func TestModal_Hide_MakesInvisible(t *testing.T) {
	m := NewModal("Title", "Message")
	m.Show()

	m.Hide()

	if m.Visible() {
		t.Error("Modal should not be visible after Hide()")
	}
}

func TestModal_View_EmptyWhenNotVisible(t *testing.T) {
	m := NewModal("Title", "Message")

	view := m.View()

	if view != "" {
		t.Error("View should be empty when modal is not visible")
	}
}

func TestModal_Update_IgnoresWhenNotVisible(t *testing.T) {
	m := NewModal("Title", "Message")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Should not return command when modal is not visible")
	}
}

func TestModal_Update_EnterReturnsResult(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Should return command when Enter is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionSave {
		t.Errorf("Expected ModalActionSave, got %v", result.Action)
	}
}

func TestModal_Update_EscapeReturnsCancelResult(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Should return command when Escape is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionCancel {
		t.Errorf("Expected ModalActionCancel, got %v", result.Action)
	}
}

func TestModal_Update_RightNavigatesToNextButton(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	if m.focusIndex != 0 {
		t.Fatalf("Expected initial focus index 0, got %d", m.focusIndex)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})

	if m.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after Right, got %d", m.focusIndex)
	}
}

func TestModal_Update_LeftNavigatesToPrevButton(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 1

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})

	if m.focusIndex != 0 {
		t.Errorf("Expected focus index 0 after Left, got %d", m.focusIndex)
	}
}

func TestModal_Update_TabNavigatesToNextButton(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	if m.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after Tab, got %d", m.focusIndex)
	}
}

func TestModal_Update_ShiftTabNavigatesToPrevButton(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 1

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	if m.focusIndex != 0 {
		t.Errorf("Expected focus index 0 after Shift+Tab, got %d", m.focusIndex)
	}
}

func TestModal_Update_NavigationWrapsAround(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 2

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	if m.focusIndex != 0 {
		t.Errorf("Expected focus to wrap to 0, got %d", m.focusIndex)
	}
}

func TestModal_Update_NavigationWrapsBackward(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 0

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})

	if m.focusIndex != 2 {
		t.Errorf("Expected focus to wrap to 2, got %d", m.focusIndex)
	}
}

func TestModal_Update_SShortcutSelectsSave(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if cmd == nil {
		t.Fatal("Should return command when S is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionSave {
		t.Errorf("Expected ModalActionSave, got %v", result.Action)
	}
}

func TestModal_Update_DShortcutSelectsDiscard(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if cmd == nil {
		t.Fatal("Should return command when D is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionDiscard {
		t.Errorf("Expected ModalActionDiscard, got %v", result.Action)
	}
}

func TestModal_Update_CShortcutSelectsCancel(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if cmd == nil {
		t.Fatal("Should return command when C is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionCancel {
		t.Errorf("Expected ModalActionCancel, got %v", result.Action)
	}
}

func TestModal_SetSize_UpdatesDimensions(t *testing.T) {
	m := NewModal("Title", "Message")
	m.Show()

	m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("Expected height 50, got %d", m.height)
	}
}

func TestModal_Update_WindowSizeUpdatesModal(t *testing.T) {
	m := NewModal("Title", "Message")
	m.Show()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

func TestModal_Update_SpaceSelectsCurrentButton(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})

	if cmd == nil {
		t.Fatal("Should return command when Space is pressed")
	}

	msg := cmd()
	result, ok := msg.(ModalResultMsg)
	if !ok {
		t.Fatal("Should return ModalResultMsg")
	}

	if result.Action != ModalActionDiscard {
		t.Errorf("Expected ModalActionDiscard for focus index 1, got %v", result.Action)
	}
}

func TestModal_Init_ReturnsNil(t *testing.T) {
	m := NewModal("Title", "Message")

	if m.Init() != nil {
		t.Error("Init should return nil")
	}
}

func TestModal_View_ContainsButtonLabels(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	view := m.View()

	if !strings.Contains(view, "Save") {
		t.Error("View should contain Save button")
	}
	if !strings.Contains(view, "Discard") {
		t.Error("View should contain Discard button")
	}
	if !strings.Contains(view, "Cancel") {
		t.Error("View should contain Cancel button")
	}
}

func TestModal_View_ContainsShortcuts(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	view := m.View()

	if !strings.Contains(view, "[S]") {
		t.Error("View should contain [S] shortcut")
	}
	if !strings.Contains(view, "[D]") {
		t.Error("View should contain [D] shortcut")
	}
	if !strings.Contains(view, "[C]") {
		t.Error("View should contain [C] shortcut")
	}
}

func TestModal_HidesAfterSelection(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.Visible() {
		t.Error("Modal should be hidden after selection")
	}
}

func TestModal_HidesAfterEscape(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.Visible() {
		t.Error("Modal should be hidden after escape")
	}
}

func TestModal_WithModalSave_EnablesSaveButton(t *testing.T) {
	m := NewModal("Title", "Message", WithModalSave())

	if !m.showSave {
		t.Error("WithModalSave should enable Save button")
	}
}

func TestModal_WithModalDiscard_EnablesDiscardButton(t *testing.T) {
	m := NewModal("Title", "Message", WithModalDiscard())

	if !m.showDiscard {
		t.Error("WithModalDiscard should enable Discard button")
	}
}

func TestModal_WithModalCancel_EnablesCancelButton(t *testing.T) {
	m := NewModal("Title", "Message")
	m.showCancel = false
	m = NewModal("Title", "Message", WithModalCancel())

	if !m.showCancel {
		t.Error("WithModalCancel should enable Cancel button")
	}
}

func TestModal_Show_ResetsFocusIndex(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.focusIndex = 2

	m.Show()

	if m.focusIndex != 0 {
		t.Errorf("Show should reset focus index to 0, got %d", m.focusIndex)
	}
}

func TestModal_HKeyNavigatesLeft(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()
	m.focusIndex = 1

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	if m.focusIndex != 0 {
		t.Errorf("Expected focus index 0 after h, got %d", m.focusIndex)
	}
}

func TestModal_LKeyNavigatesRight(t *testing.T) {
	m := NewConfirmModal("Title", "Message")
	m.Show()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if m.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after l, got %d", m.focusIndex)
	}
}
