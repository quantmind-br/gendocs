package dashboard

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProgressView_NewProgressView(t *testing.T) {
	pv := NewProgressView()

	if pv.Visible() {
		t.Error("new progress view should not be visible")
	}
	if len(pv.tasks) != 0 {
		t.Error("new progress view should have no tasks")
	}
	if pv.IsComplete() {
		t.Error("new progress view should not be complete")
	}
}

func TestProgressView_Show(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	if !pv.Visible() {
		t.Error("progress view should be visible after Show()")
	}
	if pv.startTime.IsZero() {
		t.Error("start time should be set")
	}
	if pv.completed {
		t.Error("should not be completed after Show()")
	}
	if pv.cancelled {
		t.Error("should not be cancelled after Show()")
	}
}

func TestProgressView_Hide(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.Hide()

	if pv.Visible() {
		t.Error("progress view should not be visible after Hide()")
	}
}

func TestProgressView_AddTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Test Task", "Description")

	if len(pv.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(pv.tasks))
	}

	task := pv.tasks[0]
	if task.ID != "task-1" {
		t.Errorf("expected ID 'task-1', got '%s'", task.ID)
	}
	if task.Name != "Test Task" {
		t.Errorf("expected name 'Test Task', got '%s'", task.Name)
	}
	if task.Status != StatusPending {
		t.Errorf("expected status Pending, got %v", task.Status)
	}
}

func TestProgressView_StartTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Test Task", "Description")
	pv.StartTask("task-1")

	task := pv.taskMap["task-1"]
	if task.Status != StatusRunning {
		t.Errorf("expected status Running, got %v", task.Status)
	}
	if task.StartTime.IsZero() {
		t.Error("start time should be set")
	}
}

func TestProgressView_CompleteTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Test Task", "Description")
	pv.StartTask("task-1")
	time.Sleep(10 * time.Millisecond)
	pv.CompleteTask("task-1")

	task := pv.taskMap["task-1"]
	if task.Status != StatusSuccess {
		t.Errorf("expected status Success, got %v", task.Status)
	}
	if task.EndTime.IsZero() {
		t.Error("end time should be set")
	}
	if task.EndTime.Before(task.StartTime) {
		t.Error("end time should be after start time")
	}
}

func TestProgressView_FailTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Test Task", "Description")
	pv.StartTask("task-1")

	testErr := errors.New("test error")
	pv.FailTask("task-1", testErr)

	task := pv.taskMap["task-1"]
	if task.Status != StatusFailed {
		t.Errorf("expected status Failed, got %v", task.Status)
	}
	if task.Error != testErr {
		t.Error("error should be preserved")
	}
}

func TestProgressView_SkipTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Test Task", "Description")
	pv.SkipTask("task-1")

	task := pv.taskMap["task-1"]
	if task.Status != StatusSkipped {
		t.Errorf("expected status Skipped, got %v", task.Status)
	}
}

func TestProgressView_SetCompleted(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	summary := AnalysisSummary{
		Successful: 3,
		Failed:     1,
		Skipped:    1,
		Duration:   5 * time.Second,
	}
	pv.SetCompleted(summary)

	if !pv.completed {
		t.Error("progress view should be marked completed")
	}
	if !pv.IsComplete() {
		t.Error("IsComplete should return true")
	}
	if pv.summary == nil {
		t.Error("summary should be set")
	}
	if pv.summary.Successful != 3 {
		t.Errorf("expected 3 successful, got %d", pv.summary.Successful)
	}
}

func TestProgressView_SetCancelled(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.SetCancelled()

	if !pv.cancelled {
		t.Error("progress view should be marked cancelled")
	}
	if !pv.IsComplete() {
		t.Error("IsComplete should return true when cancelled")
	}
}

func TestProgressView_SetError(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	testErr := errors.New("fatal error")
	pv.SetError(testErr)

	if pv.fatalError != testErr {
		t.Error("fatal error should be set")
	}
	if !pv.IsComplete() {
		t.Error("IsComplete should return true when error set")
	}
}

func TestProgressView_View_NotVisible(t *testing.T) {
	pv := NewProgressView()
	view := pv.View()

	if view != "" {
		t.Error("view should be empty when not visible")
	}
}

func TestProgressView_View_WithTasks(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.AddTask("task-1", "Task One", "")
	pv.AddTask("task-2", "Task Two", "")
	pv.StartTask("task-1")

	view := pv.View()

	if !strings.Contains(view, "Analysis in Progress") {
		t.Error("view should contain progress header")
	}
	if !strings.Contains(view, "Task One") {
		t.Error("view should contain task names")
	}
	if !strings.Contains(view, "Task Two") {
		t.Error("view should contain task names")
	}
	if !strings.Contains(view, "Esc") {
		t.Error("view should contain cancel instruction")
	}
}

func TestProgressView_View_Completed(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.SetCompleted(AnalysisSummary{Successful: 2, Duration: time.Second})

	view := pv.View()

	if !strings.Contains(view, "Complete") {
		t.Error("view should indicate completion")
	}
	if !strings.Contains(view, "Enter") {
		t.Error("view should contain close instruction")
	}
}

func TestProgressView_View_Cancelled(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.SetCancelled()

	view := pv.View()

	if !strings.Contains(view, "Cancelled") {
		t.Error("view should indicate cancellation")
	}
}

func TestProgressView_View_WithError(t *testing.T) {
	pv := NewProgressView()
	pv.Show()
	pv.SetError(errors.New("test error"))

	view := pv.View()

	if !strings.Contains(view, "Failed") {
		t.Error("view should indicate failure")
	}
	if !strings.Contains(view, "test error") {
		t.Error("view should contain error message")
	}
}

func TestProgressView_NonexistentTask(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	pv.StartTask("nonexistent")
	pv.CompleteTask("nonexistent")
	pv.FailTask("nonexistent", errors.New("error"))
	pv.SkipTask("nonexistent")

	if len(pv.tasks) != 0 {
		t.Error("no tasks should be added for nonexistent IDs")
	}
}

func TestProgressView_Update_Tick(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	initialFrame := pv.spinnerFrame
	pv.Update(TickMsg(time.Now()))

	if pv.spinnerFrame == initialFrame {
		t.Error("spinner frame should change on tick")
	}
}

func TestProgressView_FormatSummary(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	pv.summary = &AnalysisSummary{
		Successful: 3,
		Failed:     1,
		Skipped:    0,
		Duration:   5 * time.Second,
	}

	summary := pv.formatSummary()

	if !strings.Contains(summary, "Completed: 3") {
		t.Error("summary should contain successful count")
	}
	if !strings.Contains(summary, "Failed: 1") {
		t.Error("summary should contain failed count")
	}
}

func TestProgressView_MultipleTasks(t *testing.T) {
	pv := NewProgressView()
	pv.Show()

	pv.AddTask("t1", "Structure", "")
	pv.AddTask("t2", "Data Flow", "")
	pv.AddTask("t3", "Dependencies", "")
	pv.AddTask("t4", "API", "")

	pv.StartTask("t1")
	pv.CompleteTask("t1")
	pv.StartTask("t2")
	pv.FailTask("t2", errors.New("rate limit"))
	pv.SkipTask("t3")
	pv.StartTask("t4")
	pv.CompleteTask("t4")

	view := pv.View()

	if !strings.Contains(view, "Structure") {
		t.Error("view should contain Structure")
	}

	task := pv.taskMap["t2"]
	if task.Status != StatusFailed {
		t.Error("task t2 should be failed")
	}
	if task.Error == nil || task.Error.Error() != "rate limit" {
		t.Error("task t2 should have error 'rate limit'")
	}
}
