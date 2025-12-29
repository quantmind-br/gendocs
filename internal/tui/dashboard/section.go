package dashboard

import "github.com/user/gendocs/internal/tui/dashboard/types"

type ValidationSeverity = types.ValidationSeverity
type ValidationError = types.ValidationError
type SectionModel = types.SectionModel

const (
	SeverityError   = types.SeverityError
	SeverityWarning = types.SeverityWarning
	SeverityInfo    = types.SeverityInfo
)
