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

type AnalysisSectionModel struct {
	excludeStructure components.ToggleModel
	excludeDataFlow  components.ToggleModel
	excludeDeps      components.ToggleModel
	excludeReqFlow   components.ToggleModel
	excludeAPI       components.ToggleModel
	maxWorkers       components.TextFieldModel
	maxHashWorkers   components.TextFieldModel
	force            components.ToggleModel
	incremental      components.ToggleModel

	focusIndex int
}

func NewAnalysisSection() *AnalysisSectionModel {
	return &AnalysisSectionModel{
		excludeStructure: components.NewToggle("Exclude Code Structure", "Skip code structure analysis"),
		excludeDataFlow:  components.NewToggle("Exclude Data Flow", "Skip data flow analysis"),
		excludeDeps:      components.NewToggle("Exclude Dependencies", "Skip dependency analysis"),
		excludeReqFlow:   components.NewToggle("Exclude Request Flow", "Skip request flow analysis"),
		excludeAPI:       components.NewToggle("Exclude API Analysis", "Skip API analysis"),
		maxWorkers: components.NewTextField("Max Workers",
			components.WithPlaceholder("0 (auto)"),
			components.WithValidator(validation.ValidateIntRange(0, 32)),
			components.WithHelp("0 = auto-detect CPU cores")),
		maxHashWorkers: components.NewTextField("Max Hash Workers",
			components.WithPlaceholder("0 (auto)"),
			components.WithValidator(validation.ValidateIntRange(0, 32)),
			components.WithHelp("Parallel file hashing workers")),
		force:       components.NewToggle("Force Re-analysis", "Ignore cache and re-analyze all"),
		incremental: components.NewToggle("Incremental Analysis", "Only analyze changed files"),
	}
}

func (m *AnalysisSectionModel) Title() string       { return "Analysis Settings" }
func (m *AnalysisSectionModel) Icon() string        { return "🔍" }
func (m *AnalysisSectionModel) Description() string { return "Configure codebase analysis options" }

func (m *AnalysisSectionModel) Init() tea.Cmd { return nil }

func (m *AnalysisSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 9
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 8
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.excludeStructure, _ = m.excludeStructure.Update(msg)
	case 1:
		m.excludeDataFlow, _ = m.excludeDataFlow.Update(msg)
	case 2:
		m.excludeDeps, _ = m.excludeDeps.Update(msg)
	case 3:
		m.excludeReqFlow, _ = m.excludeReqFlow.Update(msg)
	case 4:
		m.excludeAPI, _ = m.excludeAPI.Update(msg)
	case 5:
		m.maxWorkers, _ = m.maxWorkers.Update(msg)
	case 6:
		m.maxHashWorkers, _ = m.maxHashWorkers.Update(msg)
	case 7:
		m.force, _ = m.force.Update(msg)
	case 8:
		m.incremental, _ = m.incremental.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *AnalysisSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.excludeStructure.Blur()
	case 1:
		m.excludeDataFlow.Blur()
	case 2:
		m.excludeDeps.Blur()
	case 3:
		m.excludeReqFlow.Blur()
	case 4:
		m.excludeAPI.Blur()
	case 5:
		m.maxWorkers.Blur()
	case 6:
		m.maxHashWorkers.Blur()
	case 7:
		m.force.Blur()
	case 8:
		m.incremental.Blur()
	}
}

func (m *AnalysisSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.excludeStructure.Focus()
	case 1:
		return m.excludeDataFlow.Focus()
	case 2:
		return m.excludeDeps.Focus()
	case 3:
		return m.excludeReqFlow.Focus()
	case 4:
		return m.excludeAPI.Focus()
	case 5:
		return m.maxWorkers.Focus()
	case 6:
		return m.maxHashWorkers.Focus()
	case 7:
		return m.force.Focus()
	case 8:
		return m.incremental.Focus()
	}
	return nil
}

func (m *AnalysisSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	excludeSection := tui.StyleMuted.Render("Exclusions:")
	workers := lipgloss.JoinHorizontal(lipgloss.Top,
		m.maxWorkers.View(),
		"    ",
		m.maxHashWorkers.View(),
	)

	fields := lipgloss.JoinVertical(lipgloss.Left,
		excludeSection,
		m.excludeStructure.View(),
		m.excludeDataFlow.View(),
		m.excludeDeps.View(),
		m.excludeReqFlow.View(),
		m.excludeAPI.View(),
		"",
		workers,
		"",
		m.force.View(),
		m.incremental.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *AnalysisSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *AnalysisSectionModel) IsDirty() bool {
	return m.excludeStructure.IsDirty() || m.excludeDataFlow.IsDirty() ||
		m.excludeDeps.IsDirty() || m.excludeReqFlow.IsDirty() || m.excludeAPI.IsDirty() ||
		m.maxWorkers.IsDirty() || m.maxHashWorkers.IsDirty() ||
		m.force.IsDirty() || m.incremental.IsDirty()
}

func (m *AnalysisSectionModel) GetValues() map[string]any {
	values := map[string]any{
		"exclude_code_structure": m.excludeStructure.Value(),
		"exclude_data_flow":      m.excludeDataFlow.Value(),
		"exclude_dependencies":   m.excludeDeps.Value(),
		"exclude_request_flow":   m.excludeReqFlow.Value(),
		"exclude_api_analysis":   m.excludeAPI.Value(),
		"force":                  m.force.Value(),
		"incremental":            m.incremental.Value(),
	}
	if v := m.maxWorkers.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_workers"] = i
		}
	}
	if v := m.maxHashWorkers.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values["max_hash_workers"] = i
		}
	}
	return values
}

func (m *AnalysisSectionModel) SetValues(values map[string]any) error {
	if v, ok := values["exclude_code_structure"].(bool); ok {
		m.excludeStructure.SetValue(v)
	}
	if v, ok := values["exclude_data_flow"].(bool); ok {
		m.excludeDataFlow.SetValue(v)
	}
	if v, ok := values["exclude_dependencies"].(bool); ok {
		m.excludeDeps.SetValue(v)
	}
	if v, ok := values["exclude_request_flow"].(bool); ok {
		m.excludeReqFlow.SetValue(v)
	}
	if v, ok := values["exclude_api_analysis"].(bool); ok {
		m.excludeAPI.SetValue(v)
	}
	if v, ok := values["max_workers"].(int); ok {
		m.maxWorkers.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["max_hash_workers"].(int); ok {
		m.maxHashWorkers.SetValue(strconv.Itoa(v))
	}
	if v, ok := values["force"].(bool); ok {
		m.force.SetValue(v)
	}
	if v, ok := values["incremental"].(bool); ok {
		m.incremental.SetValue(v)
	}
	return nil
}

func (m *AnalysisSectionModel) FocusFirst() tea.Cmd {
	m.focusIndex = 0
	return m.excludeStructure.Focus()
}

func (m *AnalysisSectionModel) FocusLast() tea.Cmd {
	m.focusIndex = 8
	return m.incremental.Focus()
}
