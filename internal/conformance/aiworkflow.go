package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/fakeengine"
	"github.com/krewire/krewire/pkg/toolchain/policy"
	"github.com/krewire/krewire/pkg/toolchain/projctx"
	"github.com/krewire/krewire/pkg/toolchain/workflow"
)

// aiSecret is the HMAC key the harness signs plan approvals with. It is a
// fixture value, never a credential, and must not appear in any report.
const aiSecret = "conformance-approval-key"

var aiGoodFiles = map[string]string{
	"pkg/greet/greet.go":      "package greet\n\n// Greet greets name.\nfunc Greet(name string) string { return \"hello \" + name }\n",
	"pkg/greet/greet_test.go": "package greet\n\nimport \"testing\"\n\nfunc TestGreet(t *testing.T) {\n\tif Greet(\"x\") != \"hello x\" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n",
}

// aiWorkflowSection qualifies the integrated AI-native workflow (#131/#140):
// the deterministic reference engine runs through workflow.Run — the same
// entrypoint `krew` uses — in-process and over the stdio adapter protocol,
// with real gofmt/go test gates and no LLM credentials. Failure injection
// proves the toolchain, not the engine, decides outcomes: an engine claiming
// a failed gate passed has the claim stripped, and a denied plan approval
// blocks every mutation. The verified run's EvidenceReport is attached.
func aiWorkflowSection(ctx context.Context, env *Env, tempDir string) Section {
	res := &WorkloadResult{Workload: "ai_workflow"}
	r := &run{env: env, ctx: ctx, res: res}
	var attached *contract.EvidenceReport

	for _, mode := range []string{"in-process", "stdio"} {
		r.assert("verified."+mode, gateTest, "reference engine ("+mode+") runs plan -> approval -> implement -> verify -> review; outcome verified",
			"workflow.Run(fakeengine, "+mode+")", func() (string, error) {
				eng, closeFn, err := aiEngine(ctx, &fakeengine.Engine{Files: aiGoodFiles}, mode)
				if err != nil {
					return "", err
				}
				defer closeFn()
				rep, root, err := aiRun(ctx, env, tempDir, eng, &aiSigner{})
				if err != nil {
					return "", err
				}
				if err := aiCheck(rep, contract.OutcomeVerified); err != nil {
					return aiSummary(rep), err
				}
				verified := 0
				for _, c := range rep.Claims {
					if c.VerifiedBy != "" {
						verified++
					}
				}
				switch {
				case len(rep.Approvals) != 1 || !rep.Approvals[0].Verified:
					err = fmt.Errorf("plan approval not recorded as verified: %+v", rep.Approvals)
				case len(rep.Mutations) != len(aiGoodFiles):
					err = fmt.Errorf("want %d mutations, got %d", len(aiGoodFiles), len(rep.Mutations))
				case verified != 1:
					err = fmt.Errorf("want exactly one claim linked to a passed gate, got %d", verified)
				}
				if err == nil {
					_, err = os.Stat(filepath.Join(root, "pkg/greet/greet_test.go"))
				}
				if err == nil && mode == "in-process" {
					attached = rep
				}
				return aiSummary(rep), err
			})
	}

	r.assert("inject.false-claim", gateTest, "engine claims a failed test gate passed: outcome verification_failed and the claim is stripped",
		"workflow.Run(liar engine, failing test)", func() (string, error) {
			bad := map[string]string{"pkg/greet/greet.go": aiGoodFiles["pkg/greet/greet.go"],
				"pkg/greet/greet_test.go": strings.Replace(aiGoodFiles["pkg/greet/greet_test.go"], "hello x", "bye", 1)}
			rep, _, err := aiRun(ctx, env, tempDir, &aiLiar{fakeengine.Engine{Files: bad}}, &aiSigner{})
			if err != nil {
				return "", err
			}
			if err := aiCheck(rep, contract.OutcomeVerificationFailed); err != nil {
				return aiSummary(rep), err
			}
			for _, c := range rep.Claims {
				if c.VerifiedBy != "" {
					return aiSummary(rep), fmt.Errorf("engine claim %q kept verified_by %q", c.Statement, c.VerifiedBy)
				}
			}
			return aiSummary(rep), nil
		})

	r.assert("inject.approval-denied", gateTest, "a denied plan approval blocks implementation: outcome denied, nothing executed or written",
		"workflow.Run(fakeengine, approver denies)", func() (string, error) {
			rep, root, err := aiRun(ctx, env, tempDir, &fakeengine.Engine{Files: aiGoodFiles}, aiDenier{})
			if err != nil {
				return "", err
			}
			if err := aiCheck(rep, contract.OutcomeDenied); err != nil {
				return aiSummary(rep), err
			}
			if len(rep.Executions) != 0 || len(rep.Mutations) != 0 {
				return aiSummary(rep), fmt.Errorf("denied plan still executed %d call(s), %d mutation(s)", len(rep.Executions), len(rep.Mutations))
			}
			if _, err := os.Stat(filepath.Join(root, "pkg/greet")); !errors.Is(err, os.ErrNotExist) {
				return aiSummary(rep), errors.New("denied plan wrote to the repository")
			}
			return aiSummary(rep), nil
		})

	sec := Section{Status: StatusPassed, Gates: res.Gates, EvidenceReport: attached}
	for _, g := range res.Gates {
		if g.Status != model.GatePassed {
			sec.Status = StatusFailed
		}
	}
	return sec
}

