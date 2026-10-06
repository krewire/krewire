package errs

// ExitCode is a standard process exit code used across Krewire applications.
type ExitCode int

const (
	// ExitCodeSuccess indicates successful termination.
	ExitCodeSuccess ExitCode = 0
	// ExitCodeFailure indicates a generic runtime failure.
	ExitCodeFailure ExitCode = 1
	// ExitCodeUsage indicates invalid usage: missing or malformed arguments,
	// configuration, or input.
	ExitCodeUsage ExitCode = 2
)

// Int returns the numeric value used by the operating system.
func (c ExitCode) Int() int {
	return int(c)
}

// ExitCodeFromInt maps a raw process exit code back to the closest known
// ExitCode.
func ExitCodeFromInt(code int) ExitCode {
	switch code {
	case 0:
		return ExitCodeSuccess
	case 2:
		return ExitCodeUsage
	default:
		return ExitCodeFailure
	}
}

// Error pairs a human-readable message with an ExitCode.
type Error struct {
	// Message is the human-readable error message.
	Message string
	// Code is the process exit code associated with the error.
	Code ExitCode
}

// NewError creates an Error with the given message and exit code.
func NewError(message string, code ExitCode) *Error {
	return &Error{Message: message, Code: code}
}

// UsageError creates an Error with ExitCodeUsage.
func UsageError(message string) *Error {
	return NewError(message, ExitCodeUsage)
}

// FailureError creates an Error with ExitCodeFailure.
func FailureError(message string) *Error {
	return NewError(message, ExitCodeFailure)
}

// ExitCode returns the exit code associated with the error.
func (e *Error) ExitCode() ExitCode {
	return e.Code
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}
