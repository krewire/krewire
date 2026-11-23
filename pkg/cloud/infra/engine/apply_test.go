package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/cloud/infra/engine"
	"github.com/krewire/krewire/pkg/cloud/infra/store"
	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/model"
)

func TestApprovalBinding(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	p := f.plan(t, sample())
	if !f.e.RequiresApproval(p) {
		t.Fatal("high-risk plan must require approval")
	}
	ok := approval(p).Approval
	cases := map[string]struct {
		mutate func(a engine.Approval) *engine.Approval
		want   error
	}{
		"missing":  {func(engine.Approval) *engine.Approval { return nil }, engine.ErrApprovalRequired},
		"digest":   {func(a engine.Approval) *engine.Approval { a.PlanDigest = "sha256:other"; return &a }, engine.ErrApprovalMismatch},
		"env":      {func(a engine.Approval) *engine.Approval { a.Environment = "prod"; return &a }, engine.ErrApprovalMismatch},
		"approver": {func(a engine.Approval) *engine.Approval { a.Approver = ""; return &a }, engine.ErrApprovalMismatch},
		"expired": {func(a engine.Approval) *engine.Approval {
			a.ExpiresAt = f.e.Now().Add(-time.Minute)
			return &a
		}, engine.ErrApprovalExpired},
	}
	for name, c := range cases {
		_, err := f.e.Apply(ctx, p, engine.ApplyOptions{Approval: c.mutate(*ok)})
		if !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", name, c.want, err)
		}
	}
	if f.st.Writes != 0 || len(f.p.Calls) != 0 {
		t.Fatal("rejected approvals must not mutate")
	}
	if !strings.Contains(engine.ApprovalScope(p), p.Digest) {
		t.Fatal("approval scope must bind the digest")
	}

	// Low-risk plans run without approval for humans, never for agents.
	low := f.plan(t, sample()[2:])
	if f.e.RequiresApproval(low) {
		t.Fatal("low-risk human plan should not need approval")
	}
	f.e.Principal = engine.Principal{ID: "agent-7", Agent: true}
	if _, err := f.e.Apply(ctx, low, engine.ApplyOptions{}); !errors.Is(err, engine.ErrApprovalRequired) {
		t.Fatalf("agent apply without approval: %v", err)
	}
	self := approval(low)
	self.Approval.Approver = "agent-7"
	if _, err := f.e.Apply(ctx, low, self); !errors.Is(err, engine.ErrSelfApproval) {
		t.Fatalf("agent self-approval: %v", err)
	}
	if _, err := f.e.Apply(ctx, low, approval(low)); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsTamperedStaleAndForeignPlans(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	p := f.plan(t, sample())

	edited := *p
	edited.Actions = slices.Clone(p.Actions)
	edited.Actions[0].Op = engine.OpNoOp
	if _, err := f.e.Apply(ctx, &edited, approval(p)); !errors.Is(err, engine.ErrPlanTampered) {
		t.Fatalf("tampered: %v", err)
	}
	prod := engine.New(f.p, f.st, "prod")
	if _, err := prod.Apply(ctx, p, approval(p)); !errors.Is(err, engine.ErrWrongEnv) {
		t.Fatalf("wrong env: %v", err)
	}
	f.apply(t, f.plan(t, sample()[2:])) // state moves after p was planned
	_, err := f.e.Apply(ctx, p, approval(p))
	if !errors.Is(err, engine.ErrStalePlan) || errs.HintOf(err) == "" {
		t.Fatalf("stale: %v", err)
	}
}

func TestApplyIsLocked(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	p := f.plan(t, sample())
	lk, err := f.st.Lock(ctx, store.LockInfo{Who: "other", Operation: "apply"})
	if err != nil {
		t.Fatal(err)
	}
	var le *store.LockedError
	if _, err := f.e.Apply(ctx, p, approval(p)); !errors.As(err, &le) || le.Holder.Who != "other" {
		t.Fatalf("concurrent apply: %v", err)
	}
	if len(f.p.Calls) != 0 {
		t.Fatal("locked apply called the provider")
	}
	_ = lk.Unlock()
	f.apply(t, p)
}

