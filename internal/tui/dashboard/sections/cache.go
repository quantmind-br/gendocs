package sections

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/types"
	"github.com/user/gendocs/internal/tui/dashboard/validation"
)

type CacheSectionModel struct {
	enabled   components.ToggleModel
	maxSize   components.TextFieldModel
	ttl       components.TextFieldModel
	cachePath components.TextFieldModel

	focusIndex int
}

func NewCacheSection() *CacheSectionModel {
	return &CacheSectionModel{
		enabled: components.NewToggle("Enabled", "Enable LLM response caching"),
		maxSize: components.NewTextField("Max Size",
			components.WithPlaceholder("1000"),
			components.WithValidator(validation.ValidateIntRange(1, 100000)),
			components.WithHelp("Maximum cache entries")),
		ttl: components.NewTextField("TTL (days)",
			components.WithPlaceholder("7"),
			components.WithValidator(validation.ValidateIntRange(1, 365)),
			components.WithHelp("Cache entry lifetime in days")),
		cachePath: components.NewTextField("Cache Path",
			components.WithPlaceholder(".ai/llm_cache.json"),
			components.WithValidator(validation.ValidatePath()),
			components.WithHelp("Path to cache file")),
	}
}

func (m *CacheSectionModel) Title() string { return "LLM Cache Settings" }
func (m *CacheSectionModel) Icon() string  { return "💾" }
func (m *CacheSectionModel) Description() string {
	return "Configure response caching to reduce API costs"
}

func (m *CacheSectionModel) Init() tea.Cmd { return nil }

func (m *CacheSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 4
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 3
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.enabled, _ = m.enabled.Update(msg)
	case 1:
		m.maxSize, _ = m.maxSize.Update(msg)
	case 2:
		m.ttl, _ = m.ttl.Update(msg)
	case 3:
		m.cachePath, _ = m.cachePath.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *CacheSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.enabled.Blur()
	case 1:
		m.maxSize.Blur()
	case 2:
		m.ttl.Blur()
	case 3:
		m.cachePath.Blur()
	}
}

func (m *CacheSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.enabled.Focus()
	case 1:
		return m.maxSize.Focus()
	case 2:
		return m.ttl.Focus()
	case 3:
		return m.cachePath.Focus()
	}
	return nil
}

func (m *CacheSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.enabled.View(),
		"",
		m.maxSize.View(),
		"",
		m.ttl.View(),
		"",
		m.cachePath.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *CacheSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *CacheSectionModel) IsDirty() bool {
	return m.enabled.IsDirty() || m.maxSize.IsDirty() || m.ttl.IsDirty() || m.cachePath.IsDirty()
}

func (m *CacheSectionModel) GetValues() map[string]any {
	values := map[string]any{
		KeyCacheEnabled: m.enabled.Value(),
		KeyCachePath:    m.cachePath.Value(),
	}
	if v := m.maxSize.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[KeyCacheMaxSize] = i
		}
	}
	if v := m.ttl.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[KeyCacheTTL] = i
		}
	}
	return values
}

func (m *CacheSectionModel) SetValues(values map[string]any) error {
	if v, ok := values[KeyCacheEnabled].(bool); ok {
		m.enabled.SetValue(v)
	}
	if v, ok := values[KeyCacheMaxSize].(int); ok {
		m.maxSize.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[KeyCacheTTL].(int); ok {
		m.ttl.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[KeyCachePath].(string); ok {
		m.cachePath.SetValue(v)
	}
	return nil
}

func (m *CacheSectionModel) FocusFirst() tea.Cmd {
	m.blurAll()
	m.focusIndex = 0
	return m.enabled.Focus()
}

func (m *CacheSectionModel) FocusLast() tea.Cmd {
	m.blurAll()
	m.focusIndex = 3
	return m.cachePath.Focus()
}

func (m *CacheSectionModel) blurAll() {
	m.enabled.Blur()
	m.maxSize.Blur()
	m.ttl.Blur()
	m.cachePath.Blur()
}
