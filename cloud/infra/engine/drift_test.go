package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krewire/krewire/cloud/infra/engine"
)

func statuses(r *engine.DriftReport) map[string]engine.DriftStatus {
	out := map[string]engine.DriftStatus{}
	for _, e := range r.Entries {
		out[e.Address] = e.Status
	}
	return out
}

func TestInspectClassifiesDrift(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.apply(t, f.plan(t, sample()))
	writes := f.st.Writes

	rep, err := f.e.Inspect(ctx)
	if err != nil || rep.Drifted {
		t.Fatalf("clean inspect: %+v %v", rep, err)
	}
	f.p.Tamper("service/api", "replicas", 5)
	f.p.Remove("dns/www")
	f.p.FailNext("read", "bucket/assets", errors.New("permission denied"))
	rep, err = f.e.Inspect(ctx)
	if err != nil {
		t.Fatalf("drift must be a report, not an error: %v", err)
	}
	want := map[string]engine.DriftStatus{
		"service/api": engine.DriftModified, "dns/www": engine.DriftMissing, "bucket/assets": engine.DriftUnreadable,
	}
	for addr, s := range want {
		if statuses(rep)[addr] != s {
			t.Errorf("%s: %s, want %s", addr, statuses(rep)[addr], s)
		}
	}
	if f.st.Writes != writes || countCalls(f.p.Calls, "update service/api") != 0 {
		t.Fatal("inspect mutated state or objects")
	}
	assertNoSecret(t, rep)
}

func TestReconcileRestoresDesiredState(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.apply(t, f.plan(t, sample()))
	f.p.Tamper("service/api", "image", "evil:latest")
	f.p.Remove("dns/www")
	rep, _ := f.e.Inspect(ctx)

	if _, err := f.e.PlanReconcile(ctx, sample(), nil); !errors.Is(err, engine.ErrStaleDrift) {
		t.Fatalf("nil report accepted: %v", err)
	}
	p, err := f.e.PlanReconcile(ctx, sample(), rep)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != engine.ModeReconcile || p.Summary.Update != 1 || p.Summary.Create != 1 {
		t.Fatalf("reconcile plan: %+v", p.Summary)
	}

	f.p.Tamper("service/api", "image", "evil:v2") // actual moves after review
	if _, err := f.e.Apply(ctx, p, approval(p)); !errors.Is(err, engine.ErrStalePlan) {
		t.Fatalf("reconcile against moved actual state: %v", err)
	}
	rep, _ = f.e.Inspect(ctx)
	p, _ = f.e.PlanReconcile(ctx, sample(), rep)
	f.apply(t, p)
	if rep, _ = f.e.Inspect(ctx); rep.Drifted {
		t.Fatalf("drift after reconcile: %+v", rep)
	}
}

func TestAcceptDrift(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.apply(t, f.plan(t, sample()))
	f.p.Tamper("service/api", "replicas", 5)
	f.p.Remove("dns/www")
	rep, _ := f.e.Inspect(ctx)

	stale := *rep
	stale.Digest = "sha256:old"
	if _, err := f.e.AcceptDrift(ctx, &stale, engine.ApplyOptions{}); !errors.Is(err, engine.ErrStaleDrift) {
		t.Fatalf("stale accept: %v", err)
	}
	agent := newFixture()
	agent.e.Store, agent.e.Provider = f.st, f.p
	agent.e.Principal = engine.Principal{ID: "agent", Agent: true}
	if _, err := agent.e.AcceptDrift(ctx, rep, engine.ApplyOptions{}); !errors.Is(err, engine.ErrApprovalRequired) {
		t.Fatalf("agent accept without approval: %v", err)
	}
	if _, err := f.e.AcceptDrift(ctx, rep, engine.ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	if rep, _ = f.e.Inspect(ctx); rep.Drifted {
		t.Fatalf("drift after accept: %+v", rep)
	}
	p := f.plan(t, sample())
	if p.Summary.Create != 1 || p.Summary.Update != 1 {
		t.Fatalf("plan after accept should restore desired: %+v", p.Summary)
	}
}

func TestInspectReportsPendingSeparately(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.p.FailNext("create", "bucket/assets", errors.New("boom"))
	p := f.plan(t, sample())
	_, _ = f.e.Apply(ctx, p, approval(p))
	rep, err := f.e.Inspect(ctx)
	if err != nil || statuses(rep)["bucket/assets"] != engine.DriftPending {
		t.Fatalf("pending not reported: %+v %v", rep, err)
	}
}
