package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/model"
)

func golden[T any](t *testing.T, name string) *T {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	v := new(T)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestScope(t *testing.T) {
	s := TaskScope{Allowed: []string{"pkg/greet", "docs/a.md"}, Prohibited: []string{"pkg/greet/testdata"},
		Commands: [][]string{{"go", "test"}}}
	for p, want := range map[string]bool{
		"pkg/greet/x.go": true, "pkg/greet": true, "docs/a.md": true, "pkg/greeter/x.go": false,
		"pkg/greet/testdata/f": false, "../etc/passwd": false, "/abs": false, ".": false, "pkg/greet/../../x": false,
	} {
		if got := s.InScope(p); got != want {
			t.Errorf("InScope(%q) = %v, want %v", p, got, want)
		}
	}
	if !s.AllowsCommand([]string{"go", "test", "./..."}) || s.AllowsCommand([]string{"go", "run"}) || s.AllowsCommand([]string{"go"}) {
		t.Error("AllowsCommand prefix matching is wrong")
	}
	if !(TaskScope{Allowed: []string{"."}}).InScope("any/file") {
		t.Error(`"." must grant the repository`)
	}
}

func TestErrors(t *testing.T) {
	err := fmt.Errorf("wrap: %w", Errorf(CodeScopeViolation, "x"))
	if !errors.Is(err, ErrScopeViolation) || errors.Is(err, ErrTimeout) {
		t.Error("errors.Is must match by code")
	}
	if AsError(context.DeadlineExceeded).Code != CodeTimeout || AsError(context.Canceled).Code != CodeCancelled ||
		AsError(errors.New("x")).Code != CodeInternal || AsError(nil) != nil {
		t.Error("AsError mapping is wrong")
	}
	seen := map[string]bool{}
	for _, c := range Codes() {
		if seen[c.String()] {
			t.Errorf("duplicate code name %s", c)
		}
		seen[c.String()] = true
	}
	if Code(1).String() != "code(1)" || ErrPolicyDenied.Error() == "" {
		t.Error("unexpected code rendering")
	}
}

func TestContractVersionNegotiation(t *testing.T) {
	for v, ok := range map[string]bool{"1.0.0": true, "1.9.3": true, "v1.2": true, "2.0.0": false, "": false, "0.9.0": false} {
		if err := CompatibleContract(v); (err == nil) != ok {
			t.Errorf("CompatibleContract(%q) = %v", v, err)
		}
	}
	if err := (TypeMeta{APIVersion: APIVersion, Kind: KindPlan}).Check(KindTaskBrief); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("kind mismatch: %v", err)
	}
}

func TestTaskBriefValidation(t *testing.T) {
	ctx := golden[ProjectContext](t, "project-context.json")
	base := golden[TaskBrief](t, "task-brief.json")
	if err := base.Validate(ctx); err != nil {
		t.Fatalf("golden brief: %v", err)
	}
	mutate := map[string]func(b *TaskBrief){
		"no objective":    func(b *TaskBrief) { b.Objective = " " },
		"no scope":        func(b *TaskBrief) { b.Scope.Allowed = nil },
		"escaping scope":  func(b *TaskBrief) { b.Scope.Allowed = []string{"../x"} },
		"bad risk":        func(b *TaskBrief) { b.Risk = "extreme" },
		"bad autonomy":    func(b *TaskBrief) { b.MaxAutonomy = "ROGUE" },
		"bad condition":   func(b *TaskBrief) { b.Verification.Condition = "vibes" },
		"no gates":        func(b *TaskBrief) { b.Verification.Gates = nil },
		"unknown gate":    func(b *TaskBrief) { b.Verification.Gates = []string{"lint"} },
		"negative limits": func(b *TaskBrief) { b.Limits.MaxToolCalls = -1 },
		"wrong kind":      func(b *TaskBrief) { b.Kind = KindPlan },
	}
	for name, m := range mutate {
		b := *base
		b.Scope.Allowed = append([]string(nil), base.Scope.Allowed...)
		m(&b)
		if err := b.Validate(ctx); !errors.Is(err, ErrInvalidDocument) {
			t.Errorf("%s: want invalid_document, got %v", name, err)
		}
	}
	legacy := base.ToModel(ctx)
	if legacy.Validate() != nil || len(legacy.VerificationCommands) != 2 || !legacy.RequirePlan {
		t.Errorf("ToModel lost data: %+v", legacy)
	}
}

