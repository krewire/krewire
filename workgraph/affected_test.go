package workgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func affectedIDs(a *AffectedSet) []string {
	var ids []string
	for _, n := range a.Nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

func TestAffected(t *testing.T) {
	ws := fixture(t)
	all := affectedIDs(ws.All())
	cases := []struct {
		name    string
		changed []string
		want    []string
	}{
		{"leaf package marks transitive dependents, other repos, templates, workloads",
			[]string{"core/base/base.go"}, []string{pBase, pMid, pTop, pTool, tApp, wApp}},
		{"test-only importers are affected", []string{"core/top/top.go"}, []string{pMid, pTop, tApp, wApp}},
		{"embedded asset", []string{"core/top/assets/logo.txt"}, []string{pMid, pTop, tApp, wApp}},
		{"lock state invalidates the module and its requirers",
			[]string{"core/go.sum"}, []string{mCore, mTools, pBase, pMid, pTop, pTool, tApp, wApp}},
		{"shared schema is global", []string{"core/schemas/config.schema.json"}, all},
		{"toolchain pin is global", []string{"go.work"}, all},
		{"template", []string{"core/templates/init/app/main.go.tmpl"}, []string{tApp, wApp}},
		{"workload fixture", []string{"core/fixtures/app/krewire.yaml"}, []string{wApp}},
		{"extra input", []string{"core/docs/base.md"}, []string{pBase, pMid, pTop, pTool, tApp, wApp}},
		{"unowned file affects nothing", []string{"core/README.md"}, nil},
		{"leaf binary", []string{"tools/cmd/tool/main.go"}, []string{pTool}},
	}
	for _, c := range cases {
		got := affectedIDs(ws.Affected(c.changed, AffectedOptions{}))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
	strict := ws.Affected([]string{"core/README.md"}, AffectedOptions{UnownedIsGlobal: true})
	if !reflect.DeepEqual(affectedIDs(strict), all) {
		t.Errorf("UnownedIsGlobal did not affect everything: %v", affectedIDs(strict))
	}
}

func TestAffectedExplains(t *testing.T) {
	ws := fixture(t)
	a := ws.Affected([]string{"./core/base/base.go", "core/README.md"}, AffectedOptions{Base: "b", Head: "h"})
	c, ok := a.Cause(pTool)
	if !ok || c.Kind != CauseDependency || c.Via != pBase || c.Provenance.File != "tools/cmd/tool/main.go" {
		t.Fatalf("tool cause = %+v", c)
	}
	if c, _ := a.Cause(pBase); c.Kind != CauseChanged || c.File != "core/base/base.go" {
		t.Fatalf("base cause = %+v", c)
	}
	var buf bytes.Buffer
	if err := a.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var doc AffectedSet
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil || doc.Kind != KindAffectedDocument ||
		!reflect.DeepEqual(doc.Unowned, []string{"core/README.md"}) || doc.Base != "b" {
		t.Fatalf("doc = %+v, %v", doc, err)
	}
	buf.Reset()
	if err := a.WriteText(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "depends on "+pBase+" (import) at tools/cmd/tool/main.go:3") {
		t.Fatalf("text:\n%s", buf.String())
	}
	if c, _ := ws.All().Cause(pBase); c.Kind != CauseFull || c.String() != "full run" {
		t.Fatalf("full cause = %+v", c)
	}
}

func TestChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t",
			"-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	write("a/x.go", "package a\n")
	write("b/y.go", "package b\n")
	run("add", ".")
	run("commit", "-qm", "base")
	base := run("rev-parse", "HEAD")
	write("a/x.go", "package a // changed\n")
	run("mv", "b/y.go", "b/z.go")
	run("commit", "-qam", "head")
	got, err := ChangedFiles(context.Background(), dir, base, "HEAD")
	if err != nil || !reflect.DeepEqual(got, []string{"a/x.go", "b/y.go", "b/z.go"}) {
		t.Fatalf("range = %v, %v", got, err)
	}
	write("c/new.go", "package c\n")
	got, err = ChangedFiles(context.Background(), dir, "HEAD", "")
	if err != nil || !reflect.DeepEqual(got, []string{"c/new.go"}) {
		t.Fatalf("worktree = %v, %v", got, err)
	}
	if _, err := ChangedFiles(context.Background(), dir, "", ""); err == nil {
		t.Fatal("empty base accepted")
	}
	if _, err := ChangedFiles(context.Background(), dir, "no-such-rev", "HEAD"); err == nil {
		t.Fatal("bad revision accepted")
	}
}
