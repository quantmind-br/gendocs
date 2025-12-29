package types

import tea "github.com/charmbracelet/bubbletea"

type ValidationSeverity int

const (
	SeverityError ValidationSeverity = iota
	SeverityWarning
	SeverityInfo
)

type ValidationError struct {
	Field    string
	Message  string
	Severity ValidationSeverity
}

type SectionModel interface {
	tea.Model

	Title() string
	Icon() string
	Description() string
	Validate() []ValidationError
	IsDirty() bool
	GetValues() map[string]any
	SetValues(values map[string]any) error
	FocusFirst() tea.Cmd
	FocusLast() tea.Cmd
}
