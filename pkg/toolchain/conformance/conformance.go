// Package conformance is the executable compatibility suite for adapter
// engines. Any engine — the proprietary Krew engine, a third-party agent, or
// the reference fakeengine — proves it implements the toolchain contract by
// running Run (in-process) and RunStdio (across the JSON-RPC stdio
// boundary) from its own tests, without importing private Krew source and
// without an LLM (engines run their deterministic/mock provider).
package conformance

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// Fixture returns the context and task every conforming engine must plan
// and implement: two Go deliverables inside a narrow scope.
func Fixture() (*contract.ProjectContext, *contract.TaskBrief) {
	ctx := &contract.ProjectContext{
		TypeMeta:    contract.Meta(contract.KindProjectContext),
		Fingerprint: "conformance",
		Module:      contract.ModuleIdentity{Path: "example.com/conformance", Version: "0.1.0"},
		Toolchain:   contract.ToolchainIdentity{Executable: contract.Executable, ContractVersion: contract.ContractVersion},
		Pillars:     contract.Pillars,
		Gates: []contract.Gate{{ID: "test", Type: model.GateTypeTest, Argv: []string{"go", "test", "./..."}, Required: true,
			Provenance: contract.Provenance{Path: "AGENTS.md", Schema: "agents-md"}}},
	}
	task := &contract.TaskBrief{
		TypeMeta: contract.Meta(contract.KindTaskBrief), ID: "conformance-1", Title: "Add greet package",
		Objective: "Create pkg/greet with Greet and a test", ExpectedOutcome: "go test passes",
		Scope: contract.TaskScope{Allowed: []string{"pkg/greet"}},
		Risk:  contract.RiskLow, MaxAutonomy: model.AutonomyAutonomous,
		Verification: contract.Verification{Gates: []string{"test"}, Condition: contract.ConditionAllRequiredPassed},
		Deliverables: []string{"pkg/greet/greet.go", "pkg/greet/greet_test.go"},
		Limits:       contract.Limits{TimeoutSec: 60, MaxToolCalls: 16},
	}
	return ctx, task
}

var knownTools = []contract.ToolName{contract.ToolWriteFile, contract.ToolDeleteFile, contract.ToolRunCommand, contract.ToolFetch}

