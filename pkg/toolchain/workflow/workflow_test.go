package workflow_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/conformance"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/fakeengine"
	"github.com/krewire/krewire/pkg/toolchain/policy"
	"github.com/krewire/krewire/pkg/toolchain/projctx"
	"github.com/krewire/krewire/pkg/toolchain/workflow"
)

const secret = "host-approval-key"

var goodFiles = map[string]string{
	"pkg/greet/greet.go":      "package greet\n\n// Greet greets name.\nfunc Greet(name string) string { return \"hello \" + name }\n",
	"pkg/greet/greet_test.go": "package greet\n\nimport \"testing\"\n\nfunc TestGreet(t *testing.T) {\n\tif Greet(\"x\") != \"hello x\" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n",
}

// repo creates a fresh project and derives its context the way `krew
// context` does: discovery is deterministic and needs no engine.
func repo(t *testing.T) (string, *contract.ProjectContext, *contract.TaskBrief) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":    "module example.com/demo\n\ngo 1.21\n",
		"AGENTS.md": "## Quality Gates\n\n```\ngofmt -l .\ngo test ./...\n```\n",
	}
	for rel, c := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pctx, err := projctx.Generate(projctx.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	_, task := conformance.Fixture()
	task.Verification.Gates = []string{"fmt", "test"}
	task.Approval = contract.ApprovalPolicy{Plan: true, Approvers: []string{"maintainer"}}
	return root, pctx, task
}

type signer struct{ calls int }

func (s *signer) Approve(_ context.Context, scope, _ string) (contract.ApprovalRecord, error) {
	s.calls++
	return policy.Sign("maintainer", scope, secret, time.Minute)
}

type fakeRunner struct{ status model.VerificationStatus }

func (f fakeRunner) RunGate(_ context.Context, _ string, g contract.Gate) model.VerificationEvidence {
	return model.VerificationEvidence{GateID: g.ID, Type: g.Type, Command: strings.Join(g.Argv, " "), Required: g.Required, Status: f.status}
}

func checkReport(t *testing.T, r *contract.EvidenceReport, want contract.Outcome) {
	t.Helper()
	if r.Outcome != want {
		t.Fatalf("outcome %s, want %s; stages %+v", r.Outcome, want, r.Stages)
	}
	data, _ := json.Marshal(r)
	if err := contract.ValidateJSON(contract.KindEvidenceReport, data); err != nil {
		t.Fatalf("report violates its schema: %v", err)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(r.Stages) != len(contract.Stages) {
		t.Fatalf("every lifecycle stage must be recorded: %+v", r.Stages)
	}
	for i, s := range r.Stages {
		if s.Stage != contract.Stages[i] {
			t.Fatalf("stage %d is %s, want %s", i, s.Stage, contract.Stages[i])
		}
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("the approval secret leaked into the report")
	}
}

// TestEndToEndFlow is the documented flow: discover → plan → approve →
// implement → deterministic verification (real gofmt and go test) →
// review → report.
func TestEndToEndFlow(t *testing.T) {
	for name, wrap := range map[string]func(t *testing.T, e adapter.Engine) adapter.Engine{
		"in-process": func(_ *testing.T, e adapter.Engine) adapter.Engine { return e },
		"stdio":      func(t *testing.T, e adapter.Engine) adapter.Engine { return conformance.Pipe(t, e) },
	} {
		t.Run(name, func(t *testing.T) {
			root, pctx, task := repo(t)
			ap := &signer{}
			ledger := model.NewVerificationLedger(task.ID)
			r, err := workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
				Policy: contract.DefaultPolicy(), Engine: wrap(t, &fakeengine.Engine{Files: goodFiles}),
				Approver: ap, Secret: secret, Commit: "abc123", Ledger: ledger})
			if err != nil {
				t.Fatal(err)
			}
			checkReport(t, r, contract.OutcomeVerified)
			if ap.calls != 1 || len(r.Approvals) != 1 || !r.Approvals[0].Verified || r.Approvals[0].Token != "" {
				t.Errorf("plan approval: %+v", r.Approvals)
			}
			if len(r.Mutations) != 2 || r.Mutations[0].Action != "created" {
				t.Errorf("mutations: %+v", r.Mutations)
			}
			if _, err := os.Stat(filepath.Join(root, "pkg/greet/greet_test.go")); err != nil {
				t.Error(err)
			}
			if r.Verification.LedgerBlockHash == "" || ledger.Blocks[0].BlockHash != r.Verification.LedgerBlockHash {
				t.Error("verification must be chained into the ledger")
			}
			verified := 0
			for _, c := range r.Claims {
				if c.VerifiedBy != "" {
					verified++
				}
			}
			if len(r.Claims) < 3 || verified != 1 {
				t.Errorf("claims stay separate from evidence: %+v", r.Claims)
			}
			if r.Engine == nil || r.Stages[1].Actor != "engine:"+fakeengine.Name || r.Stages[4].Actor != "toolchain" {
				t.Errorf("actors: %+v", r.Stages)
			}
		})
	}
}

// liar claims its work is verified by tests whatever happened.
type liar struct{ fakeengine.Engine }

func (l *liar) Review(ctx context.Context, p adapter.ReviewParams) (adapter.ReviewResult, error) {
	return adapter.ReviewResult{Claims: []contract.Claim{{Source: "engine:liar", Stage: contract.StageReview,
		Statement: "all tests pass", VerifiedBy: "test"}}}, nil
}

func TestFailedGateIsNotOverriddenByEngineClaims(t *testing.T) {
	root, pctx, task := repo(t)
	bad := map[string]string{"pkg/greet/greet.go": goodFiles["pkg/greet/greet.go"],
		"pkg/greet/greet_test.go": strings.Replace(goodFiles["pkg/greet/greet_test.go"], "hello x", "bye", 1)}
	r, err := workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Engine: &liar{fakeengine.Engine{Files: bad}}, Approver: &signer{}, Secret: secret})
	if err != nil {
		t.Fatal(err)
	}
	checkReport(t, r, contract.OutcomeVerificationFailed)
	for _, g := range r.Verification.Gates {
		if g.GateID == "test" && (g.Status != model.GateFailed || g.ExitCode == 0 || !strings.Contains(g.Output, "FAIL")) {
			t.Errorf("raw gate outcome must be kept: %+v", g)
		}
	}
	for _, c := range r.Claims {
		if c.VerifiedBy != "" {
			t.Errorf("engine claim %q must not be marked verified", c.Statement)
		}
	}
}
