package tui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/user/gendocs/internal/agents"
)

// captureOutput captures stdout during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// captureOutputWithWriter captures output from a progress reporter using SetWriter
func captureOutputWithWriter(f func(*bytes.Buffer)) string {
	var buf bytes.Buffer
	f(&buf)
	return buf.String()
}

// ============================================================================
// SimpleProgress Tests
// ============================================================================

// TestNewSimpleProgress tests creation of a new SimpleProgress instance
func TestNewSimpleProgress(t *testing.T) {
	progress := NewSimpleProgress("Test Title")

	if progress == nil {
		t.Fatal("Expected SimpleProgress instance, got nil")
	}
	if progress.title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", progress.title)
	}
	if progress.started {
		t.Error("Expected started to be false initially")
	}
}

// TestNewSimpleProgress_EmptyTitle tests creation with empty title
func TestNewSimpleProgress_EmptyTitle(t *testing.T) {
	progress := NewSimpleProgress("")

	if progress == nil {
		t.Fatal("Expected SimpleProgress instance, got nil")
	}
	if progress.title != "" {
		t.Errorf("Expected empty title, got '%s'", progress.title)
	}
}

// TestSimpleProgress_Start tests that Start sets the started flag
func TestSimpleProgress_Start(t *testing.T) {
	progress := NewSimpleProgress("Test")

	_ = captureOutput(func() {
		progress.Start()
	})

	if !progress.started {
		t.Error("Expected started to be true after Start()")
	}
}

// TestSimpleProgress_StartIdempotent tests that Start is idempotent
func TestSimpleProgress_StartIdempotent(t *testing.T) {
	progress := NewSimpleProgress("Test")

	// Call Start multiple times
	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()
		progress.Start()
		progress.Start()
	})

	// Title should only appear once
	count := strings.Count(output, "Test")
	if count != 1 {
		t.Errorf("Expected title to appear once, but appeared %d times", count)
	}
}

// TestSimpleProgress_StartPrintsTitle tests that Start prints the title
func TestSimpleProgress_StartPrintsTitle(t *testing.T) {
	progress := NewSimpleProgress("My Progress Title")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()
	})

	if !strings.Contains(output, "My Progress Title") {
		t.Errorf("Expected output to contain title 'My Progress Title', got: %s", output)
	}
}

// TestSimpleProgress_Step tests Step method
func TestSimpleProgress_Step(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Step("Doing something")
	})

	if !strings.Contains(output, "Doing something") {
		t.Errorf("Expected output to contain 'Doing something', got: %s", output)
	}
}

// TestSimpleProgress_Info tests Info method
func TestSimpleProgress_Info(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Info("Some information")
	})

	if !strings.Contains(output, "Some information") {
		t.Errorf("Expected output to contain 'Some information', got: %s", output)
	}
}

// TestSimpleProgress_Success tests Success method
func TestSimpleProgress_Success(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Success("Operation succeeded")
	})

	if !strings.Contains(output, "Operation succeeded") {
		t.Errorf("Expected output to contain 'Operation succeeded', got: %s", output)
	}
}

// TestSimpleProgress_Error tests Error method
func TestSimpleProgress_Error(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Error("Something went wrong")
	})

	if !strings.Contains(output, "Something went wrong") {
		t.Errorf("Expected output to contain 'Something went wrong', got: %s", output)
	}
}

// TestSimpleProgress_Warning tests Warning method
func TestSimpleProgress_Warning(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Warning("Be careful")
	})

	if !strings.Contains(output, "Be careful") {
		t.Errorf("Expected output to contain 'Be careful', got: %s", output)
	}
}

// TestSimpleProgress_FailedWithError tests Failed with an error
func TestSimpleProgress_FailedWithError(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Failed(errors.New("something failed"))
	})

	if !strings.Contains(output, "Failed") {
		t.Errorf("Expected output to contain 'Failed', got: %s", output)
	}
	if !strings.Contains(output, "something failed") {
		t.Errorf("Expected output to contain error message, got: %s", output)
	}
}

// TestSimpleProgress_FailedWithNilError tests Failed with nil error
func TestSimpleProgress_FailedWithNilError(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Failed(nil)
	})

	if !strings.Contains(output, "Failed") {
		t.Errorf("Expected output to contain 'Failed', got: %s", output)
	}
}

