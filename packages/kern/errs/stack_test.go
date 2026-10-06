package errs

import (
	"errors"
	"strings"
	"testing"
)

func TestWithStack(t *testing.T) {
	if WithStack(nil) != nil {
		t.Error("WithStack(nil) should be nil")
	}
	base := errors.New("something went wrong")
	stacked := WithStack(base)
	if stacked == nil {
		t.Fatal("WithStack returned nil")
	}
	if !errors.Is(stacked, base) {
		t.Error("errors.Is failed on stacked error")
	}
	frames := StackOf(stacked)
	if len(frames) == 0 {
		t.Fatal("StackOf returned 0 frames")
	}
	formatted := FormatStack(frames)
	if !strings.Contains(formatted, "stack_test.go") {
		t.Errorf("FormatStack missing file name: %s", formatted)
	}
	if StackOf(base) != nil {
		t.Error("StackOf(unstacked) should be nil")
	}
}
