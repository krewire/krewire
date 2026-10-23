package errs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

const stackTraceDepth = 32

// stackError attaches an immutable captured call stack to an error without
// altering its identity: errors.Is/As keep working through Unwrap
// (KWL-P8W2N KWL-ERRV-001).
type stackError struct {
	err error
	pcs []uintptr
}

// Error delegates to the wrapped error.
func (e *stackError) Error() string { return e.err.Error() }

// Unwrap exposes the wrapped error for errors.Is/As.
func (e *stackError) Unwrap() error { return e.err }

// WithStack captures the calling goroutine's stack at the wrap point and
// attaches it to err. Nil-safe; stacking an already-stacked error adds a
// second, outer trace (KWL-P8W2N KWL-ERRV-003).
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	pcs := make([]uintptr, stackTraceDepth)
	n := runtime.Callers(2, pcs)
	return &stackError{err: err, pcs: pcs[:n]}
}

// StackFrame is one rendered-ready entry of a captured stack.
type StackFrame struct {
	Func string
	File string
	Line int
}

// String renders "Func File:Line".
func (f StackFrame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.Func, f.File, f.Line)
}

// StackOf extracts the most recently attached stack from err, or nil
// (KWL-P8W2N KWL-ERRV-002).
func StackOf(err error) []StackFrame {
	var se *stackError
	if !errors.As(err, &se) {
		return nil
	}
	frames := runtime.CallersFrames(se.pcs)
	out := make([]StackFrame, 0, len(se.pcs))
	for {
		f, more := frames.Next()
		out = append(out, StackFrame{Func: f.Function, File: f.File, Line: f.Line})
		if !more {
			break
		}
	}
	return out
}

// FormatStack renders frames as an indented multi-line trace, newest first.
func FormatStack(frames []StackFrame) string {
	var b strings.Builder
	for _, f := range frames {
		b.WriteString("  at ")
		b.WriteString(f.Func)
		b.WriteString("\n      ")
		b.WriteString(f.File)
		b.WriteByte(':')
		fmt.Fprintf(&b, "%d", f.Line)
		b.WriteByte('\n')
	}
	return b.String()
}