// Run checks the engine's behaviour against the contract.
func Run(t *testing.T, newEngine func(t *testing.T) adapter.Engine) {
	t.Helper()
	pctx, task := Fixture()
	bg := context.Background()
	start := func(t *testing.T) adapter.Engine {
		eng := newEngine(t)
		if _, err := eng.Initialize(bg, adapter.InitializeParams{ContractVersion: contract.ContractVersion}); err != nil {
			t.Fatalf("initialize: %v", err)
		}
		return eng
	}

	t.Run("initialize", func(t *testing.T) {
		info, err := newEngine(t).Initialize(bg, adapter.InitializeParams{ContractVersion: contract.ContractVersion,
			Host: contract.EngineIdentity{Name: contract.Executable}})
		if err != nil {
			t.Fatal(err)
		}
		if info.Engine.Name == "" || contract.CompatibleContract(info.Engine.ContractVersion) != nil {
			t.Errorf("engine identity %+v is incomplete or incompatible", info.Engine)
		}
		for _, m := range []string{adapter.MethodPlan, adapter.MethodImplement, adapter.MethodReview} {
			if !slices.Contains(info.Methods, m) {
				t.Errorf("engine does not declare method %q", m)
			}
		}
	})

	t.Run("lifecycle", func(t *testing.T) {
		eng := start(t)
		pr, err := eng.Plan(bg, adapter.PlanParams{Context: pctx, Task: task})
		if err != nil {
			t.Fatal(err)
		}
		plan := pr.Plan
		if err := plan.Validate(task); err != nil {
			t.Fatalf("plan violates the contract: %v", err)
		}
		checkClaims(t, pr.Claims, nil)
		var execs []contract.ExecutionRecord
		for _, s := range plan.Steps {
			ir, err := eng.Implement(bg, adapter.ImplementParams{Context: pctx, Task: task, Plan: &plan, StepID: s.ID, Previous: execs})
			if err != nil {
				t.Fatalf("implement %s: %v", s.ID, err)
			}
			for _, c := range ir.Calls {
				if c.ID == "" || !slices.Contains(knownTools, c.Tool) {
					t.Errorf("call %+v has no id or an unknown tool", c.Redacted())
				}
				if (c.Tool == contract.ToolWriteFile || c.Tool == contract.ToolDeleteFile) && !task.Scope.InScope(c.Path) {
					t.Errorf("engine proposed %s outside the task scope: %q", c.Tool, c.Path)
				}
				execs = append(execs, contract.ExecutionRecord{TypeMeta: contract.Meta(contract.KindExecutionRecord),
					Call: c.Redacted(), Decision: contract.Decision{Allowed: true, Reason: "conformance"}, Status: contract.ExecApplied})
			}
			checkClaims(t, ir.Claims, nil)
		}
		if len(execs) == 0 {
			t.Fatal("engine proposed no tool calls for the fixture")
		}
		gates := []model.VerificationEvidence{{GateID: "test", Type: model.GateTypeTest, Command: "go test ./...",
			Required: true, Status: model.GateFailed, ExitCode: 1}}
		rr, err := eng.Review(bg, adapter.ReviewParams{Context: pctx, Task: task, Plan: &plan, Executions: execs,
			Verification: contract.VerificationSection{Gates: gates}})
		if err != nil {
			t.Fatal(err)
		}
		checkClaims(t, rr.Claims, gates)
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(bg)
		cancel()
		eng := start(t)
		began := time.Now()
		_, err := eng.Plan(ctx, adapter.PlanParams{Context: pctx, Task: task})
		if !errors.Is(err, contract.ErrCancelled) && !errors.Is(err, contract.ErrTimeout) && !errors.Is(err, context.Canceled) {
			t.Errorf("a cancelled request must fail with cancelled/timeout, got %v", err)
		}
		if time.Since(began) > 2*time.Second {
			t.Error("cancellation must be prompt")
		}
	})
}

// checkClaims enforces that claims are attributed, staged, and never cite
// a gate that did not pass.
func checkClaims(t *testing.T, claims []contract.Claim, gates []model.VerificationEvidence) {
	t.Helper()
	for _, c := range claims {
		if c.Source == "" || !slices.Contains(contract.Stages, c.Stage) || c.Statement == "" {
			t.Errorf("claim %+v lacks source, stage, or statement", c)
		}
		if c.VerifiedBy == "" {
			continue
		}
		ok := false
		for _, g := range gates {
			ok = ok || (g.GateID == c.VerifiedBy && g.Status == model.GatePassed)
		}
		if !ok {
			t.Errorf("claim %q cites gate %q, which did not pass", c.Statement, c.VerifiedBy)
		}
	}
}

// RunStdio runs the same suite with the engine behind adapter.Serve and the
// host's adapter.Client, proving the contract survives the process boundary.
func RunStdio(t *testing.T, newEngine func(t *testing.T) adapter.Engine) {
	t.Helper()
	Run(t, func(t *testing.T) adapter.Engine {
		return Pipe(t, newEngine(t))
	})
}

// Pipe connects a Client to eng through in-memory pipes and the real
// protocol; the session closes when the test ends.
func Pipe(t testing.TB, eng adapter.Engine) *adapter.Client {
	hostR, engW := io.Pipe()
	engR, hostW := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- adapter.Serve(context.Background(), eng, engR, engW); _ = engW.Close() }()
	c := adapter.NewClient(hostR, hostW)
	if _, err := c.Initialize(context.Background(), adapter.InitializeParams{ContractVersion: contract.ContractVersion}); err != nil {
		t.Fatalf("initialize over stdio: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Close()
		if err := <-done; err != nil {
			t.Errorf("serve: %v", err)
		}
	})
	return c
}
