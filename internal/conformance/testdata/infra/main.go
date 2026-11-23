// Command fixture is the infra workload conformance fixture: it validates the
// declarative resource topology in krewire.yaml, round-trips local state, and
// drives the pkg/cloud/infra engine (#138) through plan, apply and drift with
// the localfile reference provider, so no cloud credentials are needed. Each
// declared resource is materialised as rendered/<name>.json under the
// provider root.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/cloud/infra/engine"
	"github.com/krewire/krewire/pkg/cloud/infra/providers/localfile"
	"github.com/krewire/krewire/pkg/cloud/infra/schema"
	"github.com/krewire/krewire/pkg/cloud/infra/state"
	"github.com/krewire/krewire/pkg/cloud/infra/store"
	"github.com/krewire/krewire/pkg/config"
)

type resource struct {
	Compute  *schema.Compute  `json:"compute,omitempty"`
	Database *schema.Database `json:"database,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func load() (map[string]resource, error) {
	var raw struct {
		Infra struct {
			Resources map[string]any `yaml:"resources"`
		} `yaml:"infra"`
	}
	if err := config.Load("krewire.yaml", &raw); err != nil {
		return nil, err
	}
	if len(raw.Infra.Resources) == 0 {
		return nil, errors.New("krewire.yaml: infra.resources is empty")
	}
	// Resource schemas carry json tags; route through JSON to honour them.
	b, err := json.Marshal(raw.Infra.Resources)
	if err != nil {
		return nil, err
	}
	res := map[string]resource{}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(res))
	for n := range res {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		r := res[n]
		var v any = r.Compute
		if r.Database != nil {
			v = r.Database
		}
		if v == nil || (r.Compute == nil && r.Database == nil) {
			return nil, fmt.Errorf("resource %s: no supported kind (compute, database)", n)
		}
		if err := schema.Validate(v); err != nil {
			return nil, fmt.Errorf("resource %s: %w", n, err)
		}
	}
	return res, nil
}

func run(args []string) error {
	res, err := load()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "check" {
		fmt.Println("ok", len(res), "resources")
		return nil
	}
	if len(args) == 2 {
		switch args[0] {
		case "plan":
			return planCmd(res, args[1])
		case "apply":
			return applyCmd(res, args[1])
		case "drift":
			return driftCmd(res, args[1])
		}
	}
	if args[0] != "build" || len(args) < 2 {
		return fmt.Errorf("usage: fixture check|build <state.json>|plan <dir>|apply <dir>|drift <dir>")
	}
	if err := os.MkdirAll(".krewire", 0o755); err != nil {
		return err
	}
	st := state.New(&state.FileBackend{Path: args[1]})
	if err := st.Lock(state.LockInfo{ID: "conformance", Who: "fixture", Created: time.Now(), Operation: "build"}); err != nil {
		return err
	}
	if err := st.Lock(state.LockInfo{ID: "second"}); err == nil {
		return errors.New("state: second lock was granted while locked")
	}
	snapshot := map[string]any{}
	for n, r := range res {
		snapshot[n] = r
	}
	if err := st.Save(snapshot); err != nil {
		return err
	}
	back, err := st.Load()
	if err != nil {
		return err
	}
	if len(back) != len(res) {
		return fmt.Errorf("state: reloaded %d resources, saved %d", len(back), len(res))
	}
	if err := st.Unlock(); err != nil {
		return err
	}
	fmt.Println("state ok", len(back), "resources")
	return nil
}

// desired maps the validated topology onto localfile resources.
func desired(res map[string]resource) ([]contract.Resource, error) {
	out := []contract.Resource{{Kind: localfile.KindDirectory, Name: "rendered", SchemaVersion: "v1",
		Properties: map[string]any{"path": "rendered", "mode": "0755"}}}
	names := make([]string, 0, len(res))
	for n := range res {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := json.Marshal(res[n])
		if err != nil {
			return nil, err
		}
		out = append(out, contract.Resource{Kind: localfile.KindFile, Name: n, SchemaVersion: "v1",
			Properties: map[string]any{"path": "rendered/" + n + ".json", "content": string(b), "mode": "0644"},
			DependsOn:  []string{contract.Address(localfile.KindDirectory, "rendered")}})
	}
	return out, nil
}

// newEngine binds the localfile provider and file state under dir. Every
// change requires approval so the approval gate is exercised.
func newEngine(dir string) *engine.Engine {
	e := engine.New(localfile.New(filepath.Join(dir, "objects")), store.NewFile(filepath.Join(dir, "state")), "conformance")
	e.Principal = engine.Principal{ID: "fixture"}
	e.Policy = engine.Policy{ApprovalThreshold: engine.RiskLow}
	return e
}

func approve(p *engine.Plan) engine.ApplyOptions {
	return engine.ApplyOptions{Approval: &engine.Approval{PlanDigest: p.Digest, Environment: p.Environment,
		Approver: "reviewer", ApprovedAt: time.Now()}}
}

// planCmd requires a deterministic, digest-bound plan that creates every
// resource and writes nothing.
func planCmd(res map[string]resource, dir string) error {
	want, err := desired(res)
	if err != nil {
		return err
	}
	ctx := context.Background()
	e := newEngine(dir)
	p, err := e.Plan(ctx, want)
	if err != nil {
		return err
	}
	again, err := e.Plan(ctx, want)
	if err != nil {
		return err
	}
	if p.Digest == "" || again.Digest != p.Digest {
		return fmt.Errorf("plan: non-deterministic digest %q vs %q", p.Digest, again.Digest)
	}
	if p.Summary.Create != len(want) || !e.RequiresApproval(p) {
		return fmt.Errorf("plan: want %d creates requiring approval, got %+v", len(want), p.Summary)
	}
	if _, err := os.Stat(filepath.Join(dir, "objects")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("plan: planning touched the provider root")
	}
	fmt.Printf("plan ok create=%d digest=%s\n", p.Summary.Create, p.Digest)
	return nil
}

// applyCmd requires the approval gate to hold, the approved plan to apply,
// the objects to exist, and a re-plan to be a no-op.
func applyCmd(res map[string]resource, dir string) error {
	want, err := desired(res)
	if err != nil {
		return err
	}
	ctx := context.Background()
	e := newEngine(dir)
	p, err := e.Plan(ctx, want)
	if err != nil {
		return err
	}
	if _, err := e.Apply(ctx, p, engine.ApplyOptions{}); !errors.Is(err, engine.ErrApprovalRequired) {
		return fmt.Errorf("apply: unapproved plan was not refused: %v", err)
	}
	if r, err := e.Apply(ctx, p, approve(p)); err != nil || !r.Succeeded {
		return fmt.Errorf("apply: %v (result %+v)", err, r)
	}
	for n := range res {
		if _, err := os.Stat(filepath.Join(dir, "objects", "rendered", n+".json")); err != nil {
			return fmt.Errorf("apply: object for %s missing: %w", n, err)
		}
	}
	again, err := e.Plan(ctx, want)
	if err != nil {
		return err
	}
	if again.HasChanges() {
		return fmt.Errorf("apply: re-plan is not a no-op: %+v", again.Summary)
	}
	fmt.Printf("apply ok created=%d noop=%d\n", p.Summary.Create, again.Summary.NoOp)
	return nil
}

// driftCmd tampers with and removes applied objects, requires Inspect to
// classify both without mutating anything, then reconciles back to in-sync.
func driftCmd(res map[string]resource, dir string) error {
	want, err := desired(res)
	if err != nil {
		return err
	}
	ctx := context.Background()
	e := newEngine(dir)
	if rep, err := e.Inspect(ctx); err != nil || rep.Drifted {
		return fmt.Errorf("drift: applied state must start in sync: %v %+v", err, rep)
	}
	names := make([]string, 0, len(res))
	for n := range res {
		names = append(names, n)
	}
	sort.Strings(names)
	if len(names) < 2 {
		return errors.New("drift: fixture needs at least two resources")
	}
	modified, missing := names[0], names[1]
	if err := os.WriteFile(filepath.Join(dir, "objects", "rendered", modified+".json"), []byte(`{"tampered":true}`), 0o644); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, "objects", "rendered", missing+".json")); err != nil {
		return err
	}
	rep, err := e.Inspect(ctx)
	if err != nil {
		return err
	}
	got := map[string]engine.DriftStatus{}
	for _, en := range rep.Entries {
		got[en.Address] = en.Status
	}
	if !rep.Drifted || got[contract.Address(localfile.KindFile, modified)] != engine.DriftModified ||
		got[contract.Address(localfile.KindFile, missing)] != engine.DriftMissing {
		return fmt.Errorf("drift: misclassified: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "objects", "rendered", missing+".json")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("drift: inspect mutated objects")
	}
	p, err := e.PlanReconcile(ctx, want, rep)
	if err != nil {
		return err
	}
	if r, err := e.Apply(ctx, p, approve(p)); err != nil || !r.Succeeded {
		return fmt.Errorf("drift: reconcile apply: %v", err)
	}
	if rep, err = e.Inspect(ctx); err != nil || rep.Drifted {
		return fmt.Errorf("drift: still drifted after reconcile: %v %+v", err, rep)
	}
	fmt.Printf("drift ok modified=%s missing=%s reconciled\n", modified, missing)
	return nil
}
