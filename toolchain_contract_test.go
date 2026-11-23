package krewire_test

import (
	"encoding/json"
	"testing"

	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/projctx"
)

// TestProjectContextLint derives this repository's project context from its
// source-of-truth files and fails on any blocking (error-severity)
// diagnostic, so context and contract lint run in CI with `go test ./...`
// (ref #140). Warnings, such as stale `kiw` references, are reported only.
func TestProjectContextLint(t *testing.T) {
	root := repoRoot(t)
	ctx, err := projctx.Generate(projctx.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ctx.Diagnostics {
		if d.Severity == contract.SeverityError {
			t.Errorf("%s %s:%d: %s (hint: %s)", d.Code, d.Location.Path, d.Location.Line, d.Message, d.Hint)
		} else {
			t.Logf("%s %s %s:%d: %s", d.Severity, d.Code, d.Location.Path, d.Location.Line, d.Message)
		}
	}
	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := contract.ValidateJSON(contract.KindProjectContext, data); err != nil {
		t.Fatalf("repository context violates the published schema: %v", err)
	}
	for _, id := range []string{"fmt", "vet", "test", "build"} {
		if _, ok := ctx.Gate(id); !ok {
			t.Errorf("AGENTS.md no longer declares the %q quality gate", id)
		}
	}
	if ctx.Module.Path != modulePath(t, root) {
		t.Errorf("context module %q differs from go.mod", ctx.Module.Path)
	}
}
