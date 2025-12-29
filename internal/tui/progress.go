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
	spinnerFrame int
	ticker       *time.Ticker
	done         chan struct{}
	started      bool
	startTime    time.Time
}

func NewProgress(title string) *Progress {
	return &Progress{
		writer:    os.Stdout,
		title:     title,
		tasks:     make([]*Task, 0),
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
}

func (p *Progress) SetWriter(w io.Writer) {
	p.writer = w
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
}

func (p *Progress) StartTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.tasks {
		if t.ID == id {
			t.Status = TaskRunning
			t.StartTime = time.Now()
			break
		}
	}
}

func (p *Progress) CompleteTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.tasks {
		if t.ID == id {
			t.Status = TaskSuccess
			t.EndTime = time.Now()
			break
		}
	}
}

func (p *Progress) FailTask(id string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.tasks {
		if t.ID == id {
			t.Status = TaskError
			t.Error = err
			t.EndTime = time.Now()
			break
		}
	}
}

func (p *Progress) SkipTask(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.tasks {
		if t.ID == id {
			t.Status = TaskSkipped
			break
		}
	}
}

func (p *Progress) Start() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, StyleTitle.Render(" "+p.title+" "))
	fmt.Fprintln(p.writer)

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

	fmt.Fprint(p.writer, strings.Repeat("\033[A\033[2K", lineCount))

	for _, task := range p.tasks {
		line := p.formatTaskLine(task)
		fmt.Fprintln(p.writer, line)
	}
}

func (p *Progress) formatTaskLine(task *Task) string {
	var icon, status string
	var style lipgloss.Style

	switch task.Status {
	case TaskPending:
		icon = StyleStatusPending.Render(IconPending)
		status = StyleMuted.Render("waiting")
		style = StyleMuted
	case TaskRunning:
		spinnerIcon := SpinnerFrames[p.spinnerFrame]
		icon = StyleStatusRunning.Render(spinnerIcon)
		elapsed := time.Since(task.StartTime).Round(time.Second)
		status = StyleStatusRunning.Render(fmt.Sprintf("running %s", elapsed))
		style = StyleStatusRunning
	case TaskSuccess:
		icon = StyleStatusSuccess.Render(IconSuccess)
		duration := task.EndTime.Sub(task.StartTime).Round(time.Millisecond)
		status = StyleStatusSuccess.Render(fmt.Sprintf("done %s", duration))
		style = StyleStatusSuccess
	case TaskError:
		icon = StyleStatusError.Render(IconError)
		status = StyleStatusError.Render("failed")
		style = StyleStatusError
	case TaskSkipped:
		icon = StyleMuted.Render(IconArrow)
		status = StyleMuted.Render("skipped")
		style = StyleMuted
	}

	name := style.Render(task.Name)

	return fmt.Sprintf("  %s %s %s", icon, StyleTaskName.Render(name), status)
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

	elapsed := time.Since(p.startTime).Round(time.Second)

	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, strings.Repeat("─", 50))

	if failed == 0 {
		fmt.Fprintf(p.writer, "%s All tasks completed successfully!\n",
			StyleSuccess.Render(IconSuccess))
	} else {
		fmt.Fprintf(p.writer, "%s Completed with errors\n",
			StyleError.Render(IconError))
	}

	stats := []string{}
	if successful > 0 {
		stats = append(stats, StyleSuccess.Render(fmt.Sprintf("%d succeeded", successful)))
	}
	if failed > 0 {
		stats = append(stats, StyleError.Render(fmt.Sprintf("%d failed", failed)))
	}
	if skipped > 0 {
		stats = append(stats, StyleMuted.Render(fmt.Sprintf("%d skipped", skipped)))
	}

	fmt.Fprintf(p.writer, "  %s in %s\n", strings.Join(stats, ", "), StyleMuted.Render(elapsed.String()))

	if failed > 0 {
		fmt.Fprintln(p.writer)
		fmt.Fprintln(p.writer, StyleError.Render("Errors:"))
		for _, t := range p.tasks {
			if t.Status == TaskError && t.Error != nil {
				fmt.Fprintf(p.writer, "  %s %s: %s\n",
					StyleError.Render(IconBullet),
					t.Name,
					StyleMuted.Render(t.Error.Error()))
			}
		}
	}

	fmt.Fprintln(p.writer)
}

type SimpleProgress struct {
	writer    io.Writer
	title     string
	startTime time.Time
}

func NewSimpleProgress(title string) *SimpleProgress {
	return &SimpleProgress{
		writer:    os.Stdout,
		title:     title,
		startTime: time.Now(),
	}
}

func (sp *SimpleProgress) SetWriter(w io.Writer) {
	sp.writer = w
}

func (sp *SimpleProgress) Start() {
	fmt.Fprintln(sp.writer)
	fmt.Fprintln(sp.writer, StyleTitle.Render(" "+sp.title+" "))
	fmt.Fprintln(sp.writer)
}

func (sp *SimpleProgress) Step(message string) {
	fmt.Fprintf(sp.writer, "  %s %s\n",
		StyleStatusRunning.Render(IconArrow),
		message)
}

func (sp *SimpleProgress) Success(message string) {
	fmt.Fprintf(sp.writer, "  %s %s\n",
		StyleSuccess.Render(IconSuccess),
		StyleSuccess.Render(message))
}

func (sp *SimpleProgress) Error(message string) {
	fmt.Fprintf(sp.writer, "  %s %s\n",
		StyleError.Render(IconError),
		StyleError.Render(message))
}

func (sp *SimpleProgress) Warning(message string) {
	fmt.Fprintf(sp.writer, "  %s %s\n",
		StyleWarning.Render(IconWarning),
		message)
}

func (sp *SimpleProgress) Info(message string) {
	fmt.Fprintf(sp.writer, "  %s %s\n",
		StyleInfo.Render(IconInfo),
		StyleMuted.Render(message))
}

func (sp *SimpleProgress) Done() {
	elapsed := time.Since(sp.startTime).Round(time.Second)
	fmt.Fprintln(sp.writer)
	fmt.Fprintf(sp.writer, "%s Completed in %s\n",
		StyleSuccess.Render(IconSuccess),
		StyleMuted.Render(elapsed.String()))
	fmt.Fprintln(sp.writer)
}

func (sp *SimpleProgress) Failed(err error) {
	elapsed := time.Since(sp.startTime).Round(time.Second)
	fmt.Fprintln(sp.writer)
	if err != nil {
		fmt.Fprintf(sp.writer, "%s Failed after %s: %s\n",
			StyleError.Render(IconError),
			StyleMuted.Render(elapsed.String()),
			StyleError.Render(err.Error()))
	} else {
		fmt.Fprintf(sp.writer, "%s Failed after %s\n",
			StyleError.Render(IconError),
			StyleMuted.Render(elapsed.String()))
	}
	fmt.Fprintln(sp.writer)
}
