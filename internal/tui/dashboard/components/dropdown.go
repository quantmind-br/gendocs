package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type DropdownOption struct {
	Value string
	Label string
}

type DropdownModel struct {
	label       string
	helpText    string
	options     []DropdownOption
	selected    int
	expanded    bool
	focused     bool
	dirty       bool
	originalVal int
}

func NewDropdown(label string, options []DropdownOption, helpText string) DropdownModel {
	return DropdownModel{
		label:    label,
		options:  options,
		helpText: helpText,
	}
}

func (m DropdownModel) Init() tea.Cmd {
	return nil
}

func (m DropdownModel) Update(msg tea.Msg) (DropdownModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			if m.expanded {
				m.expanded = false
				m.dirty = m.selected != m.originalVal
			} else {
				m.expanded = true
			}
		case "esc":
			m.expanded = false
		case "up", "k":
			if m.expanded && m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.expanded && m.selected < len(m.options)-1 {
				m.selected++
			}
		}
	}
	return m, nil
}

func (m DropdownModel) View() string {
	label := tui.StyleFormLabel.Render(m.label)

	var currentView string
	if m.selected >= 0 && m.selected < len(m.options) {
		currentView = m.options[m.selected].Label
	} else {
		currentView = "Select..."
	}

	var style lipgloss.Style
	if m.focused {
		style = tui.StyleFormInputFocused
	} else {
		style = tui.StyleFormInput
	}

	arrow := " ▼"
	if m.expanded {
		arrow = " ▲"
	}

	selectedView := style.Render(currentView + arrow)

	var optionsView string
	if m.expanded {
		var opts []string
		for i, opt := range m.options {
			prefix := "  "
			optStyle := tui.StyleMuted
			if i == m.selected {
				prefix = "▸ "
				optStyle = tui.StyleHighlight
			}
			opts = append(opts, optStyle.Render(prefix+opt.Label))
		}
		optionsView = lipgloss.JoinVertical(lipgloss.Left, opts...)
		optionsView = tui.StyleBox.Render(optionsView)
	}

	helpView := tui.StyleFormHelp.Render(m.helpText)

	return lipgloss.JoinVertical(lipgloss.Left, label, selectedView, optionsView, helpView)
}

func (m *DropdownModel) SetValue(value string) {
	for i, opt := range m.options {
		if opt.Value == value {
			m.selected = i
			m.originalVal = i
			m.dirty = false
			return
		}
	}
}

func (m DropdownModel) Value() string {
	if m.selected >= 0 && m.selected < len(m.options) {
		return m.options[m.selected].Value
	}
	return ""
}

func (m DropdownModel) IsDirty() bool {
	return m.dirty
}

func (m *DropdownModel) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *DropdownModel) Blur() {
	m.focused = false
	m.expanded = false
}

func (m DropdownModel) Focused() bool {
	return m.focused
}
