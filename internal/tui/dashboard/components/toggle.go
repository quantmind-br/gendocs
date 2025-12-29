package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type ToggleModel struct {
	label       string
	helpText    string
	value       bool
	focused     bool
	dirty       bool
	originalVal bool
}

func NewToggle(label string, helpText string) ToggleModel {
	return ToggleModel{
		label:    label,
		helpText: helpText,
	}
}

func (m ToggleModel) Init() tea.Cmd {
	return nil
}

func (m ToggleModel) Update(msg tea.Msg) (ToggleModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			m.value = !m.value
			m.dirty = m.value != m.originalVal
		case "y", "Y":
			m.value = true
			m.dirty = m.value != m.originalVal
		case "n", "N":
			m.value = false
			m.dirty = m.value != m.originalVal
		}
	}
	return m, nil
}

func (m ToggleModel) View() string {
	label := tui.StyleFormLabel.Render(m.label)

	var toggleView string
	if m.value {
		toggleView = tui.StyleSuccess.Render("[✓] Enabled")
	} else {
		toggleView = tui.StyleMuted.Render("[ ] Disabled")
	}

	if m.focused {
		toggleView = tui.StyleHighlight.Render("▸ ") + toggleView
	} else {
		toggleView = "  " + toggleView
	}

	helpView := tui.StyleFormHelp.Render(m.helpText)

	return lipgloss.JoinVertical(lipgloss.Left, label, toggleView, helpView)
}

func (m *ToggleModel) SetValue(v bool) {
	m.value = v
	m.originalVal = v
	m.dirty = false
}

func (m ToggleModel) Value() bool {
	return m.value
}

func (m ToggleModel) IsDirty() bool {
	return m.dirty
}

func (m *ToggleModel) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *ToggleModel) Blur() {
	m.focused = false
}

func (m ToggleModel) Focused() bool {
	return m.focused
}
