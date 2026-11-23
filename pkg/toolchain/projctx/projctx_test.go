package projctx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// fixture is a small but complete repository.
var fixture = map[string]string{
	"go.mod": "module example.com/demo\n\ngo 1.27.1\n\ntoolchain go1.27.1\n\nrequire (\n\tgopkg.in/yaml.v3 v3.0.1\n\tgolang.org/x/sys v0.49.0 // indirect\n)\n",
	"version.go": "package demo\n\nimport \"example.com/demo/pkg/version\"\n\n" +
		"var Version = version.MustParseVersion(\"0.3.0\")\n\nvar Tier = model.TierFree\n",
	"krewire.yaml": "project:\n  name: demo\n  version: \"0.3.0\"\ndatabase:\n  password: hunter2\n  url: postgres://app:s3cr3t@db/app\nlist:\n  - a\n",
	"AGENTS.md": "# Agents\n\n## Architecture Rules\n\n- pkg/a never imports pkg/b\n- keep SSOT\n\n## Quality Gates\n\n```bash\n" +
		"gofmt -l .   # must print nothing\ngo test ./...\nCGO_ENABLED=0 go build ./...\n```\n",
	"pkg/a/a.go":              "package a\n\nimport (\n\t\"fmt\"\n\t\"example.com/demo/pkg/b\"\n)\n\nvar _ = fmt.Sprint(b.X)\n",
	"pkg/b/b.go":              "package b\n\nconst X = 1\n",
	"pkg/b/b_test.go":         "package b\n",
	"pkg/b/testdata/x.go":     "package broken(\n",
	"nested/go.mod":           "module example.com/nested\n",
	"nested/n.go":             "package nested\n",
	"guard_test.go":           "package demo_test\n\n// TestLayering asserts that pkg/a stays below pkg/b. More text.\nfunc TestLayering(t *testing.T) {}\n\nfunc TestUndocumented(t *testing.T) {}\n",
	"docs/adr/ADR-T1C4A-x.md": "# ADR T1C4A — Unified toolchain\n",
	"docs/guide.md":           "Run `kiw context --json` then kiw verify.\nA `.kiw` component file is fine.\n",
	"README.md":               "Use krew.\n",
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func generate(t *testing.T, files map[string]string) (*contract.ProjectContext, []byte) {
	t.Helper()
	c, err := Generate(Options{Root: writeTree(t, files), ToolchainVersion: "0.9.0"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := contract.ValidateJSON(contract.KindProjectContext, data); err != nil {
		t.Fatalf("generated context violates its schema: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("generated context is invalid: %v", err)
	}
	return c, data
}

func codes(c *contract.ProjectContext) map[string]int {
	out := map[string]int{}
	for _, d := range c.Diagnostics {
		out[d.Code]++
		if _, ok := Codes[d.Code]; !ok {
			panic("unregistered code " + d.Code)
		}
	}
	return out
}

func TestGenerateFromSourceOfTruth(t *testing.T) {
	c, data := generate(t, fixture)
	if c.Module.Path != "example.com/demo" || c.Module.Version != "0.3.0" || c.Module.Tier != "free" {
		t.Errorf("module identity: %+v", c.Module)
	}
	if c.Compatibility.GoVersion != "1.27.1" || c.Compatibility.Toolchain != "go1.27.1" || len(c.Compatibility.Targets) == 0 {
		t.Errorf("compatibility: %+v", c.Compatibility)
	}
	if c.Workload.Kind != "library" || c.Workload.Maturity != "preview" {
		t.Errorf("workload: %+v", c.Workload)
	}
	if len(c.Dependencies) != 2 || !c.Dependencies[0].Indirect || c.Dependencies[0].Provenance.Line == 0 {
		t.Errorf("dependencies: %+v", c.Dependencies)
	}
	var paths []string
	for _, p := range c.Packages {
		paths = append(paths, p.ImportPath)
	}
	if strings.Join(paths, ",") != "example.com/demo,example.com/demo/pkg/a,example.com/demo/pkg/b" {
		t.Errorf("packages (nested modules and testdata excluded): %v", paths)
	}
	if c.Graph.Nodes != 3 || c.Graph.Edges != 2 || c.Packages[1].Imports[0] != "example.com/demo/pkg/b" {
		t.Errorf("graph: %+v", c.Graph)
	}
	ids := map[string]contract.Gate{}
	for _, g := range c.Gates {
		ids[g.ID] = g
	}
	if len(c.Gates) != 3 || !ids["fmt"].FailOnOutput || ids["build"].Env[0] != "CGO_ENABLED=0" || ids["test"].Provenance.Path != "AGENTS.md" {
		t.Errorf("gates: %+v", c.Gates)
	}
	rules := map[string]contract.Rule{}
	for _, r := range c.Rules {
		rules[r.ID] = r
	}
	if r := rules["TestLayering"]; r.EnforcedBy != "go test -run ^TestLayering$ ." || r.Statement != "TestLayering asserts that pkg/a stays below pkg/b." {
		t.Errorf("guard rule: %+v", r)
	}
	if _, ok := rules["TestUndocumented"]; ok {
		t.Error("undocumented tests are not rules")
	}
	if rules["ADR-T1C4A"].Category != "decision" || rules["agents-md/architecture/2"].Statement != "keep SSOT" {
		t.Errorf("rules: %+v", c.Rules)
	}
	if got := codes(c); got["KWC3001"] != 1 || len(got) != 1 {
		t.Errorf("diagnostics: %+v", c.Diagnostics)
	}
	if d := c.Diagnostics[0]; d.Location.Path != "docs/guide.md" || d.Location.Line != 1 || !strings.Contains(d.Message, "1 line") {
		t.Errorf("stale diagnostic: %+v", d)
	}
	for _, secret := range []string{"hunter2", "s3cr3t"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("secret %q leaked into the context", secret)
		}
	}
	redacted := 0
	for _, cv := range c.Config {
		if cv.Redacted {
			redacted++
		}
	}
	if redacted != 2 {
		t.Errorf("want 2 redacted config values, got %d", redacted)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	_, a := generate(t, fixture)
	_, b := generate(t, fixture) // different temp root, same content
	if string(a) != string(b) {
		t.Fatal("identical inputs must produce identical context documents")
	}
	changed := map[string]string{}
	for k, v := range fixture {
		changed[k] = v
	}
	changed["pkg/b/b.go"] = "package b\n\nconst X = 2\n"
	c1, _ := generate(t, fixture)
	c2, _ := generate(t, changed)
	if c1.Fingerprint == c2.Fingerprint {
		t.Error("changed input must change the fingerprint")
	}
}

func TestLintMissingConflictingMalformed(t *testing.T) {
	c, err := Generate(Options{Root: writeTree(t, map[string]string{"README.md": "x\n"})})
	if err != nil {
		t.Fatal(err)
	}
	if got := codes(c); got["KWC1001"] != 1 || got["KWC1003"] != 1 || got["KWC1005"] != 1 || !c.Blocking() {
		t.Errorf("empty repo diagnostics: %+v", got)
	}
	if len(c.Gates) != len(DefaultGates()) {
		t.Error("default gates apply when AGENTS.md is missing")
	}
	bad := map[string]string{
		"go.mod":       "module example.com/bad\n\ngo one\n\nrequire (\n\tbroken\n)\n",
		"version.go":   "package bad\n\nvar Version = \"v-not-semver\"\n",
		"krewire.yaml": "project: [unclosed\n",
		"AGENTS.md":    "# nothing\n",
		"x.go":         "package bad(\n",
	}
	c, err = Generate(Options{Root: writeTree(t, bad)})
	if err != nil {
		t.Fatal(err)
	}
	got := codes(c)
	for _, want := range []string{"KWC4001", "KWC4002", "KWC4003", "KWC4004", "KWC1004", "KWC1002"} {
		if got[want] == 0 {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	conflict := map[string]string{
		"go.mod":       "module example.com/c\n\ngo 1.27\n",
		"version.go":   "package c\n\nvar Version = version.MustParseVersion(\"1.0.0\")\n",
		"krewire.yaml": "project:\n  version: 0.9.0\n  kind: service\n",
		"AGENTS.md":    "## Quality Gates\n\n```\ngo test ./...\ngo test -race ./...\n```\n## Architecture Rules\n- a\n",
		"main.go":      "package main\n",
	}
	c, _ = generate(t, conflict)
	if got := codes(c); got["KWC2001"] != 1 || got["KWC2002"] != 1 {
		t.Errorf("conflicts: %+v", c.Diagnostics)
	}
	if c.Workload.Kind != "service" || c.Workload.Maturity != "stable" {
		t.Errorf("explicit kind and v1 maturity: %+v", c.Workload)
	}
	if _, err := Generate(Options{Root: filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Error("unreadable root must fail")
	}
}

func TestCrossCompileGatesAreDistinct(t *testing.T) {
	src := contract.Provenance{}
	host, _ := gateFromLine("CGO_ENABLED=0 go build ./...", src)
	wasm, _ := gateFromLine("GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./pkg/runtime/...", src)
	if host.ID != "build" || wasm.ID != "build-js-wasm" || wasm.Type != host.Type {
		t.Fatalf("ids: host %q, wasm %q (%s/%s)", host.ID, wasm.ID, host.Type, wasm.Type)
	}
}

func TestCodesRegistry(t *testing.T) {
	for code, spec := range Codes {
		if len(code) != 7 || !strings.HasPrefix(code, "KWC") || spec.Title == "" {
			t.Errorf("malformed code %s", code)
		}
		class := map[byte]string{'1': "MISSING", '2': "CONFLICT", '3': "STALE", '4': "MALFORMED"}[code[3]]
		if string(spec.Class) != class {
			t.Errorf("%s: class %s does not match its range %s", code, spec.Class, class)
		}
	}
}