// TestSimpleProgress_Done tests Done method
func TestSimpleProgress_Done(t *testing.T) {
	progress := NewSimpleProgress("Test")

	// Done should complete without error
	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()
		progress.Done()
	})

	// Just verify it doesn't panic and produces some output
	if len(output) == 0 {
		t.Error("Expected some output from Start and Done")
	}
}

// TestSimpleProgress_FullWorkflow tests a complete workflow
func TestSimpleProgress_FullWorkflow(t *testing.T) {
	progress := NewSimpleProgress("Build Project")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()
		progress.Step("Compiling source files")
		progress.Info("Found 42 files")
		progress.Success("Compilation complete")
		progress.Warning("Some deprecation warnings")
		progress.Done()
	})

	expectedParts := []string{
		"Build Project",
		"Compiling source files",
		"Found 42 files",
		"Compilation complete",
		"Some deprecation warnings",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', got: %s", part, output)
		}
	}
}

// ============================================================================
// Progress Tests
// ============================================================================

// TestNewProgress tests creation of a new Progress instance
func TestNewProgress(t *testing.T) {
	progress := NewProgress("Test Title")

	if progress == nil {
		t.Fatal("Expected Progress instance, got nil")
	}
	if progress.title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", progress.title)
	}
	if progress.started {
		t.Error("Expected started to be false initially")
	}
	if len(progress.tasks) != 0 {
		t.Errorf("Expected empty tasks slice, got %d tasks", len(progress.tasks))
	}
	if len(progress.taskMap) != 0 {
		t.Errorf("Expected empty taskMap, got %d entries", len(progress.taskMap))
	}
}

// TestProgress_Start tests that Start sets the started flag
func TestProgress_Start(t *testing.T) {
	progress := NewProgress("Test")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()
	})

	if !progress.started {
		t.Error("Expected started to be true after Start()")
	}
}

// TestProgress_StartIdempotent tests that Start is idempotent
func TestProgress_StartIdempotent(t *testing.T) {
	progress := NewProgress("Test")

	output := captureOutput(func() {
		progress.Start()
		progress.Start()
		progress.Start()
	})

	// Title should only appear once
	count := strings.Count(output, "Test")
	if count != 1 {
		t.Errorf("Expected title to appear once, but appeared %d times", count)
	}
}

// TestProgress_Stop tests that Stop can be called without panic
func TestProgress_Stop(t *testing.T) {
	progress := NewProgress("Test")
	// Stop should not panic
	progress.Stop()
}

// TestProgress_StopIdempotent tests that Stop is idempotent
func TestProgress_StopIdempotent(t *testing.T) {
	progress := NewProgress("Test")
	// Multiple stops should not panic
	progress.Stop()
	progress.Stop()
	progress.Stop()
}

// TestProgress_AddTask tests adding tasks
func TestProgress_AddTask(t *testing.T) {
	progress := NewProgress("Test")

	progress.AddTask("task-1", "First Task", "Description 1")
	progress.AddTask("task-2", "Second Task", "Description 2")

	if len(progress.tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(progress.tasks))
	}
	if len(progress.taskMap) != 2 {
		t.Errorf("Expected 2 entries in taskMap, got %d", len(progress.taskMap))
	}

	// Check first task
	task1 := progress.taskMap["task-1"]
	if task1 == nil {
		t.Fatal("Expected task-1 to exist in taskMap")
	}
	if task1.ID != "task-1" {
		t.Errorf("Expected ID 'task-1', got '%s'", task1.ID)
	}
	if task1.Name != "First Task" {
		t.Errorf("Expected Name 'First Task', got '%s'", task1.Name)
	}
	if task1.Description != "Description 1" {
		t.Errorf("Expected Description 'Description 1', got '%s'", task1.Description)
	}
	if task1.Status != TaskPending {
		t.Errorf("Expected status TaskPending, got %v", task1.Status)
	}

	// Check second task
	task2 := progress.taskMap["task-2"]
	if task2 == nil {
		t.Fatal("Expected task-2 to exist in taskMap")
	}
	if task2.Name != "Second Task" {
		t.Errorf("Expected Name 'Second Task', got '%s'", task2.Name)
	}
}

// TestProgress_StartTask tests starting a task
func TestProgress_StartTask(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "First Task", "Description")

	_ = captureOutput(func() {
		progress.StartTask("task-1")
	})

	task := progress.taskMap["task-1"]
	if task.Status != TaskRunning {
		t.Errorf("Expected status TaskRunning, got %v", task.Status)
	}
}

