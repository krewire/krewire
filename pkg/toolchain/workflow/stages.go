package workflow

import (
	"context"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// plan runs the agentic plan stage and the deterministic decompose stage.
func (r *run) plan(ctx context.Context) (contract.Plan, error) {
	var plan contract.Plan
	err := r.stage(contract.StagePlan, "engine", func() error {
		info, err := r.cfg.Engine.Initialize(ctx, adapter.InitializeParams{ContractVersion: contract.ContractVersion,
			Host: contract.EngineIdentity{Name: contract.Executable, ContractVersion: contract.ContractVersion}})
		if err != nil {
			return err
		}
		if !info.Ready {
			return contract.Errorf(contract.CodeEngineUnavailable, "engine %q is not ready: %s", info.Engine.Name, info.Reason)
		}
		r.actor = "engine:" + info.Engine.Name
		r.report.Engine = &info.Engine
		res, err := r.cfg.Engine.Plan(ctx, adapter.PlanParams{Context: r.cfg.Context, Task: r.cfg.Task})
		if err != nil {
			return err
		}
		plan = res.Plan
		r.report.Claims = append(r.report.Claims, res.Claims...)
		return nil
	})
	if last := &r.report.Stages[len(r.report.Stages)-1]; last.Stage == contract.StagePlan && r.report.Engine != nil {
		last.Actor = r.actor
	}
	if err != nil {
		return plan, err
	}
	err = r.stage(contract.StageDecompose, "toolchain", func() error {
		if err := plan.Validate(r.cfg.Task); err != nil {
			return err
		}
		plan.Digest = plan.ComputeDigest()
		r.report.PlanDigest = plan.Digest
		return nil
	})
	return plan, err
}

// implement asks the engine for each step's tool calls and executes the
// ones the policy allows. It reports whether any call was denied.
func (r *run) implement(ctx context.Context, plan contract.Plan) (bool, error) {
	limit := r.cfg.Policy.MaxToolCalls
	if t := r.cfg.Task.Limits.MaxToolCalls; t > 0 && (limit <= 0 || t < limit) {
		limit = t
	}
	timeout := time.Duration(r.cfg.Policy.ToolTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	denied, count := false, 0
	for i := 0; i < len(plan.Steps); i++ {
		step := plan.Steps[i]
		if err := ctx.Err(); err != nil {
			return denied, err
		}
		res, err := r.cfg.Engine.Implement(ctx, adapter.ImplementParams{Context: r.cfg.Context, Task: r.cfg.Task,
			Plan: &plan, StepID: step.ID, Previous: r.report.Executions})
		if err != nil {
			return denied, err
		}
		if res.More && len(res.Calls) > 0 && (limit <= 0 || count+len(res.Calls) < limit) {
			i-- // the engine continues this step after seeing the results
		}
		r.report.Claims = append(r.report.Claims, res.Claims...)
		for _, call := range res.Calls {
			rec := contract.ExecutionRecord{TypeMeta: contract.Meta(contract.KindExecutionRecord), Call: call.Redacted()}
			count++
			d := contract.Decision{Code: contract.CodePolicyDenied.String(), Reason: "tool call budget exhausted"}
			if limit <= 0 || count <= limit {
				d = r.auth.Authorize(call, r.cfg.Approvals)
				if !d.Allowed && d.ApprovalScope != "" && r.approval(ctx, d.ApprovalScope, call.Rationale) {
					d = r.auth.Authorize(call, r.cfg.Approvals)
				}
			}
			rec.Decision = d
			if !d.Allowed {
				rec.Status, denied = contract.ExecDenied, true
				r.report.Executions = append(r.report.Executions, rec)
				continue
			}
			r.execute(ctx, call, &rec, timeout)
			r.report.Executions = append(r.report.Executions, rec)
		}
	}
	return denied, nil
}

// execute performs one authorized call and fills rec.
func (r *run) execute(ctx context.Context, call contract.ToolCall, rec *contract.ExecutionRecord, timeout time.Duration) {
	start := time.Now()
	defer func() { rec.DurationMs = time.Since(start).Milliseconds() }()
	rec.Status = contract.ExecApplied
	switch call.Tool {
	case contract.ToolWriteFile, contract.ToolDeleteFile:
		m, err := applyFile(r.cfg.Root, call)
		if err != nil {
			rec.Status, rec.ExitCode, rec.Output = contract.ExecFailed, 1, err.Error()
			return
		}
		rec.Mutation = m
		r.report.Mutations = append(r.report.Mutations, *m)
	case contract.ToolRunCommand:
		code, out, err := runCommand(ctx, r.cfg.Root, call, timeout, r.cfg.Policy.MaxOutputBytes, r.cfg.Secret)
		rec.ExitCode, rec.Output = code, out
		if err != nil || code != 0 {
			rec.Status = contract.ExecFailed
		}
	default:
		rec.Status, rec.ExitCode, rec.Output = contract.ExecFailed, 1, string(call.Tool)+" is not executed by this host"
	}
}

// complete runs verify, review, and report after implementation.
func (r *run) complete(ctx context.Context, plan contract.Plan, implErr error, denied bool) (*contract.EvidenceReport, error) {
	switch {
	case implErr != nil:
		r.report.Outcome = outcomeFor(implErr)
	case denied:
		r.report.Outcome = contract.OutcomeDenied
	}
	r.reviewPlan = &plan
	return r.finish(ctx, true)
}

// finish records every remaining lifecycle stage in order: verification
// runs when asked (it never needs the engine), review runs only after a
// completed implementation, and the report is assembled and validated.
func (r *run) finish(ctx context.Context, verify bool) (*contract.EvidenceReport, error) {
	done := map[contract.Stage]bool{}
	for _, s := range r.report.Stages {
		done[s.Stage] = true
	}
	for _, s := range contract.Stages {
		switch {
		case done[s]:
		case s == contract.StageVerify && verify:
			_ = r.stage(s, "toolchain", func() error {
				r.report.Verification = Verify(ctx, r.cfg.Root, r.cfg.Context, r.cfg.Task.Verification.Gates,
					r.cfg.Runner, r.cfg.Ledger, r.cfg.Commit)
				return nil
			})
		case s == contract.StageReview && r.reviewPlan != nil:
			_ = r.stage(s, r.actor, func() error {
				res, err := r.cfg.Engine.Review(ctx, adapter.ReviewParams{Context: r.cfg.Context, Task: r.cfg.Task,
					Plan: r.reviewPlan, Executions: r.report.Executions, Verification: r.report.Verification})
				r.report.Findings = append(r.report.Findings, res.Findings...)
				r.report.Claims = append(r.report.Claims, res.Claims...)
				return err
			})
		case s == contract.StageReport:
			_ = r.stage(s, "toolchain", func() error {
				if r.report.Verification.Gates == nil {
					r.report.Verification.Gates = []model.VerificationEvidence{}
					r.report.Verification.Summary = "verification did not run"
				}
				r.report.Claims = trusted(r.report.Claims, r.report.Verification.Gates)
				if r.report.Outcome == "" {
					r.report.Outcome = contract.OutcomeVerificationFailed
					if r.report.Verification.Verified {
						r.report.Outcome = contract.OutcomeVerified
					}
				}
				return r.report.Validate()
			})
		default:
			r.skip(s)
		}
	}
	return r.report, nil
}