func TestPartialFailureAndResume(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.p.LeakSecretsInErrors = true
	f.p.FailNext("create", "service/api", errors.New("quota exceeded"))
	p := f.plan(t, sample())
	res, err := f.e.Apply(ctx, p, approval(p))
	if !errors.Is(err, engine.ErrPartialApply) || errs.HintOf(err) == "" || res == nil || res.Succeeded {
		t.Fatalf("partial apply: %v %+v", err, res)
	}
	got := map[string]engine.ActionStatus{}
	for _, a := range res.Actions {
		got[a.Address] = a.Status
	}
	if got["bucket/assets"] != engine.StatusApplied || got["service/api"] != engine.StatusFailed || got["dns/www"] != engine.StatusSkipped {
		t.Fatalf("statuses: %v", got)
	}
	if len(res.Guidance) < 3 {
		t.Fatalf("guidance: %v", res.Guidance)
	}
	snap, _ := f.st.Read(ctx)
	rs, _ := snap.Find("service/api")
	if rs.Status != contract.StatusPending || rs.PendingOp != "create" {
		t.Fatalf("failed resource not journaled as pending: %+v", rs)
	}
	assertNoSecret(t, err.Error(), res, snap, f.e.NewEvidence("task-1", p, approval(p).Approval, res, nil))
	ev := f.e.NewEvidence("task-1", p, nil, res, nil).Verification()
	if ev.Status != model.GateFailed || !strings.Contains(ev.Output, "quota exceeded") {
		t.Fatalf("evidence: %+v", ev)
	}

	resume := f.plan(t, sample())
	if resume.Summary.NoOp != 1 || !resume.Actions[1].Pending {
		t.Fatalf("resume plan: %+v", resume.Actions)
	}
	res = f.apply(t, resume)
	if !res.Succeeded {
		t.Fatalf("resume failed: %+v", res)
	}
	if n := countCalls(f.p.Calls, "create bucket/assets"); n != 1 {
		t.Fatalf("completed action repeated %d times", n)
	}
	snap, _ = f.st.Read(ctx)
	api, _ := snap.Find("service/api")
	if api.Status != contract.StatusReady || api.Outputs["connection"] != contract.Redacted {
		t.Fatalf("resumed state: %+v", api)
	}
	assertNoSecret(t, snap)
	if f.e.NewEvidence("t", resume, nil, res, nil).Verification().Status != model.GatePassed {
		t.Fatal("success evidence must pass")
	}
}

func TestInterruptedUpdateAndDeleteResume(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.apply(t, f.plan(t, sample()))
	changed := sample()
	changed[1].Properties = map[string]any{"image": "app:2", "replicas": 2, "port": 8080}
	f.p.FailNext("update", "service/api", errors.New("timeout"))
	p := f.plan(t, changed)
	if _, err := f.e.Apply(ctx, p, approval(p)); err == nil {
		t.Fatal("injected update failure ignored")
	}
	resume := f.plan(t, changed)
	if a := resume.Actions[1]; a.Op != engine.OpUpdate || !a.Pending {
		t.Fatalf("resume of update: %+v", a)
	}
	f.apply(t, resume)

	f.p.FailNext("delete", "dns/www", errors.New("busy"))
	d, _ := f.e.PlanDestroy(ctx)
	if _, err := f.e.Apply(ctx, d, approval(d)); err == nil {
		t.Fatal("injected delete failure ignored")
	}
	f.p.Remove("dns/www") // someone removed it meanwhile: resume must tolerate absence
	d, _ = f.e.PlanDestroy(ctx)
	f.apply(t, d)
	if snap, _ := f.st.Read(ctx); len(snap.Resources) != 0 {
		t.Fatalf("destroy left state: %+v", snap.Resources)
	}
}

func TestApplyCanceled(t *testing.T) {
	f := newFixture()
	p := f.plan(t, sample())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := f.e.Apply(ctx, p, approval(p))
	if !errors.Is(err, engine.ErrApplyCanceled) || res.Succeeded || len(f.p.Calls) != 0 {
		t.Fatalf("canceled apply: %v %+v %v", err, res, f.p.Calls)
	}
}

func countCalls(calls []string, want string) int {
	n := 0
	for _, c := range calls {
		if c == want {
			n++
		}
	}
	return n
}

func assertNoSecret(t *testing.T, values ...any) {
	t.Helper()
	for _, v := range values {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), secretValue) {
			t.Fatalf("secret leaked into %T: %s", v, data)
		}
	}
}
