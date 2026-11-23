// Package fakeengine is the reference adapter.Engine: a deterministic,
// LLM-free engine that proves the contract end to end. It plans one step
// per deliverable, proposes write_file calls with scripted content, and
// reviews by restating the host's evidence. Engines — Krew included — are
// held to the same conformance suite (pkg/toolchain/conformance).
package fakeengine

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// Name identifies the reference engine in reports.
const Name = "krew-reference"

// Engine is the reference engine. The zero value is usable.
type Engine struct {
	// Files maps repository-relative paths to the content the engine writes;
	// deliverables without an entry get a minimal Go package stub.
	Files map[string]string
	// Extra calls are appended to the first step, e.g. to exercise denials.
	Extra []contract.ToolCall
	// Delay makes every method wait (honouring ctx) to exercise timeouts.
	Delay time.Duration
}

var _ adapter.Engine = (*Engine)(nil)

func source() string { return "engine:" + Name }

func (e *Engine) wait(ctx context.Context) error {
	if e.Delay <= 0 {
		return ctx.Err()
	}
	select {
	case <-time.After(e.Delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Initialize implements adapter.Engine.
func (e *Engine) Initialize(ctx context.Context, _ adapter.InitializeParams) (adapter.InitializeResult, error) {
	return adapter.InitializeResult{
		Engine:  contract.EngineIdentity{Name: Name, Version: contract.ContractVersion, ContractVersion: contract.ContractVersion},
		Methods: []string{adapter.MethodPlan, adapter.MethodImplement, adapter.MethodReview},
		Ready:   true,
	}, ctx.Err()
}

// Plan implements adapter.Engine: one step per deliverable, in order.
func (e *Engine) Plan(ctx context.Context, p adapter.PlanParams) (adapter.PlanResult, error) {
	if err := e.wait(ctx); err != nil {
		return adapter.PlanResult{}, err
	}
	if p.Task == nil {
		return adapter.PlanResult{}, contract.Errorf(contract.CodeInvalidParams, "task is required")
	}
	targets := p.Task.Deliverables
	if len(targets) == 0 {
		return adapter.PlanResult{}, contract.Errorf(contract.CodeInvalidParams, "the reference engine plans deliverables; none given")
	}
	plan := contract.Plan{TypeMeta: contract.Meta(contract.KindPlan), TaskID: p.Task.ID,
		Summary: fmt.Sprintf("write %d deliverable(s) for %q", len(targets), p.Task.Title)}
	for i, f := range targets {
		step := contract.PlanStep{ID: fmt.Sprintf("s%d", i+1), Description: "write " + f,
			TargetFiles: []string{f}, Risk: p.Task.Risk, Rollback: "restore " + f + " from version control"}
		if i > 0 {
			step.DependsOn = []string{plan.Steps[i-1].ID}
		}
		plan.Steps = append(plan.Steps, step)
	}
	return adapter.PlanResult{Plan: plan, Claims: []contract.Claim{{Source: source(), Stage: contract.StagePlan,
		Statement: fmt.Sprintf("the plan covers all %d deliverables", len(targets))}}}, nil
}

// Implement implements adapter.Engine: one write_file per step target.
func (e *Engine) Implement(ctx context.Context, p adapter.ImplementParams) (adapter.ImplementResult, error) {
	if err := e.wait(ctx); err != nil {
		return adapter.ImplementResult{}, err
	}
	if p.Plan == nil {
		return adapter.ImplementResult{}, contract.Errorf(contract.CodeInvalidParams, "plan is required")
	}
	for i, s := range p.Plan.Steps {
		if s.ID != p.StepID {
			continue
		}
		var res adapter.ImplementResult
		for j, f := range s.TargetFiles {
			content, ok := e.Files[f]
			if !ok {
				content = fmt.Sprintf("package %s\n", path.Base(path.Dir(f)))
			}
			res.Calls = append(res.Calls, contract.ToolCall{ID: fmt.Sprintf("%s-c%d", s.ID, j+1), StepID: s.ID,
				Tool: contract.ToolWriteFile, Path: f, Content: content, Rationale: s.Description})
		}
		if i == 0 {
			res.Calls = append(res.Calls, e.Extra...)
		}
		res.Claims = []contract.Claim{{Source: source(), Stage: contract.StageImplement, Statement: s.Description + " is done"}}
		return res, nil
	}
	return adapter.ImplementResult{}, contract.Errorf(contract.CodeInvalidParams, "unknown step %q", p.StepID)
}

// Review implements adapter.Engine. It reports denied calls and links its
// claim to a passed gate only when the host's evidence shows one.
func (e *Engine) Review(ctx context.Context, p adapter.ReviewParams) (adapter.ReviewResult, error) {
	if err := e.wait(ctx); err != nil {
		return adapter.ReviewResult{}, err
	}
	var res adapter.ReviewResult
	for _, x := range p.Executions {
		if x.Status != contract.ExecApplied {
			res.Findings = append(res.Findings, contract.Finding{Severity: contract.SeverityWarning, Path: x.Call.Path,
				Message: fmt.Sprintf("%s was %s: %s", x.Call.Tool, x.Status, x.Decision.Reason)})
		}
	}
	claim := contract.Claim{Source: source(), Stage: contract.StageReview, Statement: "the change meets the task objective"}
	for _, g := range p.Verification.Gates {
		if g.Required && g.Status == model.GatePassed {
			claim.VerifiedBy = g.GateID
			break
		}
	}
	res.Claims = append(res.Claims, claim)
	return res, nil
}
