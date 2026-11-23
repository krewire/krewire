// Package conformance is the reusable lifecycle conformance suite for infra
// providers (issue #138). A provider passes when the canonical engine can
// drive it through plan, apply, idempotent re-apply, update, drift
// inspection, reconciliation, and destroy, and when its primitive operations
// obey the contract.Provider rules.
//
// Use it from a provider's tests:
//
//	func TestConformance(t *testing.T) {
//		conformance.Run(t, func(t *testing.T) conformance.Harness { ... })
//	}
package conformance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/cloud/infra/engine"
	"github.com/krewire/krewire/pkg/cloud/infra/store"
)

// Harness describes how to exercise one provider.
type Harness struct {
	Provider contract.Provider
	// Resources is the initial desired state: at least two resources, one
	// depending on another.
	Resources []contract.Resource
	// Mutate returns desired state with an in-place updatable change.
	Mutate func([]contract.Resource) []contract.Resource
	// Secrets resolves secret references used by Resources.
	Secrets contract.SecretResolver
	// Tamper modifies the object at address out of band.
	Tamper func(t *testing.T, address string)
	// Remove deletes the object at address out of band.
	Remove func(t *testing.T, address string)
	// TamperAddress and RemoveAddress select the objects for drift tests.
	TamperAddress, RemoveAddress string
}

// Run executes the suite; newHarness must return a fresh, isolated harness.
func Run(t *testing.T, newHarness func(t *testing.T) Harness) {
	t.Run("Capabilities", func(t *testing.T) { capabilities(t, newHarness(t)) })
	t.Run("ReadMissing", func(t *testing.T) { readMissing(t, newHarness(t)) })
	t.Run("Lifecycle", func(t *testing.T) { lifecycle(t, newHarness(t)) })
	t.Run("PrimitiveIdempotence", func(t *testing.T) { primitives(t, newHarness(t)) })
}

func capabilities(t *testing.T, h Harness) {
	caps := h.Provider.Capabilities()
	if err := contract.CheckContract(caps); err != nil {
		t.Fatal(err)
	}
	if caps.Provider == "" || caps.ProviderVersion == "" || len(caps.Kinds) == 0 {
		t.Fatalf("capabilities must name provider, version, and kinds: %+v", caps)
	}
	seen := map[string]bool{}
	for _, k := range caps.Kinds {
		if k.Kind == "" || k.SchemaVersion == "" || seen[k.Kind] {
			t.Fatalf("kind %q must be unique and carry a schema version", k.Kind)
		}
		seen[k.Kind] = true
	}
}

func readMissing(t *testing.T, h Harness) {
	rs := contract.ResourceState{Resource: h.Resources[0]}
	if _, err := h.Provider.Read(context.Background(), rs); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("Read of a never-created object: want ErrNotFound, got %v", err)
	}
}

func newEngine(h Harness, st store.Store) *engine.Engine {
	e := engine.New(h.Provider, st, "conformance")
	e.Secrets = h.Secrets
	e.Principal = engine.Principal{ID: "conformance"}
	return e
}

// approve returns an approval for p, as a reviewer would.
func approve(p *engine.Plan) engine.ApplyOptions {
	return engine.ApplyOptions{Approval: &engine.Approval{PlanDigest: p.Digest, Environment: p.Environment,
		Approver: "reviewer", ApprovedAt: time.Now()}}
}

func must(t *testing.T) func(*engine.Plan, error) *engine.Plan {
	return func(p *engine.Plan, err error) *engine.Plan {
		t.Helper()
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		return p
	}
}

func mustApply(t *testing.T, e *engine.Engine, p *engine.Plan) {
	t.Helper()
	if res, err := e.Apply(context.Background(), p, approve(p)); err != nil {
		t.Fatalf("apply %s: %v (%+v)", p.Mode, err, res)
	}
}

func lifecycle(t *testing.T, h Harness) {
	ctx := context.Background()
	st := store.NewMemory()
	e := newEngine(h, st)

	p := must(t)(e.Plan(ctx, h.Resources))
	if st.Writes != 0 || p.Summary.Create != len(h.Resources) {
		t.Fatalf("first plan: %+v, writes %d", p.Summary, st.Writes)
	}
	again := must(t)(e.Plan(ctx, h.Resources))
	if again.Digest != p.Digest {
		t.Fatal("plan is not deterministic")
	}
	mustApply(t, e, p)

	idem := must(t)(e.Plan(ctx, h.Resources))
	if idem.HasChanges() {
		t.Fatalf("re-plan of unchanged desired state has changes: %+v", idem.Actions)
	}
	writes := st.Writes
	mustApply(t, e, idem)
	if st.Writes != writes {
		t.Fatal("apply of unchanged desired state wrote state")
	}

	mutated := h.Mutate(h.Resources)
	up := must(t)(e.Plan(ctx, mutated))
	if up.Summary.Update == 0 {
		t.Fatalf("Mutate produced no update: %+v", up.Summary)
	}
	mustApply(t, e, up)

	rep, err := e.Inspect(ctx)
	if err != nil || rep.Drifted {
		t.Fatalf("inspect after apply: %+v %v", rep, err)
	}
	h.Tamper(t, h.TamperAddress)
	h.Remove(t, h.RemoveAddress)
	rep, err = e.Inspect(ctx)
	if err != nil || !rep.Drifted {
		t.Fatalf("drift not detected: %+v %v", rep, err)
	}
	want := map[string]engine.DriftStatus{h.TamperAddress: engine.DriftModified, h.RemoveAddress: engine.DriftMissing}
	for _, ent := range rep.Entries {
		if w, ok := want[ent.Address]; ok && ent.Status != w {
			t.Fatalf("%s: drift %s, want %s", ent.Address, ent.Status, w)
		}
	}
	if idle := must(t)(e.Plan(ctx, mutated)); idle.HasChanges() {
		t.Fatal("plan without reconcile must reflect recorded state only")
	}
	rec := must(t)(e.PlanReconcile(ctx, mutated, rep))
	if !rec.HasChanges() {
		t.Fatal("reconcile plan has no changes")
	}
	mustApply(t, e, rec)
	if rep, err = e.Inspect(ctx); err != nil || rep.Drifted {
		t.Fatalf("drift remains after reconcile: %+v %v", rep, err)
	}

	d := must(t)(e.PlanDestroy(ctx))
	mustApply(t, e, d)
	snap, _ := st.Read(ctx)
	if len(snap.Resources) != 0 {
		t.Fatalf("destroy left state: %+v", snap.Resources)
	}
	for _, r := range mutated {
		if _, err := h.Provider.Read(ctx, contract.ResourceState{Resource: r}); !errors.Is(err, contract.ErrNotFound) {
			t.Fatalf("destroy left %s: %v", r.Address(), err)
		}
	}
}

func primitives(t *testing.T, h Harness) {
	ctx := context.Background()
	for _, r := range h.Resources {
		if _, err := h.Provider.Create(ctx, r, h.Secrets); err != nil {
			t.Fatalf("create %s: %v", r.Address(), err)
		}
		if _, err := h.Provider.Create(ctx, r, h.Secrets); err != nil {
			t.Fatalf("Create must converge an existing object %s: %v", r.Address(), err)
		}
	}
	for i := len(h.Resources) - 1; i >= 0; i-- {
		rs := contract.ResourceState{Resource: h.Resources[i]}
		for n := 0; n < 2; n++ {
			if err := h.Provider.Delete(ctx, rs); err != nil {
				t.Fatalf("Delete must be idempotent (%s, call %d): %v", rs.Address(), n+1, err)
			}
		}
	}
}
