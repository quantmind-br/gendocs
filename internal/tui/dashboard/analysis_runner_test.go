//go:build integration

package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/tui/dashboard/sections"
)

func TestAnalysisRunner_ProgressMessages(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()

	dashboard.Update(AnalysisProgressMsg{
		TaskID:   "structure",
		TaskName: "Code Structure",
		Event:    ProgressEventTaskAdded,
	})

	if len(dashboard.progressView.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(dashboard.progressView.tasks))
	}

	dashboard.Update(AnalysisProgressMsg{
		TaskID: "structure",
		Event:  ProgressEventTaskStarted,
	})

	task := dashboard.progressView.taskMap["structure"]
	if task.Status != StatusRunning {
		t.Errorf("expected Running status, got %v", task.Status)
	}

	dashboard.Update(AnalysisProgressMsg{
		TaskID: "structure",
		Event:  ProgressEventTaskCompleted,
	})

	if task.Status != StatusSuccess {
		t.Errorf("expected Success status, got %v", task.Status)
	}
}

func TestAnalysisRunner_CompletionSummary(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()
	dashboard.sections["analysis"] = sections.NewAnalysisSection()

	dashboard.Update(AnalysisCompleteMsg{
		Successful: []string{"structure", "deps"},
		Failed:     []FailedAnalysis{{Name: "api", Error: errors.New("failed")}},
		Duration:   10 * time.Second,
	})

	if dashboard.progressView.summary == nil {
		t.Fatal("summary should be set")
	}
	if dashboard.progressView.summary.Successful != 2 {
		t.Errorf("expected 2 successful, got %d", dashboard.progressView.summary.Successful)
	}
	if dashboard.progressView.summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", dashboard.progressView.summary.Failed)
	}
}

func TestAnalysisRunner_ErrorHandling(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()
	dashboard.sections["analysis"] = sections.NewAnalysisSection()

	testErr := errors.New("test error")
	dashboard.Update(AnalysisErrorMsg{Error: testErr})

	if dashboard.progressView.fatalError != testErr {
		t.Error("fatal error should be set")
	}
	if !dashboard.progressView.IsComplete() {
		t.Error("progress view should be complete after error")
	}
}

func TestAnalysisRunner_Cancellation(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()
	dashboard.sections["analysis"] = sections.NewAnalysisSection()

	dashboard.analysisCtx, dashboard.analysisCancel = context.WithCancel(context.Background())

	dashboard.Update(tea.KeyMsg{Type: tea.KeyEsc})

	select {
	case <-dashboard.analysisCtx.Done():
	case <-time.After(time.Second):
		t.Error("context should be cancelled")
	}
}

func TestAnalysisRunner_KeyHandling_DuringAnalysis(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()

	tests := []struct {
		name        string
		key         tea.KeyType
		keyRune     rune
		shouldClose bool
	}{
		{"tab ignored", tea.KeyTab, 0, false},
		{"enter ignored during progress", tea.KeyEnter, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := dashboard.progressView.Visible()
			dashboard.Update(tea.KeyMsg{Type: tt.key})
			after := dashboard.progressView.Visible()

			if tt.shouldClose && after {
				t.Error("progress view should have closed")
			}
			if !tt.shouldClose && before != after {
				t.Error("visibility should not change")
			}
		})
	}
}

func TestAnalysisRunner_CloseAfterCompletion(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.progressView.Show()
	dashboard.progressView.SetCompleted(AnalysisSummary{})

	dashboard.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if dashboard.progressView.Visible() {
		t.Error("progress view should be hidden after Enter on completion")
	}
}

func TestAnalysisRunner_ConfigValidation(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.cfg = &config.GlobalConfig{}

	err := dashboard.validateAnalyzerConfig(dashboard.buildAnalyzerConfig())
	if err == nil {
		t.Error("should return error for empty provider")
	}

	dashboard.cfg.Analyzer.LLM.Provider = "openai"
	err = dashboard.validateAnalyzerConfig(dashboard.buildAnalyzerConfig())
	if err == nil {
		t.Error("should return error for missing API key")
	}

	dashboard.cfg.Analyzer.LLM.Provider = "ollama"
	err = dashboard.validateAnalyzerConfig(dashboard.buildAnalyzerConfig())
	if err != nil {
		t.Errorf("should not return error for ollama without API key: %v", err)
	}
}

func TestAnalysisRunner_BuildConfig(t *testing.T) {
	dashboard := NewDashboard()
	dashboard.cfg = &config.GlobalConfig{
		Analyzer: config.AnalyzerConfig{
			LLM: config.LLMConfig{
				Provider: "anthropic",
				Model:    "claude-3",
				APIKey:   "test-key",
			},
			ExcludeStructure: true,
			MaxWorkers:       4,
		},
	}

	cfg := dashboard.buildAnalyzerConfig()

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got '%s'", cfg.LLM.Provider)
	}
	if cfg.RepoPath != "." {
		t.Errorf("expected RepoPath '.', got '%s'", cfg.RepoPath)
	}
	if !cfg.ExcludeStructure {
		t.Error("ExcludeStructure should be true")
	}
	if cfg.MaxWorkers != 4 {
		t.Errorf("expected MaxWorkers 4, got %d", cfg.MaxWorkers)
	}
}

func TestAnalysisRunner_TickHandling(t *testing.T) {
	dashboard := NewDashboard()

	dashboard.Update(TickMsg(time.Now()))
	if dashboard.progressView.spinnerFrame != 0 {
		t.Error("tick should be ignored when progress not visible")
	}

	dashboard.progressView.Show()
	dashboard.Update(TickMsg(time.Now()))
	if dashboard.progressView.spinnerFrame != 1 {
		t.Error("tick should update spinner when progress visible")
	}

	dashboard.progressView.SetCompleted(AnalysisSummary{})
	initialFrame := dashboard.progressView.spinnerFrame
	dashboard.Update(TickMsg(time.Now()))
	if dashboard.progressView.spinnerFrame != initialFrame {
		t.Error("tick should be ignored when analysis complete")
	}
}
