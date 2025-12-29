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

type RetrySectionModel struct {
	maxAttempts       components.TextFieldModel
	multiplier        components.TextFieldModel
	maxWaitPerAttempt components.TextFieldModel
	maxTotalWait      components.TextFieldModel

	focusIndex int
}

func NewRetrySection() *RetrySectionModel {
	return &RetrySectionModel{
		maxAttempts: components.NewTextField("Max Attempts",
			components.WithPlaceholder("5"),
			components.WithValidator(validation.ValidateIntRange(1, 20)),
			components.WithHelp("Maximum retry attempts")),
		multiplier: components.NewTextField("Backoff Multiplier",
			components.WithPlaceholder("1"),
			components.WithValidator(validation.ValidateIntRange(1, 10)),
			components.WithHelp("Exponential backoff multiplier")),
		maxWaitPerAttempt: components.NewTextField("Max Wait Per Attempt (s)",
			components.WithPlaceholder("60"),
			components.WithValidator(validation.ValidateIntRange(1, 300)),
			components.WithHelp("Max wait between attempts")),
		maxTotalWait: components.NewTextField("Max Total Wait (s)",
			components.WithPlaceholder("300"),
			components.WithValidator(validation.ValidateIntRange(1, 3600)),
			components.WithHelp("Total max wait for all retries")),
	}
}

func (m *RetrySectionModel) Title() string { return "Retry Policy" }
func (m *RetrySectionModel) Icon() string  { return "🔄" }
func (m *RetrySectionModel) Description() string {
	return "Configure HTTP retry behavior for API calls"
}

func (m *RetrySectionModel) Init() tea.Cmd { return nil }

func (m *RetrySectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.maxAttempts, _ = m.maxAttempts.Update(msg)
	case 1:
		m.multiplier, _ = m.multiplier.Update(msg)
	case 2:
		m.maxWaitPerAttempt, _ = m.maxWaitPerAttempt.Update(msg)
	case 3:
		m.maxTotalWait, _ = m.maxTotalWait.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *RetrySectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.maxAttempts.Blur()
	case 1:
		m.multiplier.Blur()
	case 2:
		m.maxWaitPerAttempt.Blur()
	case 3:
		m.maxTotalWait.Blur()
	}
}

func (m *RetrySectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.maxAttempts.Focus()
	case 1:
		return m.multiplier.Focus()
	case 2:
		return m.maxWaitPerAttempt.Focus()
	case 3:
		return m.maxTotalWait.Focus()
	}
	return nil
}

func (m *RetrySectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.maxAttempts.View(),
		"",
		m.multiplier.View(),
		"",
		m.maxWaitPerAttempt.View(),
		"",
		m.maxTotalWait.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *RetrySectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *RetrySectionModel) IsDirty() bool {
	return m.maxAttempts.IsDirty() || m.multiplier.IsDirty() ||
		m.maxWaitPerAttempt.IsDirty() || m.maxTotalWait.IsDirty()
}

func (m *RetrySectionModel) GetValues() map[string]any {
	values := map[string]any{}
	if v := m.maxAttempts.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_attempts"] = i
		}
	}
	if v := m.multiplier.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["multiplier"] = i
		}
	}
	if v := m.maxWaitPerAttempt.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_wait_per_attempt"] = i
		}
	}
	if v := m.maxTotalWait.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_total_wait"] = i
		}
	}
	return values
}

func (m *RetrySectionModel) SetValues(values map[string]any) error {
	if v, ok := values["max_attempts"].(int); ok {
		m.maxAttempts.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["multiplier"].(int); ok {
		m.multiplier.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["max_wait_per_attempt"].(int); ok {
		m.maxWaitPerAttempt.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["max_total_wait"].(int); ok {
		m.maxTotalWait.SetValue(strconv.Itoa(v))
	}
	return nil
}

func (m *RetrySectionModel) FocusFirst() tea.Cmd {
	m.focusIndex = 0
	return m.maxAttempts.Focus()
}

func (m *RetrySectionModel) FocusLast() tea.Cmd {
	m.focusIndex = 3
	return m.maxTotalWait.Focus()
}
