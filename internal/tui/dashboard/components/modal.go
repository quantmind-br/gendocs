package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type ModalAction int

const (
	ModalActionSave ModalAction = iota
	ModalActionDiscard
	ModalActionCancel
)

type ModalResultMsg struct {
	Action ModalAction
}

type ModalModel struct {
	title       string
	message     string
	visible     bool
	focusIndex  int
	width       int
	height      int
	showSave    bool
	showDiscard bool
	showCancel  bool
}

type ModalOption func(*ModalModel)

func WithModalSave() ModalOption {
	return func(m *ModalModel) {
		m.showSave = true
	}
}

func WithModalDiscard() ModalOption {
	return func(m *ModalModel) {
		m.showDiscard = true
	}
}

func WithModalCancel() ModalOption {
	return func(m *ModalModel) {
		m.showCancel = true
	}
}

func NewModal(title, message string, opts ...ModalOption) ModalModel {
	m := ModalModel{
		title:       title,
		message:     message,
		showSave:    false,
		showDiscard: false,
		showCancel:  true,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func NewConfirmModal(title, message string) ModalModel {
	return NewModal(title, message,
		WithModalSave(),
		WithModalDiscard(),
		WithModalCancel(),
	)
}

func (m ModalModel) Init() tea.Cmd {
	return nil
}

func (m ModalModel) Update(msg tea.Msg) (ModalModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		buttons := m.getButtons()
		buttonCount := len(buttons)

		switch msg.String() {
		case "left", "h":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = buttonCount - 1
			}

		case "right", "l", "tab":
			m.focusIndex = (m.focusIndex + 1) % buttonCount

		case "shift+tab":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = buttonCount - 1
			}

		case "enter", " ":
			action := buttons[m.focusIndex]
			m.visible = false
			return m, func() tea.Msg {
				return ModalResultMsg{Action: action}
			}

		case "esc":
			m.visible = false
			return m, func() tea.Msg {
				return ModalResultMsg{Action: ModalActionCancel}
			}

		case "s":
			if m.showSave {
				m.visible = false
				return m, func() tea.Msg {
					return ModalResultMsg{Action: ModalActionSave}
				}
			}

		case "d":
			if m.showDiscard {
				m.visible = false
				return m, func() tea.Msg {
					return ModalResultMsg{Action: ModalActionDiscard}
				}
			}

		case "c":
			m.visible = false
			return m, func() tea.Msg {
				return ModalResultMsg{Action: ModalActionCancel}
			}
		}
	}

	return m, nil
}

func (m ModalModel) getButtons() []ModalAction {
	var buttons []ModalAction
	if m.showSave {
		buttons = append(buttons, ModalActionSave)
	}
	if m.showDiscard {
		buttons = append(buttons, ModalActionDiscard)
	}
	if m.showCancel {
		buttons = append(buttons, ModalActionCancel)
	}
	return buttons
}

func (m ModalModel) View() string {
	if !m.visible {
		return ""
	}

	const minModalWidth = 30
	modalWidth := 50
	if m.width > 0 && m.width < modalWidth+10 {
		modalWidth = max(m.width-10, minModalWidth)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(tui.ColorPrimary).
		Bold(true).
		Width(modalWidth).
		Align(lipgloss.Center)

	messageStyle := lipgloss.NewStyle().
		Foreground(tui.ColorText).
		Width(modalWidth).
		Align(lipgloss.Center).
		Padding(1, 0)

	buttons := m.getButtons()
	var buttonViews []string

	for i, action := range buttons {
		label := m.getButtonLabel(action)
		shortcut := m.getButtonShortcut(action)

		var style lipgloss.Style
		if i == m.focusIndex {
			style = lipgloss.NewStyle().
				Background(tui.ColorPrimary).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 2).
				Bold(true)
		} else {
			style = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(tui.ColorSubtle).
				Foreground(tui.ColorTextDim).
				Padding(0, 1)
		}

		buttonText := label
		if shortcut != "" {
			buttonText = "[" + shortcut + "] " + label
		}
		buttonViews = append(buttonViews, style.Render(buttonText))
	}

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(buttonViews, "  "))
	buttonsContainer := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Render(buttonsRow)

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(m.title),
		messageStyle.Render(m.message),
		buttonsContainer,
	)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorPrimary).
		Background(tui.ColorBg).
		Padding(1, 2)

	modal := modalStyle.Render(content)

	if m.width > 0 && m.height > 0 {
		modalHeight := lipgloss.Height(modal)
		modalWidth := lipgloss.Width(modal)

		topPadding := (m.height - modalHeight) / 2
		leftPadding := (m.width - modalWidth) / 2

		if topPadding < 0 {
			topPadding = 0
		}
		if leftPadding < 0 {
			leftPadding = 0
		}

		modal = lipgloss.NewStyle().
			MarginTop(topPadding).
			MarginLeft(leftPadding).
			Render(modal)
	}

	return modal
}

func (m ModalModel) getButtonLabel(action ModalAction) string {
	switch action {
	case ModalActionSave:
		return "Save"
	case ModalActionDiscard:
		return "Discard"
	case ModalActionCancel:
		return "Cancel"
	}
	return ""
}

func (m ModalModel) getButtonShortcut(action ModalAction) string {
	switch action {
	case ModalActionSave:
		return "S"
	case ModalActionDiscard:
		return "D"
	case ModalActionCancel:
		return "C"
	}
	return ""
}

func (m *ModalModel) Show() {
	m.visible = true
	m.focusIndex = 0
}

func (m *ModalModel) Hide() {
	m.visible = false
}

func (m ModalModel) Visible() bool {
	return m.visible
}

func (m *ModalModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
