package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type ButtonStyle int

const (
	ButtonStylePrimary ButtonStyle = iota
	ButtonStyleSecondary
	ButtonStyleDanger
)

type ButtonModel struct {
	label      string
	helpText   string
	focused    bool
	loading    bool
	style      ButtonStyle
	onPressCmd func() tea.Msg
}

type ButtonOption func(*ButtonModel)

func WithButtonStyle(style ButtonStyle) ButtonOption {
	return func(b *ButtonModel) {
		b.style = style
	}
}

func WithButtonHelp(text string) ButtonOption {
	return func(b *ButtonModel) {
		b.helpText = text
	}
}

func NewButton(label string, onPressCmd func() tea.Msg, opts ...ButtonOption) ButtonModel {
	b := ButtonModel{
		label:      label,
		onPressCmd: onPressCmd,
		style:      ButtonStylePrimary,
	}
	for _, opt := range opts {
		opt(&b)
	}
	return b
}

func (m ButtonModel) Init() tea.Cmd {
	return nil
}

func (m ButtonModel) Update(msg tea.Msg) (ButtonModel, tea.Cmd) {
	if !m.focused || m.loading {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			if m.onPressCmd != nil {
				m.loading = true
				return m, func() tea.Msg {
					return m.onPressCmd()
				}
			}
		}
	}
	return m, nil
}

func (m ButtonModel) View() string {
	var style lipgloss.Style

	baseStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)

	if m.loading {
		style = baseStyle.
			Background(tui.ColorSubtle).
			Foreground(tui.ColorMuted)
		return style.Render("⏳ " + m.label + "...")
	}

	if m.focused {
		switch m.style {
		case ButtonStyleDanger:
			style = baseStyle.
				Background(tui.ColorError).
				Foreground(lipgloss.Color("#FFFFFF"))
		case ButtonStyleSecondary:
			style = baseStyle.
				Background(tui.ColorSubtle).
				Foreground(tui.ColorText)
		default:
			style = baseStyle.
				Background(tui.ColorPrimary).
				Foreground(lipgloss.Color("#FFFFFF"))
		}
	} else {
		switch m.style {
		case ButtonStyleDanger:
			style = baseStyle.
				Border(lipgloss.NormalBorder()).
				BorderForeground(tui.ColorError).
				Foreground(tui.ColorError)
		case ButtonStyleSecondary:
			style = baseStyle.
				Border(lipgloss.NormalBorder()).
				BorderForeground(tui.ColorSubtle).
				Foreground(tui.ColorTextDim)
		default:
			style = baseStyle.
				Border(lipgloss.NormalBorder()).
				BorderForeground(tui.ColorPrimary).
				Foreground(tui.ColorPrimary)
		}
	}

	buttonView := style.Render(m.label)

	if m.helpText != "" {
		helpView := tui.StyleFormHelp.Render(m.helpText)
		return lipgloss.JoinVertical(lipgloss.Left, buttonView, helpView)
	}

	return buttonView
}

func (m *ButtonModel) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *ButtonModel) Blur() {
	m.focused = false
}

func (m ButtonModel) Focused() bool {
	return m.focused
}

func (m *ButtonModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m ButtonModel) IsLoading() bool {
	return m.loading
}

func (m ButtonModel) IsDirty() bool {
	return false
}
