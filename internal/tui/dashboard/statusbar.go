package dashboard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/tui"
)

type ConfigScope int

const (
	ScopeGlobal ConfigScope = iota
	ScopeProject
)

func (s ConfigScope) String() string {
	switch s {
	case ScopeGlobal:
		return "Global (~/.gendocs.yaml)"
	case ScopeProject:
		return "Project (.ai/config.yaml)"
	default:
		return "Unknown"
	}
}

type MessageType int

const (
	MessageNone MessageType = iota
	MessageInfo
	MessageSuccess
	MessageError
)

type StatusBarModel struct {
	scope       ConfigScope
	modified    bool
	lastMessage string
	messageType MessageType
	width       int
}

func NewStatusBar() StatusBarModel {
	return StatusBarModel{
		scope: ScopeGlobal,
	}
}

func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case SetModifiedMsg:
		m.modified = bool(msg)
	case SetScopeMsg:
		m.scope = ConfigScope(msg)
	case ShowMessageMsg:
		m.lastMessage = msg.Text
		m.messageType = msg.Type
	case ClearMessageMsg:
		m.lastMessage = ""
		m.messageType = MessageNone
	}
	return m, nil
}

func (m StatusBarModel) View() string {
	scopeText := tui.StyleStatusScope.Render(tui.IconScope + " " + m.scope.String())

	var modifiedText string
	if m.modified {
		modifiedText = tui.StyleStatusModified.Render(" [Modified]")
	} else {
		modifiedText = tui.StyleStatusSaved.Render(" [Saved]")
	}

	var messageText string
	switch m.messageType {
	case MessageSuccess:
		messageText = tui.StyleSuccess.Render(m.lastMessage)
	case MessageError:
		messageText = tui.StyleError.Render(m.lastMessage)
	case MessageInfo:
		messageText = tui.StyleInfo.Render(m.lastMessage)
	}

	hints := tui.StyleMuted.Render("Tab: Switch | Ctrl+S: Save | Ctrl+T: Scope | ?: Help | q: Quit")

	left := scopeText + modifiedText
	leftWidth := len(left)
	hintsWidth := len(hints)
	messageWidth := len(messageText)

	spacer := m.width - leftWidth - hintsWidth - messageWidth - 4
	if spacer < 1 {
		spacer = 1
	}

	content := left + strings.Repeat(" ", spacer/2) + messageText + strings.Repeat(" ", spacer/2) + hints
	return tui.StyleStatusBar.Width(m.width).Render(content)
}

func (m *StatusBarModel) SetScope(scope ConfigScope) {
	m.scope = scope
}

func (m StatusBarModel) Scope() ConfigScope {
	return m.scope
}

func (m StatusBarModel) IsModified() bool {
	return m.modified
}

type SetModifiedMsg bool
type SetScopeMsg ConfigScope
type ShowMessageMsg struct {
	Text string
	Type MessageType
}
type ClearMessageMsg struct{}
