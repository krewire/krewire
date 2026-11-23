package workgraph

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	pBase  = "package:example.com/core/base"
	pMid   = "package:example.com/core/mid"
	pTop   = "package:example.com/core/top"
	pTool  = "package:example.com/tools/cmd/tool"
	mCore  = "module:example.com/core"
	mTools = "module:example.com/tools"
	tApp   = "template:core/templates/init/app"
	wApp   = "workload:app"
	xText  = "external:golang.org/x/text"
)

func fixture(t *testing.T) *Workspace {
	t.Helper()
	ws, err := Discover("testdata/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestDiscoverNodes(t *testing.T) {
	g := fixture(t).Graph
	var ids []string
	for _, n := range g.Nodes() {
		ids = append(ids, n.ID)
	}
	want := []string{"external:github.com/acme/unused", xText, mCore, mTools,
		pBase, pMid, pTop, pTool, tApp, wApp}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("nodes =\n%v\nwant\n%v", ids, want)
	}
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverEdgesAndProvenance(t *testing.T) {
	g := fixture(t).Graph
	cases := []struct {
		from, to string
		kind     EdgeKind
		file     string
		line     int
	}{
		{pMid, pBase, EdgeImport, "core/mid/mid.go", 6},
		{pMid, xText, EdgeImport, "core/mid/mid.go", 7},
		{pMid, pTop, EdgeTestImport, "core/mid/mid_test.go", 6},
		{pTop, pMid, EdgeImport, "core/top/top.go", 6},
		{pTool, pBase, EdgeImport, "tools/cmd/tool/main.go", 3},
		{mTools, mCore, EdgeRequire, "tools/go.mod", 5},
		{mCore, xText, EdgeRequire, "core/go.mod", 6},
		{tApp, pTop, EdgeUses, "core/templates/init/app/main.go.tmpl", 3},
		{wApp, tApp, EdgeTemplate, ManifestFile, 0},
		{wApp, pTop, EdgeUses, ManifestFile, 0},
		{pBase, mCore, EdgeMember, "", 0},
	}
	for _, c := range cases {
		var found *Edge
		for _, e := range g.EdgesBetween(c.from, c.to) {
			if e.Kind == c.kind {
				found = e
			}
		}
		if found == nil {
			t.Errorf("missing %s edge %s -> %s", c.kind, c.from, c.to)
			continue
		}
		if c.file != "" && (found.Provenance[0].File != c.file || found.Provenance[0].Line != c.line) {
			t.Errorf("%s -> %s provenance = %+v", c.from, c.to, found.Provenance)
		}
	}
}

func TestDiscoverOwnership(t *testing.T) {
	ws := fixture(t)
	cases := map[string][]string{
		"core/mid/mid.go":                      {pMid},
		"core/mid/testdata/fixture.txt":        {pMid},
		"core/top/assets/logo.txt":             {pTop},
		"core/go.sum":                          {mCore},
		"tools/go.mod":                         {mTools},
		"core/docs/base.md":                    {pBase},
		"core/fixtures/app/krewire.yaml":       {wApp},
		"core/templates/init/app/new.tmpl":     {tApp}, // deleted/new file in a template tree
		"core/top/removed.go":                  {pTop}, // deleted Go file
		"core/README.md":                       nil,
		"core/schemas/config.schema.json":      nil, // global input, owned by nobody
		"core/templates/init/app/main.go.tmpl": {tApp},
	}
	for file, want := range cases {
		if got := ws.Owners(file); !reflect.DeepEqual(got, want) {
			t.Errorf("Owners(%s) = %v, want %v", file, got, want)
		}
	}
}

func TestDiscoverDeterministicAndPortable(t *testing.T) {
	a, b := fixture(t), fixture(t)
	if a.Graph.Digest() != b.Graph.Digest() {
		t.Fatal("digest differs between runs")
	}
	var buf bytes.Buffer
	if err := a.Graph.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), a.Root) {
		t.Fatal("graph JSON leaks the absolute root")
	}
}

func TestDiscoverSingleModuleDefault(t *testing.T) {
	ws, err := Discover("testdata/ws/tools", &Manifest{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ws.Graph.Node("external:example.com/core"); !ok {
		t.Fatal("out-of-workspace require should be an external node")
	}
}

func TestManifestErrors(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field": "version: 1\nbogus: true\n",
		"version":       "version: 2\n",
		"no name":       "workloads:\n  - template: x\n",
	} {
		if _, err := ParseManifest([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	m, err := ParseManifest(nil)
	if err != nil || m == nil {
		t.Fatalf("empty manifest: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []*Manifest{
		{Workloads: []WorkloadSpec{{Name: "w", Template: "nope"}}},
		{Workloads: []WorkloadSpec{{Name: "w", Packages: []string{"nope"}}}},
		{Inputs: map[string][]string{"nope": {"*"}}},
	} {
		if _, err := Discover(dir, bad); err == nil {
			t.Errorf("manifest %+v accepted", bad)
		}
	}
}

func TestParseModAndWork(t *testing.T) {
	mf, err := ParseModFile([]byte("module m\ngo 1.27.1\ntoolchain go1.27.2\nrequire a v1 // indirect\nreplace (\n a => ./a\n)\n"))
	if err != nil || mf.Toolchain != "go1.27.2" || len(mf.Requires) != 1 || !mf.Requires[0].Indirect {
		t.Fatalf("mod = %+v, %v", mf, err)
	}
	if _, err := ParseModFile([]byte("go 1.27\n")); err == nil {
		t.Fatal("go.mod without module accepted")
	}
	if _, err := ParseModFile([]byte("module m\nrequire a\n")); err == nil {
		t.Fatal("malformed require accepted")
	}
	if got := ParseWorkFile([]byte("use ./a\nuse (\n\t./b // c\n)\n")); !reflect.DeepEqual(got, []string{"./a", "./b"}) {
		t.Fatalf("work uses = %v", got)
	}
	for pat, name := range map[string]string{"a/**": "a/b/c", "**/*.json": "x/y.json", "a/*/c": "a/b/c"} {
		if !matchGlob(pat, name) {
			t.Errorf("%s !~ %s", pat, name)
		}
	}
	if matchGlob("a/*", "a/b/c") {
		t.Error("single star crossed a segment")
	}
}