// aiRun creates a fresh repository, derives its ProjectContext the way
// `krew context` does, and runs the workflow with eng.
func aiRun(ctx context.Context, env *Env, tempDir string, eng adapter.Engine, ap workflow.Approver) (*contract.EvidenceReport, string, error) {
	root, err := os.MkdirTemp(tempDir, "ai-workflow-")
	if err != nil {
		return nil, "", err
	}
	files := map[string]string{
		"go.mod":    "module example.com/demo\n\ngo 1.21\n",
		"AGENTS.md": "## Quality Gates\n\n```\ngofmt -l .\ngo test ./...\n```\n",
	}
	for rel, c := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(c), 0o644); err != nil {
			return nil, "", err
		}
	}
	pctx, err := projctx.Generate(projctx.Options{Root: root})
	if err != nil {
		return nil, "", err
	}
	for i := range pctx.Gates {
		if pctx.Gates[i].Argv[0] == "go" && env.GoBin != "" {
			pctx.Gates[i].Argv[0] = env.GoBin // the toolchain the harness qualifies with
		}
	}
	task := &contract.TaskBrief{
		TypeMeta: contract.Meta(contract.KindTaskBrief), ID: "conformance-ai-1", Title: "Add greet package",
		Objective: "Create pkg/greet with Greet and a test", ExpectedOutcome: "go test passes",
		Scope: contract.TaskScope{Allowed: []string{"pkg/greet"}},
		Risk:  contract.RiskLow, MaxAutonomy: model.AutonomyAutonomous,
		Verification: contract.Verification{Gates: []string{"fmt", "test"}, Condition: contract.ConditionAllRequiredPassed},
		Deliverables: []string{"pkg/greet/greet.go", "pkg/greet/greet_test.go"},
		Limits:       contract.Limits{TimeoutSec: 120, MaxToolCalls: 16},
		Approval:     contract.ApprovalPolicy{Plan: true, Approvers: []string{"maintainer"}},
	}
	rep, err := workflow.Run(ctx, workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Engine: eng, Approver: ap, Secret: aiSecret, Commit: env.Commit,
		Ledger: model.NewVerificationLedger(task.ID)})
	return rep, root, err
}

// aiCheck validates the report against its schema and lifecycle invariants.
func aiCheck(rep *contract.EvidenceReport, want contract.Outcome) error {
	data, err := json.Marshal(rep)
	if err != nil {
		return err
	}
	if err := contract.ValidateJSON(contract.KindEvidenceReport, data); err != nil {
		return fmt.Errorf("report violates its schema: %w", err)
	}
	if err := rep.Validate(); err != nil {
		return err
	}
	if strings.Contains(string(data), aiSecret) {
		return errors.New("approval secret leaked into the report")
	}
	if len(rep.Stages) != len(contract.Stages) {
		return fmt.Errorf("want all %d lifecycle stages recorded, got %d", len(contract.Stages), len(rep.Stages))
	}
	if rep.Outcome != want {
		return fmt.Errorf("outcome %s, want %s", rep.Outcome, want)
	}
	return nil
}

func aiSummary(rep *contract.EvidenceReport) string {
	if rep == nil {
		return ""
	}
	var gates []string
	for _, g := range rep.Verification.Gates {
		gates = append(gates, g.GateID+"="+string(g.Status))
	}
	return fmt.Sprintf("outcome=%s plan=%s gates=[%s] executions=%d mutations=%d claims=%d",
		rep.Outcome, rep.PlanDigest, strings.Join(gates, " "), len(rep.Executions), len(rep.Mutations), len(rep.Claims))
}

// aiEngine returns eng in-process, or behind adapter.Serve over in-memory
// pipes so the stdio protocol is exercised end to end.
func aiEngine(ctx context.Context, eng adapter.Engine, mode string) (adapter.Engine, func(), error) {
	if mode != "stdio" {
		return eng, func() {}, nil
	}
	hostR, engW := io.Pipe()
	engR, hostW := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- adapter.Serve(ctx, eng, engR, engW); _ = engW.Close() }()
	c := adapter.NewClient(hostR, hostW)
	closeFn := func() { _ = c.Close(); <-done }
	if _, err := c.Initialize(ctx, adapter.InitializeParams{ContractVersion: contract.ContractVersion}); err != nil {
		closeFn()
		return nil, nil, fmt.Errorf("initialize over stdio: %w", err)
	}
	return c, closeFn, nil
}

// aiSigner approves every scope as the maintainer.
type aiSigner struct{}

func (aiSigner) Approve(_ context.Context, scope, _ string) (contract.ApprovalRecord, error) {
	return policy.Sign("maintainer", scope, aiSecret, time.Minute)
}

// aiDenier is a reviewer who rejects the plan.
type aiDenier struct{}

func (aiDenier) Approve(context.Context, string, string) (contract.ApprovalRecord, error) {
	return contract.ApprovalRecord{}, errors.New("plan rejected by reviewer")
}

// aiLiar claims its work is verified by the test gate whatever happened.
type aiLiar struct{ fakeengine.Engine }

func (l *aiLiar) Review(context.Context, adapter.ReviewParams) (adapter.ReviewResult, error) {
	return adapter.ReviewResult{Claims: []contract.Claim{{Source: "engine:liar", Stage: contract.StageReview,
		Statement: "all tests pass", VerifiedBy: "test"}}}, nil
}
