package apphost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sampleYAML = `
name: shop
web:
  title: ignored by apphost
topology:
  version: 1
  resources:
    db:
      kind: database
      run: {command: ./db}
      endpoints: [{name: tcp, port: 0, scheme: postgres}]
    cache:
      kind: cache
      run: {command: ./cache}
      endpoints: [{name: tcp}]
    api:
      kind: service
      run: {command: ./api, args: [--serve]}
      endpoints: [{name: http, scheme: http}]
      references: [db, cache]
      env:
        PORT: ${self.port}
        DATABASE_URL: ${db.url}
        PRICE: $$5
      readiness: {http: {endpoint: http, path: /healthz}, interval: 50ms, timeout: 5s}
      restart: {policy: on-failure, maxRestarts: 2, backoff: 10ms}
      shutdownTimeout: 2s
    worker:
      kind: worker
      run: {command: ./worker}
      env: {CACHE: "${cache.tcp.host}:${cache.tcp.port}"}
    billing:
      kind: external
      external: {url: "https://billing.example.com"}
`

func mustParse(t *testing.T, y string) *Topology {
	t.Helper()
	top, err := Parse([]byte(y))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return top
}

func TestParseNormalizesAndOrders(t *testing.T) {
	top := mustParse(t, sampleYAML)
	api := top.Resources["api"]
	if api.Name != "api" || api.Restart.MaxRestarts != 2 || api.ShutdownTimeout.String() != "2s" {
		t.Fatalf("api not decoded: %+v", api)
	}
	if got := top.Resources["db"].Readiness; got == nil || got.TCP != "tcp" {
		t.Fatalf("default tcp readiness missing: %+v", got)
	}
	if top.Resources["billing"].Readiness != nil {
		t.Fatal("external resources get no default probe")
	}
	if got := top.Resources["worker"].Dependencies(); !reflect.DeepEqual(got, []string{"cache"}) {
		t.Fatalf("env expressions imply dependencies: %v", got)
	}
	want := [][]string{{"billing", "cache", "db"}, {"api", "worker"}}
	if got := top.Levels(); !reflect.DeepEqual(got, want) {
		t.Fatalf("levels = %v, want %v", got, want)
	}
	plan, err := NewPlan(top)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Stop, []string{"worker", "api", "db", "cache", "billing"}) {
		t.Fatalf("stop order = %v", plan.Stop)
	}
	text := plan.Explain()
	if !strings.Contains(text, "wave 1: start api (service via process) after cache, db, ready when GET http/healthz succeeds") {
		t.Fatalf("explain:\n%s", text)
	}
	if _, err := json.Marshal(plan); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(top.Dependents("cache"), []string{"api", "worker"}) {
		t.Fatalf("dependents = %v", top.Dependents("cache"))
	}
}

func TestLoadFileAndErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "krewire.yaml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("missing file must fail")
	}
	if _, err := Parse([]byte("name: x\n")); err == nil || !strings.Contains(err.Error(), "topology") {
		t.Fatalf("missing section: %v", err)
	}
	if _, err := Parse([]byte("topology: [\n")); err == nil {
		t.Fatal("bad yaml must fail")
	}
	if _, err := Parse([]byte("topology: {version: 1, resources: {a: {kind: service, run: {command: x}, shutdownTimeout: soon}}}")); err == nil {
		t.Fatal("bad duration must fail")
	}
}

