package krewire_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/toolchain/adapter"
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

// TestToolchainContractSpecMatchesCode keeps the normative spec in step
// with the code: every command in the catalog, adapter method, error code,
// and lint code must be documented, so documentation examples match the
// actual unified command surface (ref #131, #140).
func TestToolchainContractSpecMatchesCode(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "tools", "krew", "ai-native-contract.md"))
	if err != nil {
		t.Fatal(err)
	}
	spec := string(data)
	var want []string
	for _, c := range contract.Catalog().Commands {
		want = append(want, "`"+contract.Executable+" "+c.Path)
	}
	for _, c := range contract.Codes() {
		want = append(want, fmt.Sprintf("| %d | `%s` |", int(c), c))
	}
	for code := range projctx.Codes {
		want = append(want, "`"+code+"`")
	}
	for _, m := range []string{adapter.MethodInitialize, adapter.MethodPlan, adapter.MethodImplement,
		adapter.MethodReview, adapter.MethodShutdown, adapter.NotifyCancel, adapter.EnvEngine,
		contract.APIVersion, contract.ContractVersion} {
		want = append(want, m)
	}
	for _, s := range contract.Stages {
		want = append(want, string(s))
	}
	for _, w := range want {
		if !strings.Contains(spec, w) {
			t.Errorf("docs/tools/krew/ai-native-contract.md does not document %q", w)
		}
	}
}
