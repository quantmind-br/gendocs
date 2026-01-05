package dashboard

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/agents"
)

var _ agents.ProgressReporter = (*TUIProgressReporter)(nil)

type TUIProgressReporter struct {
	program *tea.Program
	mu      sync.Mutex
	tasks   map[string]taskInfo
}

type taskInfo struct {
	ID          string
	Name        string
	Description string
}

func NewTUIProgressReporter(program *tea.Program) *TUIProgressReporter {
	return &TUIProgressReporter{
		program: program,
		tasks:   make(map[string]taskInfo),
	}
}

func (r *TUIProgressReporter) AddTask(id, name, description string) {
	r.mu.Lock()
	r.tasks[id] = taskInfo{ID: id, Name: name, Description: description}
	r.mu.Unlock()

	r.send(AnalysisProgressMsg{
		TaskID:      id,
		TaskName:    name,
		Description: description,
		Event:       ProgressEventTaskAdded,
	})
}

func (r *TUIProgressReporter) StartTask(id string) {
	r.send(AnalysisProgressMsg{
		TaskID: id,
		Event:  ProgressEventTaskStarted,
	})
}

func (r *TUIProgressReporter) CompleteTask(id string) {
	r.send(AnalysisProgressMsg{
		TaskID: id,
		Event:  ProgressEventTaskCompleted,
	})
}

func (r *TUIProgressReporter) FailTask(id string, err error) {
	r.send(AnalysisProgressMsg{
		TaskID: id,
		Event:  ProgressEventTaskFailed,
		Error:  err,
	})
}

func (r *TUIProgressReporter) SkipTask(id string) {
	r.send(AnalysisProgressMsg{
		TaskID: id,
		Event:  ProgressEventTaskSkipped,
	})
}

func (r *TUIProgressReporter) send(msg tea.Msg) {
	if r.program != nil {
		r.program.Send(msg)
	}
}
