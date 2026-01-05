package dashboard

import (
	"time"
)

type RunAnalysisMsg struct{}

type ProgressEvent int

const (
	ProgressEventTaskAdded ProgressEvent = iota
	ProgressEventTaskStarted
	ProgressEventTaskCompleted
	ProgressEventTaskFailed
	ProgressEventTaskSkipped
)

type AnalysisProgressMsg struct {
	TaskID      string
	TaskName    string
	Description string
	Event       ProgressEvent
	Error       error
}

type AnalysisCompleteMsg struct {
	Successful []string
	Failed     []FailedAnalysis
	Duration   time.Duration
}

type FailedAnalysis struct {
	Name  string
	Error error
}

type AnalysisErrorMsg struct {
	Error error
}

type CancelAnalysisMsg struct{}

type AnalysisCancelledMsg struct{}

type AnalysisStartedMsg struct{}
