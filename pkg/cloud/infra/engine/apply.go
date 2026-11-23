package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/cloud/infra/store"
	"github.com/krewire/krewire/pkg/errs"
)

// Apply errors.
var (
	ErrPlanTampered  = errors.New("infra: plan digest does not match its contents")
	ErrStalePlan     = errors.New("infra: plan is stale")
	ErrWrongEnv      = errors.New("infra: plan targets another environment")
	ErrPartialApply  = errors.New("infra: apply stopped after a failed action")
	ErrApplyCanceled = errors.New("infra: apply canceled")
)

// ActionStatus is the outcome of one action during apply.
type ActionStatus string

// Action outcomes.
const (
	StatusApplied   ActionStatus = "applied"
	StatusUnchanged ActionStatus = "unchanged"
	StatusFailed    ActionStatus = "failed"
	StatusSkipped   ActionStatus = "skipped"
)

// ActionResult records what happened to one planned action.
type ActionResult struct {
	Address string       `json:"address"`
	Op      Op           `json:"op"`
	Status  ActionStatus `json:"status"`
	Error   string       `json:"error,omitempty"`
}

// Result is the structured outcome of Apply.
type Result struct {
	PlanDigest  string         `json:"planDigest"`
	Environment string         `json:"environment"`
	Mode        Mode           `json:"mode"`
	StartSerial uint64         `json:"startSerial"`
	EndSerial   uint64         `json:"endSerial"`
	Succeeded   bool           `json:"succeeded"`
	Actions     []ActionResult `json:"actions"`
	// Guidance tells an operator (or agent) how to recover.
	Guidance []string `json:"guidance,omitempty"`
}

// ApplyOptions carries per-apply inputs.
type ApplyOptions struct {
	Approval *Approval
}

// Apply executes a reviewed plan. It verifies the digest, enforces approval,
// takes the state lock, re-plans from current state, and refuses to proceed
// unless the re-plan is identical. Each mutation is journaled: the resource is
// recorded as pending before the provider call and as ready (or removed)
// after it, so an interruption always leaves state the next plan can resume.
func (e *Engine) Apply(ctx context.Context, p *Plan, opts ApplyOptions) (*Result, error) {
	if d, err := p.ComputeDigest(); err != nil || d != p.Digest {
		return nil, ErrPlanTampered
	}
	if p.Environment != e.Environment {
		return nil, fmt.Errorf("%w: plan %q, engine %q", ErrWrongEnv, p.Environment, e.Environment)
	}
	if err := e.authorize(p, opts.Approval); err != nil {
		return nil, err
	}
	lock, err := e.Store.Lock(ctx, store.LockInfo{
		Who: e.Principal.ID, Operation: string(p.Mode), PlanDigest: p.Digest, Created: e.now(),
	})
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	snap, err := e.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if err := e.replanMatches(ctx, snap, p); err != nil {
		return nil, err
	}
	run := &applyRun{e: e, lock: lock, snap: snap, secrets: newScrubber(e.Secrets)}
	if run.snap.Lineage == "" {
		run.snap.Lineage = store.NewLockID()
	}
	res := &Result{PlanDigest: p.Digest, Environment: p.Environment, Mode: p.Mode, StartSerial: snap.Serial}
	var failure error
	for _, a := range p.Actions {
		if failure != nil {
			res.Actions = append(res.Actions, ActionResult{Address: a.Address, Op: a.Op, Status: StatusSkipped})
			continue
		}
		if a.Op == OpNoOp {
			res.Actions = append(res.Actions, ActionResult{Address: a.Address, Op: a.Op, Status: StatusUnchanged})
			continue
		}
		if err := ctx.Err(); err != nil {
			failure = fmt.Errorf("%w: %v", ErrApplyCanceled, err)
			res.Actions = append(res.Actions, ActionResult{Address: a.Address, Op: a.Op, Status: StatusSkipped})
			continue
		}
		if err := run.execute(ctx, a); err != nil {
			msg := run.secrets.scrub(err.Error())
			failure = fmt.Errorf("%w: %s %s: %s", ErrPartialApply, a.Op, a.Address, msg)
			res.Actions = append(res.Actions, ActionResult{Address: a.Address, Op: a.Op, Status: StatusFailed, Error: msg})
			continue
		}
		res.Actions = append(res.Actions, ActionResult{Address: a.Address, Op: a.Op, Status: StatusApplied})
	}
	res.EndSerial = run.snap.Serial
	res.Succeeded = failure == nil
	if failure != nil {
		res.Guidance = guidance(res)
		return res, errs.WithHint(failure, res.Guidance[0])
	}
	return res, nil
}

