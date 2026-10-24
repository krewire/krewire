package errs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Attr is one structured key/value pair attached to an error for diagnostics
// (KWL-P8W2N KWL-ERRV-008).
type Attr struct {
	Key   string
	Value any
}

// attrError attaches structured attributes without altering the message or
// errors.Is/As behavior of the wrapped error.
type attrError struct {
	err   error
	attrs []Attr
}

func (e *attrError) Error() string { return e.err.Error() }

func (e *attrError) Unwrap() error { return e.err }

// WithAttrs attaches structured diagnostic attributes to err. Nil-safe;
// wrapping repeatedly accumulates layers extractable by AttrsOf.
func WithAttrs(err error, attrs ...Attr) error {
	if err == nil {
		return nil
	}
	return &attrError{err: err, attrs: attrs}
}

// AttrsOf collects attributes across the wrap chain, outermost first. Each
// wrap layer contributes its own attributes exactly once.
func AttrsOf(err error) []Attr {
	var out []Attr
	for layer := err; layer != nil; layer = errors.Unwrap(layer) {
		if ae, ok := layer.(*attrError); ok {
			out = append(out, ae.attrs...)
		}
	}
	return out
}

// hintError attaches exactly one actionable next step to an error.
type hintError struct {
	err  error
	hint string
}

func (e *hintError) Error() string { return e.err.Error() }

func (e *hintError) Unwrap() error { return e.err }

// WithHint attaches one actionable user-facing hint to err. Nil-safe.
func WithHint(err error, text string) error {
	if err == nil {
		return nil
	}
	return &hintError{err: err, hint: text}
}

// HintOf returns the nearest hint found walking the wrap chain from the
// outside in, or "" when none is attached (KWL-P8W2N KWL-ERRV-009).
func HintOf(err error) string {
	for layer := err; layer != nil; layer = errors.Unwrap(layer) {
		if he, ok := layer.(*hintError); ok && he.hint != "" {
			return he.hint
		}
	}
	return ""
}

// FormatTree renders err as a human-readable diagnostic tree: the message
// chain top-down, each annotated with its creation point when a stack was
// captured, attributes inline, and the nearest hint as footer
// (KWL-P8W2N KWL-ERRV-010).
func FormatTree(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	first := true
	var lastMsg string
	for layer := err; layer != nil; layer = errors.Unwrap(layer) {
		msg := layer.Error()
		if first {
			fmt.Fprintf(&b, "Error: %s\n", msg)
			first = false
		} else if msg != lastMsg {
			fmt.Fprintf(&b, "caused by: %s\n", msg)
		}
		lastMsg = msg

		if se, ok := layer.(*stackError); ok && len(se.pcs) > 0 {
			if f, ok := runtime.CallersFrames(se.pcs).Next(); ok {
				fmt.Fprintf(&b, "  at %s (%s:%d)\n", f.Function, f.File, f.Line)
			}
		}
		if ae, ok := layer.(*attrError); ok {
			for _, a := range ae.attrs {
				fmt.Fprintf(&b, "  %s=%v\n", a.Key, a.Value)
			}
		}
	}
	if hint := HintOf(err); hint != "" {
		fmt.Fprintf(&b, "Hint: %s\n", hint)
	}
	return b.String()
}
