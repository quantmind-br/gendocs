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

type CronjobSectionModel struct {
	maxDaysSinceLastCommit components.TextFieldModel
	workingPath            components.TextFieldModel
	groupProjectID         components.TextFieldModel

	focusIndex int
}

func NewCronjobSection() *CronjobSectionModel {
	return &CronjobSectionModel{
		maxDaysSinceLastCommit: components.NewTextField("Max Days Since Last Commit",
			components.WithPlaceholder("30"),
			components.WithValidator(validation.ValidateIntRange(1, 365)),
			components.WithHelp("Skip repos inactive for this many days")),
		workingPath: components.NewTextField("Working Path",
			components.WithPlaceholder("/tmp/gendocs"),
			components.WithValidator(validation.ValidatePath()),
			components.WithHelp("Temp directory for cronjob operations")),
		groupProjectID: components.NewTextField("Group/Project ID",
			components.WithPlaceholder("12345"),
			components.WithValidator(validation.ValidatePositiveInt()),
			components.WithHelp("GitLab group or project ID")),
	}
}

func (m *CronjobSectionModel) Title() string { return "Cronjob Settings" }
func (m *CronjobSectionModel) Icon() string  { return "⏰" }
func (m *CronjobSectionModel) Description() string {
	return "Configure scheduled documentation updates"
}

func (m *CronjobSectionModel) Init() tea.Cmd { return nil }

func (m *CronjobSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 3
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 2
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.maxDaysSinceLastCommit, _ = m.maxDaysSinceLastCommit.Update(msg)
	case 1:
		m.workingPath, _ = m.workingPath.Update(msg)
	case 2:
		m.groupProjectID, _ = m.groupProjectID.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *CronjobSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.maxDaysSinceLastCommit.Blur()
	case 1:
		m.workingPath.Blur()
	case 2:
		m.groupProjectID.Blur()
	}
}

func (m *CronjobSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.maxDaysSinceLastCommit.Focus()
	case 1:
		return m.workingPath.Focus()
	case 2:
		return m.groupProjectID.Focus()
	}
	return nil
}

func (m *CronjobSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.maxDaysSinceLastCommit.View(),
		"",
		m.workingPath.View(),
		"",
		m.groupProjectID.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *CronjobSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *CronjobSectionModel) IsDirty() bool {
	return m.maxDaysSinceLastCommit.IsDirty() || m.workingPath.IsDirty() || m.groupProjectID.IsDirty()
}

func (m *CronjobSectionModel) GetValues() map[string]any {
	values := map[string]any{
		"working_path": m.workingPath.Value(),
	}
	if v := m.maxDaysSinceLastCommit.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_days_since_last_commit"] = i
		}
	}
	if v := m.groupProjectID.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["group_project_id"] = i
		}
	}
	return values
}

func (m *CronjobSectionModel) SetValues(values map[string]any) error {
	if v, ok := values["max_days_since_last_commit"].(int); ok {
		m.maxDaysSinceLastCommit.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["working_path"].(string); ok {
		m.workingPath.SetValue(v)
	}
	if v, ok := values["group_project_id"].(int); ok {
		m.groupProjectID.SetValue(strconv.Itoa(v))
	}
	return nil
}

func (m *CronjobSectionModel) FocusFirst() tea.Cmd {
	m.focusIndex = 0
	return m.maxDaysSinceLastCommit.Focus()
}

func (m *CronjobSectionModel) FocusLast() tea.Cmd {
	m.focusIndex = 2
	return m.groupProjectID.Focus()
}
