package workflow_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/fakeengine"
	"github.com/krewire/krewire/pkg/toolchain/workflow"
)

// TestNoEngineStillVerifies: without an engine or credentials the agentic
// stages fail with engine_unavailable while deterministic verification runs.
func TestNoEngineStillVerifies(t *testing.T) {
	root, pctx, task := repo(t)
	r, err := workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Runner: fakeRunner{model.GatePassed}})
	if err != nil {
		t.Fatal(err)
	}
	checkReport(t, r, contract.OutcomeEngineUnavailable)
	if r.Stages[1].Error == nil || !errors.Is(r.Stages[1].Error, contract.ErrEngineUnavailable) {
		t.Errorf("plan stage: %+v", r.Stages[1])
	}
	if r.Stages[4].Status != contract.StageOK || !r.Verification.Verified || len(r.Verification.Gates) != 2 {
		t.Errorf("deterministic verification must run without an engine: %+v", r.Verification)
	}
}

func TestScopeAndApprovalBoundExecution(t *testing.T) {
	root, pctx, task := repo(t)
	task.Approval.Plan = false
	eng := &fakeengine.Engine{Files: goodFiles, Extra: []contract.ToolCall{
		{ID: "x1", Tool: contract.ToolWriteFile, Path: "cmd/evil/main.go", Content: "package main\n"},
		{ID: "x2", Tool: contract.ToolDeleteFile, Path: "pkg/greet/greet.go"},
		{ID: "x3", Tool: contract.ToolRunCommand, Argv: []string{"curl", "https://example.com"}},
		{ID: "x4", Tool: contract.ToolFetch, URL: "https://example.com"},
	}}
	r, err := workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Engine: eng, Runner: fakeRunner{model.GatePassed}})
	if err != nil {
		t.Fatal(err)
	}
	checkReport(t, r, contract.OutcomeDenied)
	codes := map[string]string{}
	for _, x := range r.Executions {
		codes[x.Call.ID] = x.Decision.Code
		if x.Call.Content != "" {
			t.Error("execution records keep only the content hash")
		}
	}
	want := map[string]string{"x1": "scope_violation", "x2": "approval_required", "x3": "scope_violation", "x4": "policy_denied"}
	for id, code := range want {
		if codes[id] != code {
			t.Errorf("%s: decision %q, want %q", id, codes[id], code)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "cmd/evil/main.go")); err == nil {
		t.Error("an out-of-scope write was executed")
	}
	if len(r.Findings) == 0 {
		t.Error("the engine review sees denials")
	}
}

func TestUnapprovedPlanStopsBeforeMutation(t *testing.T) {
	root, pctx, task := repo(t)
	r, err := workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Engine: &fakeengine.Engine{Files: goodFiles}, Secret: secret})
	if err != nil {
		t.Fatal(err)
	}
	checkReport(t, r, contract.OutcomeDenied)
	if r.PlanDigest == "" || r.Stages[2].Error == nil || r.Stages[2].Error.Code != contract.CodeApprovalRequired {
		t.Errorf("decompose stage must carry approval_required: %+v", r.Stages[2])
	}
	if len(r.Executions) != 0 || r.Stages[3].Status != contract.StageSkipped {
		t.Error("nothing may execute without plan approval")
	}
}

func TestCancellationAndBudget(t *testing.T) {
	root, pctx, task := repo(t)
	task.Approval.Plan = false
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	r, err := workflow.Run(ctx, workflow.Config{Root: root, Context: pctx, Task: task, Policy: contract.DefaultPolicy(),
		Engine: &fakeengine.Engine{Delay: 5 * time.Second}, Runner: fakeRunner{model.GatePassed}})
	if err != nil {
		t.Fatal(err)
	}
	checkReport(t, r, contract.OutcomeCancelled)

	task.Limits.MaxToolCalls = 1
	r, _ = workflow.Run(context.Background(), workflow.Config{Root: root, Context: pctx, Task: task,
		Policy: contract.DefaultPolicy(), Engine: &fakeengine.Engine{Files: goodFiles}, Runner: fakeRunner{model.GatePassed}})
	checkReport(t, r, contract.OutcomeDenied)
	if last := r.Executions[len(r.Executions)-1]; last.Decision.Reason != "tool call budget exhausted" {
		t.Errorf("budget: %+v", last.Decision)
	}
}

func TestInvalidInputs(t *testing.T) {
	_, pctx, task := repo(t)
	if _, err := workflow.Run(context.Background(), workflow.Config{}); !errors.Is(err, contract.ErrInvalidDocument) && err == nil {
		t.Error("missing inputs must fail")
	}
	task.Verification.Gates = []string{"lint"}
	if _, err := workflow.Run(context.Background(), workflow.Config{Context: pctx, Task: task}); !errors.Is(err, contract.ErrInvalidDocument) {
		t.Errorf("unknown gate: %v", err)
	}
	blocking := *pctx
	blocking.Diagnostics = []contract.Diagnostic{{Code: "KWC1001", Severity: contract.SeverityError}}
	task.Verification.Gates = []string{"test"}
	r, err := workflow.Run(context.Background(), workflow.Config{Context: &blocking, Task: task})
	if err != nil || r.Outcome != contract.OutcomeFailed {
		t.Errorf("blocking context: %v %+v", err, r)
	}
}
