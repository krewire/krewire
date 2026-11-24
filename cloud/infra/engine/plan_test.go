package engine_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/cloud/infra/engine"
	"github.com/krewire/krewire/cloud/infra/providers/memory"
	"github.com/krewire/krewire/cloud/infra/store"
)

var update = flag.Bool("update", false, "rewrite golden plan files")

const secretValue = "s3cr3t-value"

func sample() []contract.Resource {
	return []contract.Resource{
		{Kind: memory.KindDNS, Name: "www", SchemaVersion: "v1", Properties: map[string]any{"target": "api"}, DependsOn: []string{"service/api"}},
		{Kind: memory.KindService, Name: "api", SchemaVersion: "v1", Properties: map[string]any{"image": "app:1", "replicas": 2, "port": 8080},
			Secrets: map[string]contract.SecretRef{"token": {Source: "env", Key: "API_TOKEN"}}, DependsOn: []string{"bucket/assets"}},
		{Kind: memory.KindBucket, Name: "assets", SchemaVersion: "v1", Properties: map[string]any{"region": "ap-southeast-1"}},
	}
}

type fixture struct {
	p  *memory.Provider
	st *store.Memory
	e  *engine.Engine
}

func newFixture() *fixture {
	f := &fixture{p: memory.New(), st: store.NewMemory()}
	f.e = engine.New(f.p, f.st, "dev")
	f.e.Secrets = engine.MapSecrets{"env:API_TOKEN": secretValue}
	f.e.Principal = engine.Principal{ID: "alice"}
	f.e.NewLineage = func() string { return "lineage-test" }
	f.e.Now = func() time.Time { return time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC) }
	return f
}

func approval(p *engine.Plan) engine.ApplyOptions {
	return engine.ApplyOptions{Approval: &engine.Approval{PlanDigest: p.Digest, Environment: p.Environment, Approver: "bob"}}
}

func (f *fixture) plan(t *testing.T, desired []contract.Resource) *engine.Plan {
	t.Helper()
	p, err := f.e.Plan(context.Background(), desired)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return p
}

func (f *fixture) apply(t *testing.T, p *engine.Plan) *engine.Result {
	t.Helper()
	res, err := f.e.Apply(context.Background(), p, approval(p))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return res
}

func golden(t *testing.T, name string, p *engine.Plan) {
	t.Helper()
	got, err := p.JSON()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", name+".golden.json")
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test -update to create)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("plan %s differs from golden:\n%s", name, got)
	}
}

func TestPlanGolden(t *testing.T) {
	f := newFixture()
	create := f.plan(t, sample())
	golden(t, "plan_create", create)
	if f.st.Writes != 0 || len(f.p.Calls) != 0 {
		t.Fatalf("plan mutated: writes=%d calls=%v", f.st.Writes, f.p.Calls)
	}
	f.apply(t, create)

	changed := sample()[1:] // drop dns/www
	changed[0].Properties = map[string]any{"image": "app:2", "replicas": 2, "port": 9090}
	changed = append(changed, contract.Resource{Kind: memory.KindBucket, Name: "logs", SchemaVersion: "v1",
		Properties: map[string]any{"region": "ap-southeast-1"}})
	mixed := f.plan(t, changed)
	golden(t, "plan_mixed", mixed)
	f.apply(t, mixed)

	destroy, err := f.e.PlanDestroy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "plan_destroy", destroy)
}

func TestPlanDeterministic(t *testing.T) {
	a, b := newFixture(), newFixture()
	rev := sample()
	rev[0], rev[2] = rev[2], rev[0]
	pa, pb := a.plan(t, sample()), b.plan(t, rev)
	if pa.Digest != pb.Digest {
		t.Fatal("declaration order changed the plan digest")
	}
	if pa.Actions[0].Address != "bucket/assets" || pa.Actions[2].Address != "dns/www" {
		t.Fatalf("not dependency ordered: %+v", pa.Actions)
	}
	if pa.MaxRisk() != engine.RiskHigh {
		t.Fatalf("dns create should be high risk, got %s", pa.MaxRisk())
	}
}

func TestValidationErrors(t *testing.T) {
	f := newFixture()
	cases := map[string][]contract.Resource{
		"duplicate":   {sample()[2], sample()[2]},
		"unknown":     {{Kind: "queue", Name: "q", SchemaVersion: "v1"}},
		"missing dep": {sample()[1]},
		"self dep":    {{Kind: memory.KindBucket, Name: "a", SchemaVersion: "v1", Properties: map[string]any{"region": "x"}, DependsOn: []string{"bucket/a"}}},
		"cycle": {
			{Kind: memory.KindBucket, Name: "a", SchemaVersion: "v1", Properties: map[string]any{"region": "x"}, DependsOn: []string{"bucket/b"}},
			{Kind: memory.KindBucket, Name: "b", SchemaVersion: "v1", Properties: map[string]any{"region": "x"}, DependsOn: []string{"bucket/a"}},
		},
		"inline secret": {{Kind: memory.KindDNS, Name: "d", SchemaVersion: "v1", Properties: map[string]any{"target": "x", "apiToken": secretValue}}},
		"bad name":      {{Kind: memory.KindBucket, Name: "a/b", SchemaVersion: "v1"}},
		"bad version":   {{Kind: memory.KindBucket, Name: "a", SchemaVersion: "v9", Properties: map[string]any{"region": "x"}}},
	}
	for name, desired := range cases {
		_, err := f.e.Plan(context.Background(), desired)
		var ve *engine.ValidationError
		if !errors.As(err, &ve) || len(ve.Problems) == 0 {
			t.Errorf("%s: want ValidationError, got %v", name, err)
		}
		if err != nil && strings.Contains(err.Error(), secretValue) {
			t.Errorf("%s: validation error leaked a value", name)
		}
	}
}
