package sections

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/types"
	"github.com/user/gendocs/internal/tui/dashboard/validation"
)

type LoggingSectionModel struct {
	logDir       components.TextFieldModel
	fileLevel    components.DropdownModel
	consoleLevel components.DropdownModel

	focusIndex int
}

func NewLoggingSection() *LoggingSectionModel {
	levelOpts := []components.DropdownOption{
		{Value: "debug", Label: "Debug"},
		{Value: "info", Label: "Info"},
		{Value: "warn", Label: "Warning"},
		{Value: "error", Label: "Error"},
	}

	return &LoggingSectionModel{
		logDir: components.NewTextField("Log Directory",
			components.WithPlaceholder(".ai/logs"),
			components.WithValidator(validation.ValidatePath()),
			components.WithHelp("Directory for log files")),
		fileLevel:    components.NewDropdown("File Log Level", levelOpts, "Minimum level for file logs"),
		consoleLevel: components.NewDropdown("Console Log Level", levelOpts, "Minimum level for console output"),
	}
}

func (m *LoggingSectionModel) Title() string       { return "Logging Configuration" }
func (m *LoggingSectionModel) Icon() string        { return "📋" }
func (m *LoggingSectionModel) Description() string { return "Configure logging behavior" }

func (m *LoggingSectionModel) Init() tea.Cmd { return nil }

func (m *LoggingSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.logDir, _ = m.logDir.Update(msg)
	case 1:
		m.fileLevel, _ = m.fileLevel.Update(msg)
	case 2:
		m.consoleLevel, _ = m.consoleLevel.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *LoggingSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.logDir.Blur()
	case 1:
		m.fileLevel.Blur()
	case 2:
		m.consoleLevel.Blur()
	}
}

func (m *LoggingSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.logDir.Focus()
	case 1:
		return m.fileLevel.Focus()
	case 2:
		return m.consoleLevel.Focus()
	}
	return nil
}

func (m *LoggingSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.logDir.View(),
		"",
		m.fileLevel.View(),
		"",
		m.consoleLevel.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *LoggingSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *LoggingSectionModel) IsDirty() bool {
	return m.logDir.IsDirty() || m.fileLevel.IsDirty() || m.consoleLevel.IsDirty()
}

func (m *LoggingSectionModel) GetValues() map[string]any {
	return map[string]any{
		KeyLogDir:       m.logDir.Value(),
		KeyFileLevel:    m.fileLevel.Value(),
		KeyConsoleLevel: m.consoleLevel.Value(),
	}
}

func (m *LoggingSectionModel) SetValues(values map[string]any) error {
	if v, ok := values[KeyLogDir].(string); ok {
		m.logDir.SetValue(v)
	}
	if v, ok := values[KeyFileLevel].(string); ok {
		m.fileLevel.SetValue(v)
	}
	if v, ok := values[KeyConsoleLevel].(string); ok {
		m.consoleLevel.SetValue(v)
	}
	return nil
}

func (m *LoggingSectionModel) FocusFirst() tea.Cmd {
	m.blurAll()
	m.focusIndex = 0
	return m.logDir.Focus()
}

func (m *LoggingSectionModel) FocusLast() tea.Cmd {
	m.blurAll()
	m.focusIndex = 2
	return m.consoleLevel.Focus()
}

func (m *LoggingSectionModel) blurAll() {
	m.logDir.Blur()
	m.fileLevel.Blur()
	m.consoleLevel.Blur()
}
