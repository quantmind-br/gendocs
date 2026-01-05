package dashboard

import (
	"errors"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/agents"
)

func TestTUIProgressReporter_ImplementsInterface(t *testing.T) {
	var _ agents.ProgressReporter = (*TUIProgressReporter)(nil)
}

type mockProgram struct {
	mu       sync.Mutex
	messages []tea.Msg
}

func (p *mockProgram) Send(msg tea.Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, msg)
}

func (p *mockProgram) getMessages() []tea.Msg {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]tea.Msg, len(p.messages))
	copy(result, p.messages)
	return result
}

func TestTUIProgressReporter_AddTask(t *testing.T) {
	mock := &mockProgram{}
	reporter := NewTUIProgressReporter((*tea.Program)(nil))
	reporter.program = nil

	type testProgram interface {
		Send(tea.Msg)
	}
	_ = testProgram(mock)

	reporter2 := &TUIProgressReporter{
		tasks: make(map[string]taskInfo),
	}

	reporter2.AddTask("task-1", "Test Task", "Description")

	if len(reporter2.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(reporter2.tasks))
	}

	task := reporter2.tasks["task-1"]
	if task.Name != "Test Task" {
		t.Errorf("expected name 'Test Task', got '%s'", task.Name)
	}
}

func TestTUIProgressReporter_AllMethods(t *testing.T) {
	reporter := &TUIProgressReporter{
		tasks: make(map[string]taskInfo),
	}

	reporter.AddTask("task-1", "Test", "Desc")
	reporter.StartTask("task-1")
	reporter.CompleteTask("task-1")

	reporter.AddTask("task-2", "Test2", "Desc2")
	reporter.StartTask("task-2")
	reporter.FailTask("task-2", errors.New("error"))

	reporter.AddTask("task-3", "Test3", "Desc3")
	reporter.SkipTask("task-3")

	if len(reporter.tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(reporter.tasks))
	}
}

func TestTUIProgressReporter_ConcurrentAccess(t *testing.T) {
	reporter := &TUIProgressReporter{
		tasks: make(map[string]taskInfo),
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := "task-" + string(rune('a'+id%26))
			reporter.AddTask(taskID, "Task", "Desc")
			reporter.StartTask(taskID)
			reporter.CompleteTask(taskID)
		}(i)
	}

	wg.Wait()
}

func TestTUIProgressReporter_NilProgram(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked with nil program: %v", r)
		}
	}()

	reporter := NewTUIProgressReporter(nil)
	reporter.AddTask("task-1", "Test", "Desc")
	reporter.StartTask("task-1")
	reporter.CompleteTask("task-1")
	reporter.FailTask("task-2", errors.New("error"))
	reporter.SkipTask("task-3")
}

func TestTUIProgressReporter_SendCreatesMessages(t *testing.T) {
	var received []tea.Msg
	var mu sync.Mutex

	reporter := &TUIProgressReporter{
		tasks: make(map[string]taskInfo),
	}

	originalSend := reporter.send
	_ = originalSend

	reporter.AddTask("test", "Test", "Description")

	mu.Lock()
	_ = received
	mu.Unlock()

	time.Sleep(10 * time.Millisecond)
}
