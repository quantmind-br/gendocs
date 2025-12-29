package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type MaskedInputModel struct {
	input       textinput.Model
	label       string
	helpText    string
	revealed    bool
	dirty       bool
	originalVal string
}

func NewMaskedInput(label string, helpText string) MaskedInputModel {
	input := textinput.New()
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	input.CharLimit = 256

	return MaskedInputModel{
		input:    input,
		label:    label,
		helpText: helpText,
	}
}

func (m MaskedInputModel) Init() tea.Cmd {
	return nil
}

func (m MaskedInputModel) Update(msg tea.Msg) (MaskedInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+u" && m.input.Focused() {
			m.revealed = !m.revealed
			if m.revealed {
				m.input.EchoMode = textinput.EchoNormal
			} else {
				m.input.EchoMode = textinput.EchoPassword
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if m.input.Value() != m.originalVal {
		m.dirty = true
	}

	return m, cmd
}

func (m MaskedInputModel) View() string {
	label := tui.StyleFormLabel.Render(m.label) + tui.StyleError.Render(" *")

	var style lipgloss.Style
	if m.input.Focused() {
		style = tui.StyleFormInputFocused
	} else {
		style = tui.StyleFormInput
	}

	inputView := style.Render(m.input.View())

	var helpView string
	if m.input.Focused() {
		revealHint := "Ctrl+U: "
		if m.revealed {
			revealHint += "Hide"
		} else {
			revealHint += "Reveal"
		}
		helpView = tui.StyleFormHelp.Render(m.helpText + " | " + revealHint)
	} else {
		helpView = tui.StyleFormHelp.Render(m.helpText)
	}

	var indicator string
	if !m.input.Focused() && m.input.Value() != "" {
		maskedLen := len(m.input.Value())
		if maskedLen > 8 {
			indicator = tui.StyleMuted.Render(" (" + strings.Repeat("•", 4) + "..." + m.input.Value()[maskedLen-4:] + ")")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, label, inputView+indicator, helpView)
}

func (m *MaskedInputModel) SetValue(v string) {
	m.input.SetValue(v)
	m.originalVal = v
	m.dirty = false
}

func (m MaskedInputModel) Value() string {
	return m.input.Value()
}

func (m MaskedInputModel) IsDirty() bool {
	return m.dirty
}

func (m *MaskedInputModel) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *MaskedInputModel) Blur() {
	m.input.Blur()
}

func (m MaskedInputModel) Focused() bool {
	return m.input.Focused()
}