// replanMatches recomputes p under the lock and compares digests.
func (e *Engine) replanMatches(ctx context.Context, snap contract.Snapshot, p *Plan) error {
	var desired []contract.Resource
	for _, a := range p.Actions {
		if a.Resource != nil {
			desired = append(desired, *a.Resource)
		}
	}
	var drift *DriftReport
	if p.Mode == ModeReconcile {
		var err error
		if drift, err = e.inspect(ctx, snap); err != nil {
			return err
		}
	}
	again, err := e.planFrom(snap, p.Mode, desired, drift)
	if err != nil {
		return err
	}
	if again.Digest != p.Digest {
		return errs.WithHint(fmt.Errorf("%w: state or provider changed since planning (serial %d -> %d)",
			ErrStalePlan, p.BaseSerial, snap.Serial), "re-run plan, review the new actions, and approve again")
	}
	return nil
}

type applyRun struct {
	e       *Engine
	lock    store.Lock
	snap    contract.Snapshot
	secrets *scrubber
}

// commit writes the next snapshot after mutate.
func (r *applyRun) commit(ctx context.Context, mutate func(*contract.Snapshot)) error {
	next := r.snap.Clone()
	mutate(&next)
	next.Serial = r.snap.Serial + 1
	next.Environment = r.e.Environment
	next.Provider = r.e.Provider.Capabilities().Provider
	written, err := r.e.Store.Write(ctx, r.lock, next)
	if err != nil {
		return fmt.Errorf("record state: %w", err)
	}
	r.snap = written
	return nil
}

func (r *applyRun) execute(ctx context.Context, a Action) error {
	prior, hasPrior := r.snap.Find(a.Address)
	journal := prior
	if a.Resource != nil {
		journal.Resource = *a.Resource
	}
	journal.Status, journal.PendingOp = contract.StatusPending, string(a.Op)
	if hasPrior && prior.Status == contract.StatusPending {
		journal.PendingOp = prior.PendingOp
	}
	if err := r.commit(ctx, func(s *contract.Snapshot) { s.Put(journal) }); err != nil {
		return err
	}
	obs, gone, err := r.mutate(ctx, a, prior, hasPrior)
	if err != nil {
		return err
	}
	if gone {
		return r.commit(ctx, func(s *contract.Snapshot) { s.Remove(a.Address) })
	}
	schema, _ := r.e.Provider.Capabilities().Kind(a.Resource.Kind)
	done := contract.ResourceState{Resource: *a.Resource, ID: obs.ID, Status: contract.StatusReady,
		Outputs: redactOutputs(schema, obs.Outputs)}
	return r.commit(ctx, func(s *contract.Snapshot) { s.Put(done) })
}

// mutate performs the provider calls for one action. gone reports that the
// object no longer exists and should leave state.
func (r *applyRun) mutate(ctx context.Context, a Action, prior contract.ResourceState, hasPrior bool) (contract.Observed, bool, error) {
	p := r.e.Provider
	if a.Pending && hasPrior {
		_, err := p.Read(ctx, prior)
		exists := err == nil
		if err != nil && !errors.Is(err, contract.ErrNotFound) {
			return contract.Observed{}, false, fmt.Errorf("re-read interrupted object: %w", err)
		}
		switch {
		case a.Resource == nil && exists:
			return contract.Observed{}, true, p.Delete(ctx, prior)
		case a.Resource == nil:
			return contract.Observed{}, true, nil
		case exists:
			obs, err := p.Update(ctx, *a.Resource, prior, r.secrets)
			return obs, false, err
		}
		obs, err := p.Create(ctx, *a.Resource, r.secrets)
		return obs, false, err
	}
	switch a.Op {
	case OpCreate:
		obs, err := p.Create(ctx, *a.Resource, r.secrets)
		return obs, false, err
	case OpUpdate:
		obs, err := p.Update(ctx, *a.Resource, prior, r.secrets)
		return obs, false, err
	case OpReplace:
		if err := p.Delete(ctx, prior); err != nil {
			return contract.Observed{}, false, err
		}
		obs, err := p.Create(ctx, *a.Resource, r.secrets)
		return obs, false, err
	case OpDelete:
		return contract.Observed{}, true, p.Delete(ctx, prior)
	}
	return contract.Observed{}, false, fmt.Errorf("unknown op %q", a.Op)
}

func redactOutputs(schema contract.KindSchema, out map[string]any) map[string]any {
	if len(out) == 0 {
		return nil
	}
	red := make(map[string]any, len(out))
	for k, v := range out {
		red[k] = v
		if contract.IsSensitiveName(k) || containsString(schema.SensitiveOutputs, k) {
			red[k] = contract.Redacted
		}
	}
	m, _ := canonicalMap(red)
	return m
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// guidance produces operator instructions for a failed apply.
func guidance(res *Result) []string {
	var failed string
	for _, a := range res.Actions {
		if a.Status == StatusFailed {
			failed = a.Address
		}
	}
	g := []string{
		"fix the cause, then run plan and apply again; run inspect to compare recorded and actual state",
		"state was saved at serial " + fmt.Sprint(res.EndSerial) + "; completed actions are recorded and will not be repeated",
	}
	if failed != "" {
		g = append(g, failed+" is recorded as pending; the next plan re-reads it and resumes the interrupted operation")
	}
	return g
}
