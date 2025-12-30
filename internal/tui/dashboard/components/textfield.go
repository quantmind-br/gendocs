package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type TextFieldModel struct {
	input       textinput.Model
	label       string
	helpText    string
	required    bool
	validator   func(string) error
	lastError   error
	dirty       bool
	originalVal string
}

type TextFieldOption func(*TextFieldModel)

func WithPlaceholder(p string) TextFieldOption {
	return func(m *TextFieldModel) {
		m.input.Placeholder = p
	}
}

func WithValidator(v func(string) error) TextFieldOption {
	return func(m *TextFieldModel) {
		m.validator = v
	}
}

func WithRequired() TextFieldOption {
	return func(m *TextFieldModel) {
		m.required = true
	}
}

func WithHelp(h string) TextFieldOption {
	return func(m *TextFieldModel) {
		m.helpText = h
	}
}

func WithCharLimit(limit int) TextFieldOption {
	return func(m *TextFieldModel) {
		m.input.CharLimit = limit
	}
}

func NewTextField(label string, opts ...TextFieldOption) TextFieldModel {
	input := textinput.New()
	input.CharLimit = 256
	// Explicitly initialize with empty value to prevent placeholder leakage
	input.SetValue("")

	m := TextFieldModel{
		input: input,
		label: label,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

func (m TextFieldModel) Init() tea.Cmd {
	return nil
}

func (m TextFieldModel) Update(msg tea.Msg) (TextFieldModel, tea.Cmd) {
	if !m.input.Focused() {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if m.input.Value() != m.originalVal {
		m.dirty = true
	}

	if m.validator != nil {
		m.lastError = m.validator(m.input.Value())
	}

	return m, cmd
}

func (m TextFieldModel) View() string {
	var style lipgloss.Style

	if m.input.Focused() {
		if m.lastError != nil {
			style = tui.StyleFormInputError
		} else {
			style = tui.StyleFormInputFocused
		}
	} else {
		style = tui.StyleFormInput
	}

	label := tui.StyleFormLabel.Render(m.label)
	if m.required {
		label += tui.StyleError.Render(" *")
	}

	inputView := style.Render(m.input.View())

	var helpView string
	if m.lastError != nil {
		helpView = tui.StyleError.Render(m.lastError.Error())
	} else if m.helpText != "" {
		helpView = tui.StyleFormHelp.Render(m.helpText)
	}

	return lipgloss.JoinVertical(lipgloss.Left, label, inputView, helpView)
}

func (m *TextFieldModel) SetValue(v string) {
	m.input.SetValue(v)
	m.input.SetCursor(len(v))
	m.originalVal = v
	m.dirty = false
}

func (m TextFieldModel) Value() string {
	return m.input.Value()
}

func (m TextFieldModel) IsDirty() bool {
	return m.dirty
}

func (m TextFieldModel) IsValid() bool {
	if m.required && m.input.Value() == "" {
		return false
	}
	return m.lastError == nil
}

func (m *TextFieldModel) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *TextFieldModel) Blur() {
	m.input.Blur()
}

func (m TextFieldModel) Focused() bool {
	return m.input.Focused()
}
