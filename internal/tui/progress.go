package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Progress styles for the TUI
var (
	progressTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	progressStepStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4"))

	progressInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A0A0A0"))

	progressSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B"))

	progressErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5F87"))

	progressWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFB86C"))
)

type ProgressReporter interface {
	AddTask(id, name, description string)
	StartTask(id string)
	CompleteTask(id string)
	FailTask(id string, err error)
	SkipTask(id string)
}

type NopProgressReporter struct{}

func (n *NopProgressReporter) AddTask(id, name, description string) {}
func (n *NopProgressReporter) StartTask(id string)                  {}
func (n *NopProgressReporter) CompleteTask(id string)               {}
func (n *NopProgressReporter) FailTask(id string, err error)        {}
func (n *NopProgressReporter) SkipTask(id string)                   {}

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskSuccess
	TaskError
	TaskSkipped
)

type Task struct {
	ID          string
	Name        string
	Description string
	Status      TaskStatus
	Error       error
	StartTime   time.Time
	EndTime     time.Time
}

type Progress struct {
	mu           sync.Mutex
	writer       io.Writer
	title        string
	tasks        []*Task
	taskMap      map[string]*Task
	spinnerFrame int
	ticker       *time.Ticker
	done         chan struct{}
	started      bool
	startTime    time.Time
}

func NewProgress(title string) *Progress {
	return &Progress{
		title:     title,
		tasks:     make([]*Task, 0),
		taskMap:   make(map[string]*Task),
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
}

func (p *Progress) SetWriter(w io.Writer) {
	p.writer = w
}

func (p *Progress) getWriter() io.Writer {
	if p.writer == nil {
		return os.Stdout
	}
	return p.writer
}

func (p *Progress) AddTask(id, name, description string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	task := &Task{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      TaskPending,
	}
	p.tasks = append(p.tasks, task)
	p.taskMap[id] = task
}

func (p *Progress) StartTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.taskMap[id]
	if !ok {
		return
	}
	task.Status = TaskRunning
	task.StartTime = time.Now()
	// Note: No direct output here - render() handles all display updates
}

func (p *Progress) CompleteTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.taskMap[id]
	if !ok {
		return
	}
	task.Status = TaskSuccess
	task.EndTime = time.Now()
	// Note: No direct output here - render() handles all display updates
}

func (p *Progress) FailTask(id string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.taskMap[id]
	if !ok {
		return
	}
	task.Status = TaskError
	task.Error = err
	task.EndTime = time.Now()
	// Note: No direct output here - render() handles all display updates
}

func (p *Progress) SkipTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.taskMap[id]
	if !ok {
		return
	}
	task.Status = TaskSkipped
	// Note: No direct output here - render() handles all display updates
}

func (p *Progress) Start() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true

	_, _ = fmt.Fprintln(p.getWriter())
	_, _ = fmt.Fprintln(p.getWriter(), progressTitleStyle.Render(" "+p.title+" "))
	_, _ = fmt.Fprintln(p.getWriter())

	// Initial render to create task lines
	for _, task := range p.tasks {
		line := p.formatTaskLine(task)
		_, _ = fmt.Fprintln(p.getWriter(), line)
	}

	p.mu.Unlock()

	p.ticker = time.NewTicker(100 * time.Millisecond)
	go p.animate()
}

func (p *Progress) animate() {
	for {
		select {
		case <-p.done:
			return
		case <-p.ticker.C:
			p.mu.Lock()
			p.spinnerFrame = (p.spinnerFrame + 1) % len(SpinnerFrames)
			p.render()
			p.mu.Unlock()
		}
	}
}

func (p *Progress) render() {
	lineCount := len(p.tasks)
	if lineCount == 0 {
		return
	}

	_, _ = fmt.Fprint(p.getWriter(), strings.Repeat("\033[A\033[2K", lineCount))

	for _, task := range p.tasks {
		line := p.formatTaskLine(task)
		_, _ = fmt.Fprintln(p.getWriter(), line)
	}
}

func (p *Progress) formatTaskLine(task *Task) string {
	var icon, status string
	var style lipgloss.Style

	switch task.Status {
	case TaskPending:
		icon = progressInfoStyle.Render("○")
		status = progressInfoStyle.Render("waiting")
		style = progressInfoStyle
	case TaskRunning:
		spinnerIcon := SpinnerFrames[p.spinnerFrame]
		icon = progressStepStyle.Render(spinnerIcon)
		elapsed := time.Since(task.StartTime).Round(time.Second)
		status = progressStepStyle.Render(fmt.Sprintf("running %s", elapsed))
		style = progressStepStyle
	case TaskSuccess:
		icon = progressSuccessStyle.Render("✓")
		duration := task.EndTime.Sub(task.StartTime).Round(time.Millisecond)
		status = progressSuccessStyle.Render(fmt.Sprintf("done %s", duration))
		style = progressSuccessStyle
	case TaskError:
		icon = progressErrorStyle.Render("✗")
		status = progressErrorStyle.Render("failed")
		style = progressErrorStyle
	case TaskSkipped:
		icon = progressInfoStyle.Render("○")
		status = progressInfoStyle.Render("skipped")
		style = progressInfoStyle
	}

	name := style.Render(task.Name)

	return fmt.Sprintf("  %s %s %s", icon, name, status)
}

