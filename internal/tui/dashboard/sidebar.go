package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type NavItem struct {
	ID          string
	Title       string
	Icon        string
	Description string
}

var DefaultNavItems = []NavItem{
	{ID: "llm", Title: "Analyzer LLM", Icon: "🤖", Description: "LLM for analysis"},
	{ID: "documenter_llm", Title: "Documenter LLM", Icon: "📝", Description: "LLM for README"},
	{ID: "ai_rules_llm", Title: "AI Rules LLM", Icon: "📋", Description: "LLM for AI rules"},
	{ID: "cache", Title: "LLM Cache", Icon: "💾", Description: "Response caching"},
	{ID: "analysis", Title: "Analysis", Icon: "🔍", Description: "Analyzer options"},
	{ID: "retry", Title: "Retry Policy", Icon: "🔄", Description: "HTTP retry settings"},
	{ID: "gemini", Title: "Gemini/Vertex", Icon: "☁️", Description: "Google Cloud options"},
	{ID: "gitlab", Title: "GitLab", Icon: "🦊", Description: "GitLab integration"},
	{ID: "cronjob", Title: "Cronjob", Icon: "⏰", Description: "Scheduled tasks"},
	{ID: "logging", Title: "Logging", Icon: "📝", Description: "Log configuration"},
}

type SidebarModel struct {
	items       []NavItem
	activeIndex int
	focused     bool
	width       int
	height      int
}

func NewSidebar(items []NavItem) SidebarModel {
	return SidebarModel{
		items:       items,
		activeIndex: 0,
		focused:     true,
		width:       26,
	}
}

func (m SidebarModel) Init() tea.Cmd {
	return nil
}

func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.activeIndex > 0 {
				m.activeIndex--
			}
		case "down", "j":
			if m.activeIndex < len(m.items)-1 {
				m.activeIndex++
			}
		case "home", "g":
			m.activeIndex = 0
		case "end", "G":
			m.activeIndex = len(m.items) - 1
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height - 4
	}
	return m, nil
}

func (m SidebarModel) View() string {
	var items []string
	for i, item := range m.items {
		style := tui.StyleNavItem
		prefix := "  "

		if i == m.activeIndex {
			if m.focused {
				style = tui.StyleNavItemActive
				prefix = tui.IconArrow + " "
			} else {
				style = tui.StyleNavItemHover
			}
		}

		line := prefix + item.Icon + " " + item.Title
		items = append(items, style.Render(line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	return tui.StyleSidebarContainer.Height(m.height).Render(content)
}

func (m SidebarModel) ActiveItem() NavItem {
	if m.activeIndex >= 0 && m.activeIndex < len(m.items) {
		return m.items[m.activeIndex]
	}
	return NavItem{}
}

func (m *SidebarModel) SetFocused(focused bool) {
	m.focused = focused
}

func (m SidebarModel) IsFocused() bool {
	return m.focused
}

func (m SidebarModel) ActiveIndex() int {
	return m.activeIndex
}
