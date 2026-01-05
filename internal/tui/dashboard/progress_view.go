package dashboard

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
)

type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
	StatusSkipped
)

type ProgressTask struct {
	ID          string
	Name        string
	Description string
	Status      TaskStatus
	Error       error
	StartTime   time.Time
	EndTime     time.Time
}

type AnalysisSummary struct {
	Successful int
	Failed     int
	Skipped    int
	Duration   time.Duration
}

type ProgressViewModel struct {
	tasks        []ProgressTask
	taskMap      map[string]*ProgressTask
	spinnerFrame int
	startTime    time.Time
	visible      bool
	completed    bool
	cancelled    bool
	fatalError   error
	summary      *AnalysisSummary
}

func NewProgressView() *ProgressViewModel {
	return &ProgressViewModel{
		tasks:   make([]ProgressTask, 0),
		taskMap: make(map[string]*ProgressTask),
	}
}

func (m *ProgressViewModel) Show() {
	m.visible = true
	m.startTime = time.Now()
	m.completed = false
	m.cancelled = false
	m.fatalError = nil
	m.summary = nil
	m.tasks = make([]ProgressTask, 0)
	m.taskMap = make(map[string]*ProgressTask)
	m.spinnerFrame = 0
}

func (m *ProgressViewModel) Hide() {
	m.visible = false
}

func (m *ProgressViewModel) Visible() bool {
	return m.visible
}

func (m *ProgressViewModel) IsComplete() bool {
	return m.completed || m.cancelled || m.fatalError != nil
}

func (m *ProgressViewModel) AddTask(id, name, description string) {
	task := ProgressTask{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      StatusPending,
	}
	m.tasks = append(m.tasks, task)
	m.taskMap[id] = &m.tasks[len(m.tasks)-1]
}

func (m *ProgressViewModel) StartTask(id string) {
	if task, ok := m.taskMap[id]; ok {
		task.Status = StatusRunning
		task.StartTime = time.Now()
	}
}

func (m *ProgressViewModel) CompleteTask(id string) {
	if task, ok := m.taskMap[id]; ok {
		task.Status = StatusSuccess
		task.EndTime = time.Now()
	}
}

func (m *ProgressViewModel) FailTask(id string, err error) {
	if task, ok := m.taskMap[id]; ok {
		task.Status = StatusFailed
		task.Error = err
		task.EndTime = time.Now()
	}
}

func (m *ProgressViewModel) SkipTask(id string) {
	if task, ok := m.taskMap[id]; ok {
		task.Status = StatusSkipped
	}
}

func (m *ProgressViewModel) SetCompleted(summary AnalysisSummary) {
	m.completed = true
	m.summary = &summary
}

func (m *ProgressViewModel) SetCancelled() {
	m.cancelled = true
}

func (m *ProgressViewModel) SetError(err error) {
	m.fatalError = err
	m.completed = true
}

func (m *ProgressViewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case TickMsg:
		m.spinnerFrame = (m.spinnerFrame + 1) % len(tui.SpinnerFrames)
	}
	return nil
}

func (m *ProgressViewModel) View() string {
	if !m.visible {
		return ""
	}

	var sections []string

	elapsed := time.Since(m.startTime).Round(time.Second)
	headerText := fmt.Sprintf("  %s Analysis in Progress... (%s)", tui.SpinnerFrames[m.spinnerFrame], elapsed)
	if m.completed && m.fatalError == nil {
		headerText = "  " + tui.StyleSuccess.Render("✓") + " Analysis Complete"
	} else if m.cancelled {
		headerText = "  " + tui.StyleWarning.Render("⚠") + " Analysis Cancelled"
	} else if m.fatalError != nil {
		headerText = "  " + tui.StyleError.Render("✗") + " Analysis Failed"
	}
	header := tui.StyleSectionHeader.Render(headerText)
	sections = append(sections, header, "")

	for _, task := range m.tasks {
		line := m.formatTaskLine(&task)
		sections = append(sections, line)
	}

	if m.fatalError != nil {
		sections = append(sections, "")
		errorBox := tui.StyleError.Render(fmt.Sprintf("  Error: %s", m.fatalError.Error()))
		sections = append(sections, errorBox)
	}

	if m.completed && m.summary != nil && m.fatalError == nil {
		sections = append(sections, "", m.formatSummary())
	}

	sections = append(sections, "")
	if !m.IsComplete() {
		sections = append(sections, tui.StyleMuted.Render("  Press Esc to cancel"))
	} else {
		sections = append(sections, tui.StyleMuted.Render("  Press Enter to close"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *ProgressViewModel) formatTaskLine(task *ProgressTask) string {
	var icon string
	var style lipgloss.Style

	switch task.Status {
	case StatusPending:
		icon = "○"
		style = tui.StyleMuted
	case StatusRunning:
		icon = tui.SpinnerFrames[m.spinnerFrame]
		style = lipgloss.NewStyle().Foreground(tui.ColorPrimary)
	case StatusSuccess:
		icon = "✓"
		style = tui.StyleSuccess
	case StatusFailed:
		icon = "✗"
		style = tui.StyleError
	case StatusSkipped:
		icon = "○"
		style = tui.StyleMuted
	}

	name := style.Render(task.Name)

	var suffix string
	if task.Status == StatusSuccess && !task.EndTime.IsZero() {
		duration := task.EndTime.Sub(task.StartTime).Round(time.Millisecond)
		suffix = tui.StyleMuted.Render(fmt.Sprintf(" (%s)", duration))
	} else if task.Status == StatusRunning && !task.StartTime.IsZero() {
		elapsed := time.Since(task.StartTime).Round(time.Second)
		suffix = tui.StyleMuted.Render(fmt.Sprintf(" %s", elapsed))
	} else if task.Status == StatusFailed && task.Error != nil {
		suffix = tui.StyleError.Render(fmt.Sprintf(" - %s", task.Error.Error()))
	} else if task.Status == StatusSkipped {
		suffix = tui.StyleMuted.Render(" (skipped)")
	}

	return fmt.Sprintf("    %s %s%s", style.Render(icon), name, suffix)
}

func (m *ProgressViewModel) formatSummary() string {
	if m.summary == nil {
		return ""
	}

	s := m.summary
	text := fmt.Sprintf("  Completed: %d | Failed: %d | Skipped: %d | Duration: %s",
		s.Successful, s.Failed, s.Skipped, s.Duration.Round(time.Second))

	if s.Failed > 0 {
		return tui.StyleWarning.Render(text)
	}
	return tui.StyleSuccess.Render(text)
}

type TickMsg time.Time

func TickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
