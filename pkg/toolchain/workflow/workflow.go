// Package workflow orchestrates the shared AI-native lifecycle — discover,
// plan, decompose, implement, verify, review, report — inside the toolchain.
// Deterministic stages run in-process; agentic stages go through an
// adapter.Engine. The toolchain alone authorizes and executes tool calls,
// runs verification gates, and writes the evidence report, so engine claims
// can never be mistaken for verified results.
package workflow

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/policy"
)

// Approver asks a human or control surface (e.g. Dash) to approve scope.
// It returns an ApprovalRecord signed with the host secret, or an error.
type Approver interface {
	Approve(ctx context.Context, scope, reason string) (contract.ApprovalRecord, error)
}

// Config is one workflow run.
type Config struct {
	Root      string
	Context   *contract.ProjectContext
	Task      *contract.TaskBrief
	Policy    contract.Policy
	Engine    adapter.Engine // nil: agentic stages report engine_unavailable
	Runner    GateRunner     // nil: ExecRunner
	Approver  Approver       // nil: only pre-collected Approvals count
	Approvals []contract.ApprovalRecord
	Secret    string // HMAC key verifying approvals; never emitted
	Commit    string
	Ledger    *model.VerificationLedger
}

type run struct {
	cfg    Config
	auth   *policy.Authorizer
	report *contract.EvidenceReport
	actor  string
	// reviewPlan is set once implementation ran, enabling the review stage.
	reviewPlan *contract.Plan
}

// stage times fn and records it; fn's error is recorded, not propagated.
func (r *run) stage(s contract.Stage, actor string, fn func() error) error {
	start := time.Now()
	err := fn()
	rec := contract.StageRecord{Stage: s, Status: contract.StageOK, Actor: actor, DurationMs: time.Since(start).Milliseconds()}
	if err != nil {
		rec.Status, rec.Error = contract.StageFailed, contract.AsError(err)
	}
	r.report.Stages = append(r.report.Stages, rec)
	return err
}

func (r *run) skip(stages ...contract.Stage) {
	for _, s := range stages {
		r.report.Stages = append(r.report.Stages, contract.StageRecord{Stage: s, Status: contract.StageSkipped, Actor: "toolchain"})
	}
}

// approval finds or requests a verified approval for scope.
func (r *run) approval(ctx context.Context, scope, reason string) bool {
	if rec, ok := r.auth.FindApproval(r.cfg.Approvals, scope); ok {
		r.addApproval(rec)
		return true
	}
	if r.cfg.Approver == nil {
		return false
	}
	rec, err := r.cfg.Approver.Approve(ctx, scope, reason)
	if err != nil || !r.auth.VerifyApproval(rec, scope) {
		return false
	}
	r.cfg.Approvals = append(r.cfg.Approvals, rec)
	rec.Verified = true
	r.addApproval(rec)
	return true
}

func (r *run) addApproval(rec contract.ApprovalRecord) {
	rec.Token = "" // the token is a credential-bearing MAC; reports keep the verdict only
	rec.TypeMeta = contract.Meta(contract.KindApprovalRecord)
	r.report.Approvals = append(r.report.Approvals, rec)
}

// Run executes the full lifecycle and returns the evidence report. The
// error is non-nil only for invalid inputs; every runtime outcome —
// denial, cancellation, missing engine, failed gates — is in the report.
func Run(ctx context.Context, cfg Config) (*contract.EvidenceReport, error) {
	if cfg.Context == nil || cfg.Task == nil {
		return nil, contract.Errorf(contract.CodeInvalidParams, "context and task are required")
	}
	if err := cfg.Context.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.Task.Validate(cfg.Context); err != nil {
		return nil, err
	}
	if cfg.Engine == nil {
		cfg.Engine = adapter.Unavailable{Reason: "no engine configured (set " + adapter.EnvEngine + ")"}
	}
	if cfg.Runner == nil {
		cfg.Runner = ExecRunner{Commit: cfg.Commit, Secrets: []string{cfg.Secret}}
	}
	if t := cfg.Task.Limits.TimeoutSec; t > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(t)*time.Second)
		defer cancel()
	}
	r := &run{cfg: cfg, actor: "engine:unknown",
		auth: &policy.Authorizer{Root: cfg.Root, Policy: cfg.Policy, Task: cfg.Task, Context: cfg.Context, Secret: cfg.Secret},
		report: &contract.EvidenceReport{TypeMeta: contract.Meta(contract.KindEvidenceReport), TaskID: cfg.Task.ID,
			ContextFingerprint: cfg.Context.Fingerprint, Stages: []contract.StageRecord{}}}
	_ = r.stage(contract.StageDiscover, "toolchain", func() error {
		if cfg.Context.Blocking() {
			return contract.Errorf(contract.CodeInvalidDocument, "project context has blocking diagnostics")
		}
		return nil
	})
	if cfg.Context.Blocking() {
		r.report.Outcome = contract.OutcomeFailed
		return r.finish(ctx, false)
	}
	plan, err := r.plan(ctx)
	if err != nil {
		r.report.Outcome = outcomeFor(err)
		return r.finish(ctx, true) // deterministic verification still runs
	}
	if cfg.Task.Approval.Plan && !r.approval(ctx, policy.PlanScope(cfg.Task.ID, plan.Digest), "approve plan "+plan.Digest) {
		r.report.Stages[len(r.report.Stages)-1].Status = contract.StageFailed
		r.report.Stages[len(r.report.Stages)-1].Error = contract.Errorf(contract.CodeApprovalRequired, "plan %s is not approved", plan.Digest)
		r.report.Outcome = contract.OutcomeDenied
		r.skip(contract.StageImplement, contract.StageVerify, contract.StageReview)
		return r.finish(ctx, false)
	}
	denied := false
	implErr := r.stage(contract.StageImplement, r.actor, func() error {
		var err error
		denied, err = r.implement(ctx, plan)
		return err
	})
	return r.complete(ctx, plan, implErr, denied)
}

// outcomeFor maps a stage error onto a report outcome.
func outcomeFor(err error) contract.Outcome {
	err = contract.AsError(err)
	switch {
	case errors.Is(err, contract.ErrEngineUnavailable):
		return contract.OutcomeEngineUnavailable
	case errors.Is(err, contract.ErrCancelled), errors.Is(err, contract.ErrTimeout):
		return contract.OutcomeCancelled
	case errors.Is(err, contract.ErrScopeViolation), errors.Is(err, contract.ErrPolicyDenied), errors.Is(err, contract.ErrApprovalRequired):
		return contract.OutcomeDenied
	}
	return contract.OutcomeFailed
}

// runCommand executes an authorized run_command tool call.
func runCommand(ctx context.Context, root string, call contract.ToolCall, timeout time.Duration, max int, secret string) (int, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, call.Argv[0], call.Argv[1:]...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), policy.Redact(string(out), max, secret), nil
	}
	return 0, policy.Redact(string(out), max, secret), err
}

// trusted clears claim citations of gates that did not pass: the host
// never lets an engine upgrade its own claims to verified results.
func trusted(claims []contract.Claim, gates []model.VerificationEvidence) []contract.Claim {
	out := make([]contract.Claim, 0, len(claims))
	for _, c := range claims {
		if c.VerifiedBy != "" && !slices.ContainsFunc(gates, func(g model.VerificationEvidence) bool {
			return g.GateID == c.VerifiedBy && g.Status == model.GatePassed
		}) {
			c.VerifiedBy = ""
		}
		out = append(out, c)
	}
	return out
}