// TestProgress_StartTask_NonExistent tests starting a non-existent task
func TestProgress_StartTask_NonExistent(t *testing.T) {
	progress := NewProgress("Test")

	// Should not panic
	progress.StartTask("non-existent")

	// Nothing should be in taskMap
	if len(progress.taskMap) != 0 {
		t.Errorf("Expected empty taskMap, got %d entries", len(progress.taskMap))
	}
}

// TestProgress_CompleteTask tests completing a task
func TestProgress_CompleteTask(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "First Task", "Description")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.StartTask("task-1")
		progress.CompleteTask("task-1")
	})

	task := progress.taskMap["task-1"]
	if task.Status != TaskSuccess {
		t.Errorf("Expected status TaskSuccess, got %v", task.Status)
	}
}

// TestProgress_CompleteTask_NonExistent tests completing a non-existent task
func TestProgress_CompleteTask_NonExistent(t *testing.T) {
	progress := NewProgress("Test")

	// Should not panic
	progress.CompleteTask("non-existent")
}

// TestProgress_FailTask tests failing a task with error
func TestProgress_FailTask(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "First Task", "Description")
	testErr := errors.New("task failed")

	_ = captureOutput(func() {
		progress.StartTask("task-1")
		progress.FailTask("task-1", testErr)
	})

	task := progress.taskMap["task-1"]
	if task.Status != TaskError {
		t.Errorf("Expected status TaskError, got %v", task.Status)
	}
	if task.Error != testErr {
		t.Errorf("Expected error to be set, got %v", task.Error)
	}
}

// TestProgress_FailTask_NilError tests failing a task with nil error
func TestProgress_FailTask_NilError(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "First Task", "Description")

	_ = captureOutput(func() {
		progress.StartTask("task-1")
		progress.FailTask("task-1", nil)
	})

	task := progress.taskMap["task-1"]
	if task.Status != TaskError {
		t.Errorf("Expected status TaskError, got %v", task.Status)
	}
	if task.Error != nil {
		t.Errorf("Expected nil error, got %v", task.Error)
	}
}

// TestProgress_FailTask_NonExistent tests failing a non-existent task
func TestProgress_FailTask_NonExistent(t *testing.T) {
	progress := NewProgress("Test")

	// Should not panic
	progress.FailTask("non-existent", errors.New("some error"))
}

// TestProgress_SkipTask tests skipping a task
func TestProgress_SkipTask(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "First Task", "Description")

	_ = captureOutput(func() {
		progress.SkipTask("task-1")
	})

	task := progress.taskMap["task-1"]
	if task.Status != TaskSkipped {
		t.Errorf("Expected status TaskSkipped, got %v", task.Status)
	}
}

// TestProgress_SkipTask_NonExistent tests skipping a non-existent task
func TestProgress_SkipTask_NonExistent(t *testing.T) {
	progress := NewProgress("Test")

	// Should not panic
	progress.SkipTask("non-existent")
}

// TestProgress_PrintSummary_NoTasks tests summary with no tasks
func TestProgress_PrintSummary_NoTasks(t *testing.T) {
	progress := NewProgress("Test")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.PrintSummary()
	})

	// With no tasks, should show "Completed: 0/0"
	if !strings.Contains(output, "Completed: 0/0") {
		t.Errorf("Expected 'Completed: 0/0' message, got: %s", output)
	}
}

// TestProgress_PrintSummary_AllCompleted tests summary with all tasks completed
func TestProgress_PrintSummary_AllCompleted(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "Task 1", "Desc")
	progress.AddTask("task-2", "Task 2", "Desc")
	progress.AddTask("task-3", "Task 3", "Desc")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.StartTask("task-1")
		progress.CompleteTask("task-1")
		progress.StartTask("task-2")
		progress.CompleteTask("task-2")
		progress.StartTask("task-3")
		progress.CompleteTask("task-3")
	})

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.PrintSummary()
	})

	if !strings.Contains(output, "Completed: 3/3") {
		t.Errorf("Expected 'Completed: 3/3' in summary, got: %s", output)
	}
}