func TestValidateReportsEveryIssue(t *testing.T) {
	cases := []struct {
		name, yaml, code string
	}{
		{"version", "version: 2\nresources: {a: {kind: service, run: {command: x}}}", CodeVersion},
		{"empty", "version: 1\nresources: {}", CodeName},
		{"name", "version: 1\nresources: {Bad: {kind: service, run: {command: x}}}", CodeName},
		{"kind", "version: 1\nresources: {a: {kind: blob, run: {command: x}}}", CodeKind},
		{"no runner", "version: 1\nresources: {a: {kind: service}}", CodeRunner},
		{"two runners", "version: 1\nresources: {a: {kind: service, run: {command: x}, external: {url: y}}}", CodeRunner},
		{"no command", "version: 1\nresources: {a: {kind: service, run: {}}}", CodeRunner},
		{"no image", "version: 1\nresources: {a: {kind: service, container: {}}}", CodeRunner},
		{"external no addr", "version: 1\nresources: {a: {kind: external, external: {}}}", CodeRunner},
		{"missing dep", "version: 1\nresources: {a: {kind: service, run: {command: x}, dependsOn: [ghost]}}", CodeMissingDep},
		{"self dep", "version: 1\nresources: {a: {kind: service, run: {command: x}, dependsOn: [a]}}", CodeCycle},
		{"cycle", "version: 1\nresources: {a: {kind: service, run: {command: x}, dependsOn: [b]}, b: {kind: service, run: {command: x}, references: [a]}}", CodeCycle},
		{"ports", "version: 1\nresources: {a: {kind: service, run: {command: x}, endpoints: [{name: h, port: 8080}]}, b: {kind: service, run: {command: x}, endpoints: [{name: h, port: 8080}]}}", CodePortCollision},
		{"dup endpoint", "version: 1\nresources: {a: {kind: service, run: {command: x}, endpoints: [{name: h}, {name: h}]}}", CodeEndpoint},
		{"port range", "version: 1\nresources: {a: {kind: service, run: {command: x}, endpoints: [{name: h, port: 70000}]}}", CodeEndpoint},
		{"ext dyn port", "version: 1\nresources: {a: {kind: external, external: {url: u}, endpoints: [{name: h}]}}", CodeEndpoint},
		{"container target", "version: 1\nresources: {a: {kind: cache, container: {image: redis}, endpoints: [{name: h}]}}", CodeEndpoint},
		{"env unknown", "version: 1\nresources: {a: {kind: service, run: {command: x}, env: {U: '${ghost.url}'}}}", CodeEnvReference},
		{"env endpoint", "version: 1\nresources: {a: {kind: service, run: {command: x}, env: {U: '${self.http.port}'}}}", CodeEnvReference},
		{"env prop", "version: 1\nresources: {a: {kind: service, run: {command: x}, env: {U: '${self.pid}'}}}", CodeEnvReference},
		{"env shape", "version: 1\nresources: {a: {kind: service, run: {command: x}, env: {U: '${self}'}}}", CodeEnvReference},
		{"env open", "version: 1\nresources: {a: {kind: service, run: {command: x}, env: {U: '${self.port'}}}", CodeEnvReference},
		{"probe endpoint", "version: 1\nresources: {a: {kind: service, run: {command: x}, readiness: {tcp: nope}}}", CodeProbe},
		{"probe http", "version: 1\nresources: {a: {kind: service, run: {command: x}, readiness: {http: {endpoint: nope}}}}", CodeProbe},
		{"probe exec", "version: 1\nresources: {a: {kind: service, run: {command: x}, readiness: {exec: {}}}}", CodeProbe},
		{"probe none", "version: 1\nresources: {a: {kind: service, run: {command: x}, readiness: {interval: 1s}}}", CodeProbe},
		{"restart", "version: 1\nresources: {a: {kind: service, run: {command: x}, restart: {policy: sometimes}}}", CodeRestart},
		{"restart neg", "version: 1\nresources: {a: {kind: service, run: {command: x}, restart: {maxRestarts: -1}}}", CodeRestart},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse([]byte("topology:\n" + indent(c.yaml)))
			ve, ok := err.(*ValidationError)
			if !ok || !ve.Has(c.code) {
				t.Fatalf("want %s, got %v", c.code, err)
			}
			if !strings.Contains(ve.Error(), c.code) {
				t.Fatalf("message lacks code: %s", ve.Error())
			}
		})
	}
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}

func TestExpandExpressions(t *testing.T) {
	top := mustParse(t, sampleYAML)
	top.Resources["db"].Endpoints[0].Port = 5432
	top.Resources["api"].Endpoints[0].Port = 8080
	find := func(n string) (*Resource, bool) { r, ok := top.Resources[n]; return r, ok }
	api := top.Resources["api"]
	for in, want := range map[string]string{
		"${db.url}":               "postgres://127.0.0.1:5432",
		"${self.port}":            "8080",
		"${db.tcp.host}:x":        "127.0.0.1:x",
		"${billing.url}":          "https://billing.example.com",
		"$$5 and $x and $":        "$5 and $x and $",
		"${self.http.url}/health": "http://127.0.0.1:8080/health",
	} {
		got, err := expand(in, api, find)
		if err != nil || got != want {
			t.Errorf("expand(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"${ghost.url}", "${db.nope.port}", "${db.pid}", "${db"} {
		if _, err := expand(bad, api, find); err == nil {
			t.Errorf("expand(%q) must fail", bad)
		}
	}
}

func TestValidateProposalDiffs(t *testing.T) {
	cur := mustParse(t, sampleYAML)
	next := cur.Clone()
	delete(next.Resources, "billing")
	next.Resources["worker"].Env["EXTRA"] = "1"
	next.Resources["queue"] = &Resource{Kind: KindQueue, Run: &RunSpec{Command: "./q"}}
	p, err := ValidateProposal(cur, next)
	if err != nil {
		t.Fatal(err)
	}
	want := Diff{Added: []string{"queue"}, Removed: []string{"billing"}, Changed: []string{"worker"}}
	if !reflect.DeepEqual(p.Diff, want) {
		t.Fatalf("diff = %+v, want %+v", p.Diff, want)
	}
	if same, _ := ValidateProposal(cur, cur); !same.Diff.Empty() {
		t.Fatalf("identical proposal must have empty diff: %+v", same.Diff)
	}
	if fresh, _ := ValidateProposal(nil, cur); len(fresh.Diff.Added) != 5 {
		t.Fatalf("nil current adds everything: %+v", fresh.Diff)
	}
	next.Resources["api"].DependsOn = []string{"ghost"}
	if _, err := ValidateProposal(cur, next); err == nil {
		t.Fatal("invalid proposal must be rejected before execution")
	}
}

func TestDurationJSON(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`"1.5s"`), &d); err != nil || d.D().String() != "1.5s" {
		t.Fatalf("%v %v", d, err)
	}
	if err := json.Unmarshal([]byte(`3`), &d); err == nil {
		t.Fatal("non-string duration must fail")
	}
	if err := json.Unmarshal([]byte(`"x"`), &d); err == nil {
		t.Fatal("bad duration must fail")
	}
	b, _ := json.Marshal(Duration(DefaultProbeInterval))
	if string(b) != `"200ms"` {
		t.Fatalf("got %s", b)
	}
}