func (p *Progress) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)

	p.render()
}

func (p *Progress) PrintSummary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	var successful, failed, skipped int
	for _, t := range p.tasks {
		switch t.Status {
		case TaskSuccess:
			successful++
		case TaskError:
			failed++
		case TaskSkipped:
			skipped++
		}
	}

	_, _ = fmt.Fprintln(p.getWriter())
	_, _ = fmt.Fprintln(p.getWriter(), strings.Repeat("─", 50))

	total := len(p.tasks)
	summary := fmt.Sprintf("Completed: %d/%d", successful, total-skipped)
	if failed > 0 {
		summary += fmt.Sprintf(", Failed: %d", failed)
	}
	if skipped > 0 {
		summary += fmt.Sprintf(", Skipped: %d", skipped)
	}

	if failed == 0 {
		_, _ = fmt.Fprintf(p.getWriter(), "%s %s\n",
			progressSuccessStyle.Render("✓"),
			progressSuccessStyle.Render(summary))
	} else {
		_, _ = fmt.Fprintf(p.getWriter(), "%s %s\n",
			progressErrorStyle.Render("✗"),
			progressWarningStyle.Render(summary))
	}

	if failed > 0 {
		_, _ = fmt.Fprintln(p.getWriter())
		_, _ = fmt.Fprintln(p.getWriter(), progressErrorStyle.Render("Failed tasks:"))
		for _, t := range p.tasks {
			if t.Status == TaskError && t.Error != nil {
				_, _ = fmt.Fprintf(p.getWriter(), "  %s %s: %s\n",
					progressErrorStyle.Render("✗"),
					t.Name,
					t.Error.Error())
			}
		}
	}

	_, _ = fmt.Fprintln(p.getWriter())
}

type SimpleProgress struct {
	writer  io.Writer
	title   string
	started bool
}

func NewSimpleProgress(title string) *SimpleProgress {
	return &SimpleProgress{
		title: title,
	}
}

func (sp *SimpleProgress) SetWriter(w io.Writer) {
	sp.writer = w
}

func (sp *SimpleProgress) getWriter() io.Writer {
	if sp.writer == nil {
		return os.Stdout
	}
	return sp.writer
}

func (sp *SimpleProgress) Start() {
	if sp.started {
		return
	}
	sp.started = true
	_, _ = fmt.Fprintln(sp.getWriter())
	_, _ = fmt.Fprintln(sp.getWriter(), progressTitleStyle.Render(" "+sp.title+" "))
	_, _ = fmt.Fprintln(sp.getWriter())
}

func (sp *SimpleProgress) Step(message string) {
	_, _ = fmt.Fprintf(sp.getWriter(), "%s %s\n",
		progressStepStyle.Render("●"),
		message)
}

func (sp *SimpleProgress) Success(message string) {
	_, _ = fmt.Fprintf(sp.getWriter(), "%s %s\n",
		progressSuccessStyle.Render("✓"),
		progressSuccessStyle.Render(message))
}

func (sp *SimpleProgress) Error(message string) {
	_, _ = fmt.Fprintf(sp.getWriter(), "%s %s\n",
		progressErrorStyle.Render("✗"),
		progressErrorStyle.Render(message))
}

func (sp *SimpleProgress) Warning(message string) {
	_, _ = fmt.Fprintf(sp.getWriter(), "%s %s\n",
		progressWarningStyle.Render("⚠"),
		message)
}

func (sp *SimpleProgress) Info(message string) {
	_, _ = fmt.Fprintf(sp.getWriter(), "  %s\n",
		progressInfoStyle.Render(message))
}

func (sp *SimpleProgress) Done() {
	_, _ = fmt.Fprintln(sp.getWriter())
}

func (sp *SimpleProgress) Failed(err error) {
	_, _ = fmt.Fprintln(sp.getWriter())
	if err != nil {
		_, _ = fmt.Fprintf(sp.getWriter(), "%s %s\n",
			progressErrorStyle.Render("✗ Failed:"),
			err.Error())
	} else {
		_, _ = fmt.Fprintf(sp.getWriter(), "%s\n",
			progressErrorStyle.Render("✗ Failed"))
	}
}