// TestProgress_PrintSummary_SomeFailed tests summary with some failures
func TestProgress_PrintSummary_SomeFailed(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "Task 1", "Desc")
	progress.AddTask("task-2", "Task 2", "Desc")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.StartTask("task-1")
		progress.CompleteTask("task-1")
		progress.StartTask("task-2")
		progress.FailTask("task-2", errors.New("oops"))
	})

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.PrintSummary()
	})

	if !strings.Contains(output, "Failed: 1") {
		t.Errorf("Expected 'Failed: 1' in summary, got: %s", output)
	}
	if !strings.Contains(output, "Failed tasks") {
		t.Errorf("Expected 'Failed tasks' section, got: %s", output)
	}
}

// TestProgress_PrintSummary_SomeSkipped tests summary with some skipped
func TestProgress_PrintSummary_SomeSkipped(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "Task 1", "Desc")
	progress.AddTask("task-2", "Task 2", "Desc")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.StartTask("task-1")
		progress.CompleteTask("task-1")
		progress.SkipTask("task-2")
	})

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.PrintSummary()
	})

	if !strings.Contains(output, "Skipped: 1") {
		t.Errorf("Expected 'Skipped: 1' in summary, got: %s", output)
	}
}

// TestProgress_PrintSummary_Mixed tests summary with mixed statuses
func TestProgress_PrintSummary_Mixed(t *testing.T) {
	progress := NewProgress("Test")
	progress.AddTask("task-1", "Task 1", "Desc")
	progress.AddTask("task-2", "Task 2", "Desc")
	progress.AddTask("task-3", "Task 3", "Desc")
	progress.AddTask("task-4", "Task 4", "Desc")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.StartTask("task-1")
		progress.CompleteTask("task-1")
		progress.StartTask("task-2")
		progress.FailTask("task-2", errors.New("error 2"))
		progress.SkipTask("task-3")
		progress.StartTask("task-4")
		progress.CompleteTask("task-4")
	})

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.PrintSummary()
	})

	// Should show completed count (excluding skipped from total)
	if !strings.Contains(output, "Completed: 2/3") {
		t.Errorf("Expected 'Completed: 2/3' in summary (2 complete, 1 failed, 1 skipped), got: %s", output)
	}
	if !strings.Contains(output, "Failed: 1") {
		t.Errorf("Expected 'Failed: 1' in summary, got: %s", output)
	}
	if !strings.Contains(output, "Skipped: 1") {
		t.Errorf("Expected 'Skipped: 1' in summary, got: %s", output)
	}
}

// TestProgress_FullWorkflow tests a complete workflow
func TestProgress_FullWorkflow(t *testing.T) {
	progress := NewProgress("Analysis")

	output := captureOutputWithWriter(func(buf *bytes.Buffer) {
		progress.SetWriter(buf)
		progress.Start()

		progress.AddTask("analyze-deps", "Analyze Dependencies", "Check project dependencies")
		progress.AddTask("analyze-code", "Analyze Code", "Static code analysis")
		progress.AddTask("analyze-api", "Analyze API", "API documentation")

		progress.StartTask("analyze-deps")
		progress.CompleteTask("analyze-deps")

		progress.StartTask("analyze-code")
		progress.CompleteTask("analyze-code")

		progress.StartTask("analyze-api")
		progress.CompleteTask("analyze-api")

		progress.PrintSummary()
		progress.Stop()
	})

	expectedParts := []string{
		"Analysis",
		"Analyze Dependencies",
		"Analyze Code",
		"Analyze API",
		"Completed: 3/3",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', got: %s", part, output)
		}
	}
}

// ============================================================================
// TaskStatus Tests
// ============================================================================

// TestTaskStatus_Values tests TaskStatus constant values
func TestTaskStatus_Values(t *testing.T) {
	// Verify the iota values are as expected
	if TaskPending != 0 {
		t.Errorf("Expected TaskPending = 0, got %d", TaskPending)
	}
	if TaskRunning != 1 {
		t.Errorf("Expected TaskRunning = 1, got %d", TaskRunning)
	}
	if TaskSuccess != 2 {
		t.Errorf("Expected TaskSuccess = 2, got %d", TaskSuccess)
	}
	if TaskError != 3 {
		t.Errorf("Expected TaskError = 3, got %d", TaskError)
	}
	if TaskSkipped != 4 {
		t.Errorf("Expected TaskSkipped = 4, got %d", TaskSkipped)
	}
}