func TestPlanValidation(t *testing.T) {
	brief := golden[TaskBrief](t, "task-brief.json")
	plan := golden[Plan](t, "plan.json")
	if err := plan.Validate(brief); err != nil {
		t.Fatal(err)
	}
	d := plan.ComputeDigest()
	plan.Digest = d
	if plan.ComputeDigest() != d {
		t.Error("digest must ignore the digest field")
	}
	plan.Steps[0].TargetFiles = []string{"pkg/other/x.go"}
	if err := plan.Validate(brief); !errors.Is(err, ErrScopeViolation) {
		t.Errorf("want scope violation, got %v", err)
	}
	if plan.ComputeDigest() == d {
		t.Error("an edited plan must change its digest")
	}
	plan.Steps[0].DependsOn = []string{"s2"}
	if err := plan.Validate(nil); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("forward dependency must fail, got %v", err)
	}
	other := golden[Plan](t, "plan.json")
	other.TaskID = "999"
	if err := other.Validate(brief); err == nil {
		t.Error("task id mismatch must fail")
	}
}

func TestReportKeepsClaimsApartFromEvidence(t *testing.T) {
	r := golden[EvidenceReport](t, "evidence-report.json")
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Verification.Gates[0].Status = model.GateFailed
	if err := r.Validate(); err == nil {
		t.Error("verified=true over a failed gate must be rejected")
	}
	r.Verification.Verified, r.Outcome = false, OutcomeVerificationFailed
	if err := r.Validate(); err == nil {
		t.Error("a claim citing a failed gate must be rejected")
	}
	r.Claims[0].VerifiedBy = ""
	if err := r.Validate(); err != nil {
		t.Errorf("unverified claims are allowed: %v", err)
	}
	r.Outcome = OutcomeVerified
	if err := r.Validate(); err == nil {
		t.Error("outcome verified without passing gates must be rejected")
	}
}

func TestCatalogInvariants(t *testing.T) {
	c := Catalog()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range c.Commands {
		if cmd.Class != ClassAgentic && cmd.RequiresEngine {
			t.Errorf("%s: non-agentic command requires an engine", cmd.Path)
		}
	}
	for _, p := range []string{"context", "verify", "test", "build", "fmt", "compat", "release"} {
		if cmd, ok := c.Lookup(p); !ok || cmd.Class != ClassDeterministic {
			t.Errorf("%s must be a deterministic krew command", p)
		}
	}
	if _, ok := c.Lookup("nope"); ok {
		t.Error("lookup of unknown command")
	}
	c.Commands = append(c.Commands, Command{Path: "fmt", Class: ClassDeterministic})
	if c.Validate() == nil {
		t.Error("duplicate commands must fail")
	}
	bad := Catalog()
	bad.Commands[0].RequiresEngine = true
	if bad.Validate() == nil {
		t.Error("deterministic command requiring an engine must fail")
	}
	bad = Catalog()
	bad.Executable = "kiw"
	if bad.Validate() == nil {
		t.Error("kiw is retired")
	}
}

func TestToolCall(t *testing.T) {
	w := ToolCall{ID: "1", Tool: ToolWriteFile, Path: "a", Content: "x"}
	if w.Redacted().Content != "" || w.Redacted().ContentSHA256 == "" || w.Digest() == (ToolCall{ID: "1", Tool: ToolWriteFile, Path: "a", Content: "y"}).Digest() {
		t.Error("redaction/digest must bind content by hash")
	}
	for call, want := range map[*ToolCall]model.ActionCategory{
		{Tool: ToolWriteFile}: model.ActionWrite, {Tool: ToolDeleteFile}: model.ActionDelete,
		{Tool: ToolRunCommand, Argv: []string{"go", "test"}}:  model.ActionCommand,
		{Tool: ToolRunCommand, Argv: []string{"git", "push"}}: model.ActionGitPush,
		{Tool: ToolFetch}: model.ActionNetwork, {Tool: "x"}: "UNKNOWN",
	} {
		if call.Category() != want {
			t.Errorf("%s: category %s, want %s", call.Tool, call.Category(), want)
		}
	}
	ctx := golden[ProjectContext](t, "project-context.json")
	if ctx.Blocking() || ctx.Validate() != nil {
		t.Error("golden context has only a warning and must validate")
	}
	ctx.Gates = append(ctx.Gates, ctx.Gates[0])
	if ctx.Validate() == nil {
		t.Error("duplicate gate must fail")
	}
}
