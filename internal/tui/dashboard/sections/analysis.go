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

type RunAnalysisMsg struct{}

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
	runButton        components.ButtonModel

	inputs          *components.FocusableSlice
	analysisRunning bool
}

func NewAnalysisSection() *AnalysisSectionModel {
	m := &AnalysisSectionModel{
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
		runButton: components.NewButton(
			"▶ Run Analysis",
			func() tea.Msg { return RunAnalysisMsg{} },
			components.WithButtonHelp("Start codebase analysis with current settings"),
		),
	}

	m.inputs = components.NewFocusableSlice(
		components.WrapToggle(&m.excludeStructure),
		components.WrapToggle(&m.excludeDataFlow),
		components.WrapToggle(&m.excludeDeps),
		components.WrapToggle(&m.excludeReqFlow),
		components.WrapToggle(&m.excludeAPI),
		components.WrapTextField(&m.maxWorkers),
		components.WrapTextField(&m.maxHashWorkers),
		components.WrapToggle(&m.force),
		components.WrapToggle(&m.incremental),
		components.WrapButton(&m.runButton),
	)

	return m
}

func (m *AnalysisSectionModel) Title() string       { return "Analysis Settings" }
func (m *AnalysisSectionModel) Icon() string        { return "🔍" }
func (m *AnalysisSectionModel) Description() string { return "Configure codebase analysis options" }

func (m *AnalysisSectionModel) Init() tea.Cmd { return nil }

func (m *AnalysisSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AnalysisStartedMsg:
		m.analysisRunning = true
		m.runButton.SetLoading(true)
		return m, nil
	case AnalysisStoppedMsg:
		m.analysisRunning = false
		m.runButton.SetLoading(false)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return m, m.inputs.FocusNext()
		case "shift+tab":
			return m, m.inputs.FocusPrev()
		}
	}

	return m, m.inputs.UpdateCurrent(msg)
}

type AnalysisStartedMsg struct{}
type AnalysisStoppedMsg struct{}

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
		"",
		m.runButton.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *AnalysisSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *AnalysisSectionModel) IsDirty() bool {
	return m.inputs.IsDirty()
}

func (m *AnalysisSectionModel) GetValues() map[string]any {
	values := map[string]any{
		KeyExcludeCodeStructure: m.excludeStructure.Value(),
		KeyExcludeDataFlow:      m.excludeDataFlow.Value(),
		KeyExcludeDependencies:  m.excludeDeps.Value(),
		KeyExcludeRequestFlow:   m.excludeReqFlow.Value(),
		KeyExcludeAPIAnalysis:   m.excludeAPI.Value(),
		KeyForce:                m.force.Value(),
		KeyIncremental:          m.incremental.Value(),
	}
	if v := m.maxWorkers.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[KeyMaxWorkers] = i
		}
	}
	if v := m.maxHashWorkers.Value(); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			values[KeyMaxHashWorkers] = i
		}
	}
	return values
}

func (m *AnalysisSectionModel) SetValues(values map[string]any) error {
	if v, ok := values[KeyExcludeCodeStructure].(bool); ok {
		m.excludeStructure.SetValue(v)
	}
	if v, ok := values[KeyExcludeDataFlow].(bool); ok {
		m.excludeDataFlow.SetValue(v)
	}
	if v, ok := values[KeyExcludeDependencies].(bool); ok {
		m.excludeDeps.SetValue(v)
	}
	if v, ok := values[KeyExcludeRequestFlow].(bool); ok {
		m.excludeReqFlow.SetValue(v)
	}
	if v, ok := values[KeyExcludeAPIAnalysis].(bool); ok {
		m.excludeAPI.SetValue(v)
	}
	if v, ok := values[KeyMaxWorkers].(int); ok {
		m.maxWorkers.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[KeyMaxHashWorkers].(int); ok {
		m.maxHashWorkers.SetValue(strconv.Itoa(v))
	}
	if v, ok := values[KeyForce].(bool); ok {
		m.force.SetValue(v)
	}
	if v, ok := values[KeyIncremental].(bool); ok {
		m.incremental.SetValue(v)
	}
	return nil
}

func (m *AnalysisSectionModel) FocusFirst() tea.Cmd {
	return m.inputs.FocusFirst()
}

func (m *AnalysisSectionModel) FocusLast() tea.Cmd {
	return m.inputs.FocusLast()
}