// ============================================================================
// Task Tests
// ============================================================================

// TestTask_Fields tests Task struct fields
func TestTask_Fields(t *testing.T) {
	task := Task{
		ID:          "test-id",
		Name:        "Test Name",
		Description: "Test Description",
		Status:      TaskRunning,
		Error:       errors.New("test error"),
	}

	if task.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", task.ID)
	}
	if task.Name != "Test Name" {
		t.Errorf("Expected Name 'Test Name', got '%s'", task.Name)
	}
	if task.Description != "Test Description" {
		t.Errorf("Expected Description 'Test Description', got '%s'", task.Description)
	}
	if task.Status != TaskRunning {
		t.Errorf("Expected Status TaskRunning, got %d", task.Status)
	}
	if task.Error == nil || task.Error.Error() != "test error" {
		t.Errorf("Expected Error 'test error', got %v", task.Error)
	}
}

// ============================================================================
// Interface Compliance Tests
// ============================================================================

// TestProgress_ImplementsProgressReporter verifies that Progress implements agents.ProgressReporter
func TestProgress_ImplementsProgressReporter(t *testing.T) {
	var _ agents.ProgressReporter = (*Progress)(nil)
	// If this compiles, the interface is implemented
}

// TestProgress_AsProgressReporter tests using Progress as a ProgressReporter
func TestProgress_AsProgressReporter(t *testing.T) {
	var reporter agents.ProgressReporter = NewProgress("Test")

	// All interface methods should work
	reporter.AddTask("task-1", "Task 1", "Description")

	_ = captureOutputWithWriter(func(buf *bytes.Buffer) {
		reporter.(*Progress).SetWriter(buf)
		reporter.StartTask("task-1")
		reporter.CompleteTask("task-1")
	})

	// Verify the task was processed
	progress := reporter.(*Progress)
	task := progress.taskMap["task-1"]
	if task.Status != TaskSuccess {
		t.Errorf("Expected TaskSuccess, got %v", task.Status)
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

// TestProgress_DuplicateTaskIds tests behavior with duplicate task IDs
func TestProgress_DuplicateTaskIds(t *testing.T) {
	progress := NewProgress("Test")

	progress.AddTask("task-1", "First Task", "First description")
	progress.AddTask("task-1", "Duplicate Task", "Duplicate description")

	// Both should be in the tasks slice
	if len(progress.tasks) != 2 {
		t.Errorf("Expected 2 tasks in slice, got %d", len(progress.tasks))
	}

	// Only the second should be in the map (overwrites first)
	task := progress.taskMap["task-1"]
	if task.Name != "Duplicate Task" {
		t.Errorf("Expected map to contain second task 'Duplicate Task', got '%s'", task.Name)
	}
}

// TestSimpleProgress_MultipleMessages tests multiple consecutive messages
func TestSimpleProgress_MultipleMessages(t *testing.T) {
	progress := NewSimpleProgress("Test")

	output := captureOutput(func() {
		progress.Step("Step 1")
		progress.Step("Step 2")
		progress.Info("Info 1")
		progress.Info("Info 2")
		progress.Success("Success 1")
		progress.Error("Error 1")
		progress.Warning("Warning 1")
	})

	expectedMessages := []string{
		"Step 1", "Step 2",
		"Info 1", "Info 2",
		"Success 1",
		"Error 1",
		"Warning 1",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(output, msg) {
			t.Errorf("Expected output to contain '%s'", msg)
		}
	}
}

// TestProgress_TaskOrder tests that tasks maintain insertion order
func TestProgress_TaskOrder(t *testing.T) {
	progress := NewProgress("Test")

	progress.AddTask("task-a", "Task A", "")
	progress.AddTask("task-b", "Task B", "")
	progress.AddTask("task-c", "Task C", "")

	if len(progress.tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(progress.tasks))
	}

	if progress.tasks[0].ID != "task-a" {
		t.Errorf("Expected first task to be 'task-a', got '%s'", progress.tasks[0].ID)
	}
	if progress.tasks[1].ID != "task-b" {
		t.Errorf("Expected second task to be 'task-b', got '%s'", progress.tasks[1].ID)
	}
	if progress.tasks[2].ID != "task-c" {
		t.Errorf("Expected third task to be 'task-c', got '%s'", progress.tasks[2].ID)
	}
}
