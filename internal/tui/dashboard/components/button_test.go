package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type testButtonMsg struct{}

func TestButton_NewButton_CreatesWithLabel(t *testing.T) {
	b := NewButton("Click Me", nil)

	if !strings.Contains(b.View(), "Click Me") {
		t.Error("Button view should contain the label")
	}
}

func TestButton_NewButton_DefaultStyleIsPrimary(t *testing.T) {
	b := NewButton("Test", nil)

	if b.style != ButtonStylePrimary {
		t.Errorf("Expected default style to be ButtonStylePrimary, got %v", b.style)
	}
}

func TestButton_WithButtonStyle_SetsStyle(t *testing.T) {
	b := NewButton("Test", nil, WithButtonStyle(ButtonStyleDanger))

	if b.style != ButtonStyleDanger {
		t.Errorf("Expected style to be ButtonStyleDanger, got %v", b.style)
	}
}

func TestButton_WithButtonHelp_SetsHelpText(t *testing.T) {
	b := NewButton("Test", nil, WithButtonHelp("Help text"))

	if b.helpText != "Help text" {
		t.Errorf("Expected help text 'Help text', got %q", b.helpText)
	}
}

func TestButton_View_ContainsHelpText(t *testing.T) {
	b := NewButton("Test", nil, WithButtonHelp("Press Enter to submit"))
	view := b.View()

	if !strings.Contains(view, "Press Enter to submit") {
		t.Error("View should contain help text")
	}
}

func TestButton_Focus_SetsFocused(t *testing.T) {
	b := NewButton("Test", nil)

	if b.Focused() {
		t.Error("Button should not be focused initially")
	}

	b.Focus()

	if !b.Focused() {
		t.Error("Button should be focused after Focus()")
	}
}

func TestButton_Blur_UnsetsFocused(t *testing.T) {
	b := NewButton("Test", nil)
	b.Focus()

	b.Blur()

	if b.Focused() {
		t.Error("Button should not be focused after Blur()")
	}
}

func TestButton_Update_EnterTriggersCallback(t *testing.T) {
	called := false
	b := NewButton("Test", func() tea.Msg {
		called = true
		return testButtonMsg{}
	})
	b.Focus()

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	if !called {
		msg := cmd()
		if _, ok := msg.(testButtonMsg); !ok {
			t.Error("Expected testButtonMsg from command")
		}
	}
}

func TestButton_Update_SpaceTriggersCallback(t *testing.T) {
	called := false
	b := NewButton("Test", func() tea.Msg {
		called = true
		return testButtonMsg{}
	})
	b.Focus()

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeySpace})

	if cmd == nil {
		t.Error("Expected command to be returned")
	}
	_ = called
}

func TestButton_Update_IgnoresWhenNotFocused(t *testing.T) {
	b := NewButton("Test", func() tea.Msg {
		return testButtonMsg{}
	})

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Should not trigger callback when not focused")
	}
}

func TestButton_Update_IgnoresWhenLoading(t *testing.T) {
	b := NewButton("Test", func() tea.Msg {
		return testButtonMsg{}
	})
	b.Focus()
	b.SetLoading(true)

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Should not trigger callback when loading")
	}
}

func TestButton_SetLoading_ChangesState(t *testing.T) {
	b := NewButton("Test", nil)

	if b.IsLoading() {
		t.Error("Button should not be loading initially")
	}

	b.SetLoading(true)

	if !b.IsLoading() {
		t.Error("Button should be loading after SetLoading(true)")
	}

	b.SetLoading(false)

	if b.IsLoading() {
		t.Error("Button should not be loading after SetLoading(false)")
	}
}

func TestButton_View_LoadingState(t *testing.T) {
	b := NewButton("Submit", nil)
	b.SetLoading(true)

	view := b.View()

	if !strings.Contains(view, "⏳") {
		t.Error("Loading view should contain spinner icon")
	}
	if !strings.Contains(view, "Submit...") {
		t.Error("Loading view should contain label with ellipsis")
	}
}

func TestButton_IsDirty_AlwaysFalse(t *testing.T) {
	b := NewButton("Test", nil)

	if b.IsDirty() {
		t.Error("Button should never be dirty")
	}

	b.Focus()
	b.SetLoading(true)

	if b.IsDirty() {
		t.Error("Button should still not be dirty after state changes")
	}
}

func TestButton_Init_ReturnsNil(t *testing.T) {
	b := NewButton("Test", nil)

	if b.Init() != nil {
		t.Error("Init should return nil")
	}
}

func TestButton_View_FocusedStyleDiffers(t *testing.T) {
	b := NewButton("Test", nil)
	unfocusedView := b.View()

	b.Focus()
	focusedView := b.View()

	if unfocusedView == focusedView {
		t.Error("Focused and unfocused views should differ")
	}
}

func TestButton_View_SecondaryStyle(t *testing.T) {
	b := NewButton("Cancel", nil, WithButtonStyle(ButtonStyleSecondary))
	b.Focus()
	_ = b.View()
}

func TestButton_View_DangerStyle(t *testing.T) {
	b := NewButton("Delete", nil, WithButtonStyle(ButtonStyleDanger))
	b.Focus()
	_ = b.View()
}

func TestButton_Update_NoCallbackDoesNotPanic(t *testing.T) {
	b := NewButton("Test", nil)
	b.Focus()

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Should not return command when callback is nil")
	}
}

func TestButton_Update_SetsLoadingOnPress(t *testing.T) {
	b := NewButton("Test", func() tea.Msg {
		return testButtonMsg{}
	})
	b.Focus()

	b, _ = b.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !b.IsLoading() {
		t.Error("Button should be loading after press")
	}
}
