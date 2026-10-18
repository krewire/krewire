package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMountTemplateFuncEmitsMarkers(t *testing.T) {
	site, err := LoadFromDir("testdata/mount")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)

	for _, want := range []string{
		`data-kiw-mount="Counter"`,
		`data-kiw-hydrate="load"`,
		`data-kiw-props="{`,
		// SSR content stays fully rendered inside the mount wrapper.
		`<span class="count">0</span>`,
		`<button type="button">+1</button>`,
		// Scoped component CSS contract unchanged.
		`data-kiw-component="Counter"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("mount html missing %q\n got %q", want, html[:min(len(html), 800)])
		}
	}

	css, err := os.ReadFile(filepath.Join(out, "assets", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), `[data-kiw-component="Counter"]`) {
		t.Errorf("scoped css missing Counter: %s", string(css))
	}
}

func TestMountRejectsUnknownHydrateValue(t *testing.T) {
	site, err := LoadFromDir("testdata/mount")
	if err != nil {
		t.Fatal(err)
	}
	if err := site.prepare(); err != nil {
		t.Fatalf("template preparation must succeed: %v", err)
	}
	if _, err := site.renderMount("Counter", "hover", nil); err == nil {
		t.Fatal("unknown hydrate value must error")
	} else if !strings.Contains(err.Error(), "load|idle|visible") {
		t.Fatalf("error must list valid hydrate values: %v", err)
	}
	if _, err := site.renderMount("Nope", "load", nil); err == nil {
		t.Fatal("undefined component in mount must error")
	}
}

func TestMount_KiwDSLSyntax(t *testing.T) {
	dir := t.TempDir()

	// Write krewire.yaml
	_ = os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("project:\n  kind: site\n"), 0o644)

	// Layouts
	layoutDir := filepath.Join(dir, "layouts")
	_ = os.MkdirAll(layoutDir, 0o755)
	_ = os.WriteFile(filepath.Join(layoutDir, "Base.kiw"), []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0o644)

	// Components
	compDir := filepath.Join(dir, "components")
	_ = os.MkdirAll(compDir, 0o755)
	counterKiw := `---
---
<div class="counter">
  <span>{initial}</span>
</div>
<style>
.counter { color: green; }
</style>`
	_ = os.WriteFile(filepath.Join(compDir, "Counter.kiw"), []byte(counterKiw), 0o644)

	// Page with <Counter client:load initial=5 />
	pageDir := filepath.Join(dir, "pages")
	_ = os.MkdirAll(pageDir, 0o755)
	pageKiw := `---
title: Islands Test
layout: Base
---
<h1>Islands</h1>
<Counter client:load initial=5 />
`
	_ = os.WriteFile(filepath.Join(pageDir, "index.kiw"), []byte(pageKiw), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir error: %v", err)
	}

	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatalf("Build error: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	html := string(body)

	for _, want := range []string{
		`data-kiw-mount="Counter"`,
		`data-kiw-hydrate="load"`,
		`data-kiw-props="`,
		`&#34;initial&#34;:5`,
		`data-kiw-component="Counter"`,
		`<span>5</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing expected island marker %q\nHTML:\n%s", want, html)
		}
	}
}
