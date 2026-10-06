// Tests for KWF-M4R8T. Scope: Forge — Components — Atomic.
// FRK-FA-002, FRK-FA-004, FRK-FA-006, FRK-FA-020..026.
//
// Forge ships its components as .kiw modules, so they are only ever exercised
// by the real pipeline: parse as a DSL module, then render through a site.
// This lives in framework/web/ssg because that is the loader that consumes
// ui.ComponentsFS; forge cannot import libs/kiw itself without violating
// the layering (KWL-LAYER-001).
package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ui "github.com/krewire/krewire/packages/ui"
)

// forgeComponents lists the component names load.go will register.
func forgeComponents() ([]string, error) {
	entries, err := ui.ComponentsFS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kiw") {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".kiw"))
	}
	return out, nil
}

// atomicProbeSite writes a minimal site whose page renders every named Forge
// component, so each can be asserted through real output instead of by matching
// its source text.
func atomicProbeSite(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"pages", "layouts", "components"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("krewire.yaml", "project:\n  name: atoms\n  kind: site\n  version: \"v1.2.3\"\n")
	write("layouts/Base.kiw", `<!DOCTYPE html><html><head><title>t</title></head><body>{{.Content}}</body></html>`)
	write("components/Probe.kiw", `<div data-kiw-component="Probe">{{.Body}}</div>`)
	body := "<p>probed</p>"
	for _, n := range names {
		body += "\n<" + n + " Text=\"probe\" />"
	}
	write("pages/index.kiw", "---\ntitle: Home\nlayout: Base\nroot: Probe\n---\n"+body+"\n")
	return root
}

// The atomic components added to close the primitive gap must each render.
//
// The concrete failure this guards: an early draft of Progress called a helper
// ({{progressPercent}}) that no funcmap defines. load.go silently skips a module
// it cannot parse, so a broken component would disappear from every site rather
// than breaking a build — the check belongs here, where they actually render.
func TestForgeAtomicComponentsRender(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"List", `<ul class="kiw-list`},
		{"Code", `<code class="kiw-code`},
		{"Kbd", `<kbd class="kiw-kbd`},
		{"Progress", `role="progressbar"`},
		{"Avatar", `data-kiw-component="Avatar"`},
		{"Spinner", `data-kiw-component="Spinner"`},
		{"Tabs", `role="tablist"`},
		{"Tooltip", `role="tooltip"`},
		{"Collapse", `<details class="kiw-collapse`},
		{"Breadcrumb", `<nav class="kiw-breadcrumb`},
		{"BreadcrumbItem", `kiw-breadcrumb-item`},
	}
	names := make([]string, 0, len(cases))
	for _, c := range cases {
		names = append(names, c.name)
	}
	site, err := LoadFromDir(atomicProbeSite(t, names...))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatalf("build: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, c := range cases {
		if !strings.Contains(html, c.want) {
			t.Errorf("%s: rendered page missing %q\n%s", c.name, c.want, html)
		}
	}
}

// Every component in the embedded FS must reach the site. load.go `continue`s
// past a module it cannot parse, so without this a broken component would
// vanish from every generated site and no build would report it.
func TestAllForgeComponentsLoadIntoSite(t *testing.T) {
	site, err := LoadFromDir(atomicProbeSite(t))
	if err != nil {
		t.Fatal(err)
	}
	names, err := forgeComponents()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no Forge components found in the embedded FS")
	}
	for _, name := range names {
		if _, ok := site.comps[name]; !ok {
			t.Errorf("component %q did not load — its .kiw module probably fails to parse", name)
		}
	}
}
