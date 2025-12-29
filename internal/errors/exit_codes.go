package errors

type ExitCode int

const (
	ExitSuccess         ExitCode = 0
	ExitGeneralError    ExitCode = 1
	ExitConfigError     ExitCode = 2
	ExitValidationError ExitCode = 3
	ExitLLMError        ExitCode = 4
	ExitAgentError      ExitCode = 5
	ExitIOError         ExitCode = 6
	ExitGitLabError     ExitCode = 7
	ExitPartialSuccess  ExitCode = 10
)

func (e ExitCode) Int() int {
	return int(e)
}
