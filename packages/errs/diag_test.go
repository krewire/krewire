package errs

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWithAttrs(t *testing.T) {
	base := errors.New("disk full")
	wrapped := WithAttrs(WithStack(WithAttrs(base, Attr{Key: "path", Value: "/tmp/a.yaml"})), Attr{Key: "step", Value: "build"})

	got := AttrsOf(wrapped)
	if len(got) != 2 {
		t.Fatalf("AttrsOf len = %d (%v), want 2 attrs outermost-first", len(got), got)
	}
	if got[0].Key != "step" || got[1].Key != "path" {
		t.Errorf("AttrsOf order = %v/%v, want step then path", got[0], got[1])
	}

	var target *Error
	if !errors.As(wrapped, &target) && !errors.Is(wrapped, base) {
		t.Error("errors.Is/As broken through attr wrap")
	}
	if !errors.Is(wrapped, base) {
		t.Error("errors.Is(wrapped, base) = false, want true")
	}
	if AttrsOf(nil) != nil {
		t.Error("AttrsOf(nil) should be nil")
	}
	if WithAttrs(nil, Attr{Key: "k", Value: "v"}) != nil {
		t.Error("WithAttrs(nil) should stay nil")
	}
}

func TestHintOf(t *testing.T) {
	base := FailureError("config invalid")
	deep := WithHint(base, "check project.kind in krewire.yaml")
	wrapped := WithHint(deep, "run 'kiw init' to create one")

	if got := HintOf(wrapped); got != "run 'kiw init' to create one" {
		t.Errorf("HintOf(nearest) = %q", got)
	}
	if got := HintOf(deep); got != "check project.kind in krewire.yaml" {
		t.Errorf("HintOf(deep) = %q", got)
	}
	if got := HintOf(base); got != "" {
		t.Errorf("HintOf(unhinted) = %q, want empty", got)
	}
	if got := HintOf(nil); got != "" {
		t.Errorf("HintOf(nil) = %q, want empty", got)
	}
}

func TestFormatTree(t *testing.T) {
	inner := errors.New("open krewire.yaml: no such file or directory")
	err := WithStack(fmt.Errorf("cannot read config: %w",
		WithHint(
			WithAttrs(inner, Attr{Key: "file", Value: "krewire.yaml"}),
			"run 'kiw init' to create one",
		),
	))

	tree := FormatTree(err)
	for _, want := range []string{
		"Error: cannot read config: open krewire.yaml",
		"file=krewire.yaml",
		"caused by: open krewire.yaml: no such file or directory",
		"caused by:",
		"at ",
		".go:",
		"Hint: run 'kiw init' to create one",
	} {
		if !strings.Contains(tree, want) {
			t.Errorf("tree missing %q:\n%s", want, tree)
		}
	}

	plain := FormatTree(errors.New("boom"))
	if !strings.HasPrefix(plain, "Error: boom\n") || strings.Contains(plain, "Hint:") {
		t.Errorf("plain tree malformed:\n%s", plain)
	}
	if FormatTree(nil) != "" {
		t.Error("FormatTree(nil) should be empty")
	}
}
