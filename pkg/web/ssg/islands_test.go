package ssg

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIslands_LoaderInjectedForSvelte(t *testing.T) {
	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("project:\n  name: site\n"), 0o644)

	layoutDir := filepath.Join(dir, "layouts")
	_ = os.MkdirAll(layoutDir, 0o755)
	_ = os.WriteFile(filepath.Join(layoutDir, "Base.kiw"), []byte("<!DOCTYPE html><html><head></head><body>{{.Content}}</body></html>"), 0o644)

	compDir := filepath.Join(dir, "components")
	_ = os.MkdirAll(compDir, 0o755)
	_ = os.WriteFile(filepath.Join(compDir, "Counter.kiw"), []byte("<div><span>Svelte SSR</span></div>"), 0o644)

	pageDir := filepath.Join(dir, "pages")
	_ = os.MkdirAll(pageDir, 0o755)
	pageKiw := `---
title: Svelte Island Test
layout: Base
---
<h1>Svelte Island</h1>
{{mountSvelte "Counter" "load" (dict "initial" 42)}}
`
	_ = os.WriteFile(filepath.Join(pageDir, "index.kiw"), []byte(pageKiw), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	out := t.TempDir()
	created, err := site.Build(out)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile index.html failed: %v", err)
	}
	html := string(body)

	// Must contain Svelte mount marker
	if !strings.Contains(html, `data-kiw-framework="svelte"`) {
		t.Errorf("expected data-kiw-framework=\"svelte\", got:\n%s", html)
	}
	if !strings.Contains(html, `data-kiw-mount="Counter"`) {
		t.Errorf("expected data-kiw-mount=\"Counter\", got:\n%s", html)
	}

	// Must inject client island loader
	if !strings.Contains(html, `<script type="module" src="/assets/kiw-islands.js"></script>`) {
		t.Errorf("expected kiw-islands.js module script injected, got:\n%s", html)
	}

	// Must NOT inject Go WASM loader because there are no WASM mounts
	if strings.Contains(html, "wasm_exec.js") {
		t.Errorf("unexpected wasm_exec.js in page with pure Svelte island:\n%s", html)
	}

	// kiw-islands.js must be created in output directory
	foundAsset := false
	for _, f := range created {
		if f == DefaultIslandsAssetPath {
			foundAsset = true
			break
		}
	}
	if !foundAsset {
		t.Errorf("expected %s in created files list: %v", DefaultIslandsAssetPath, created)
	}

	assetBody, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(DefaultIslandsAssetPath)))
	if err != nil {
		t.Fatalf("ReadFile kiw-islands.js failed: %v", err)
	}
	if !strings.Contains(string(assetBody), "KiwIslands") {
		t.Errorf("kiw-islands.js missing KiwIslands definition: %s", string(assetBody)[:200])
	}
}

func TestIslands_CoexistenceWasmAndVue(t *testing.T) {
	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("project:\n  name: site\n"), 0o644)

	layoutDir := filepath.Join(dir, "layouts")
	_ = os.MkdirAll(layoutDir, 0o755)
	_ = os.WriteFile(filepath.Join(layoutDir, "Base.kiw"), []byte("<!DOCTYPE html><html><head></head><body>{{.Content}}</body></html>"), 0o644)

	compDir := filepath.Join(dir, "components")
	_ = os.MkdirAll(compDir, 0o755)
	_ = os.WriteFile(filepath.Join(compDir, "GoCounter.kiw"), []byte("<div>Go Counter</div>"), 0o644)
	_ = os.WriteFile(filepath.Join(compDir, "VueWidget.kiw"), []byte("<div>Vue Widget</div>"), 0o644)

	pageDir := filepath.Join(dir, "pages")
	_ = os.MkdirAll(pageDir, 0o755)
	pageKiw := `---
title: Coexistence Test
layout: Base
---
{{mount "GoCounter" "load" (dict)}}
{{mountVue "VueWidget" "visible" (dict "theme" "dark")}}
`
	_ = os.WriteFile(filepath.Join(pageDir, "index.kiw"), []byte(pageKiw), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	out := t.TempDir()
	_, err = site.Build(out)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile index.html failed: %v", err)
	}
	html := string(body)

	// Both loaders must be injected!
	if !strings.Contains(html, "wasm_exec.js") {
		t.Errorf("expected wasm_exec.js for GoCounter, got:\n%s", html)
	}
	if !strings.Contains(html, "/assets/kiw-islands.js") {
		t.Errorf("expected kiw-islands.js for VueWidget, got:\n%s", html)
	}
}

func TestIslands_OpenServesDefaultScript(t *testing.T) {
	site := New()
	file, err := site.Open("assets/kiw-islands.js")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !strings.Contains(string(content), "KiwIslands") {
		t.Fatalf("expected KiwIslands content, got: %s", string(content)[:200])
	}
}

func TestIslands_CustomLoaderURL(t *testing.T) {
	site := New().Islands(IslandsConfig{
		Enabled:   true,
		LoaderURL: "/static/custom-islands.js",
	})
	site.Page(Page{
		Path:   "/",
		Layout: "Base",
		Root:   "Comp",
	})
	site.Layout(Layout{
		Name: "Base",
		Body: "<html><body>{{.Content}}</body></html>",
	})
	site.Component(Component{
		Name: "Comp",
		Body: "<div>hello</div>",
	})

	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	html := string(body)

	if !strings.Contains(html, `<script type="module" src="/static/custom-islands.js"></script>`) {
		t.Errorf("expected custom loader URL, got:\n%s", html)
	}
}

func TestIslands_EndToEndDSLDirectives(t *testing.T) {
	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("project:\n  name: site\n"), 0o644)

	layoutDir := filepath.Join(dir, "layouts")
	_ = os.MkdirAll(layoutDir, 0o755)
	_ = os.WriteFile(filepath.Join(layoutDir, "Base.kiw"), []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0o644)

	compDir := filepath.Join(dir, "components")
	_ = os.MkdirAll(compDir, 0o755)
	_ = os.WriteFile(filepath.Join(compDir, "Counter.kiw"), []byte("<div>Counter SSR</div>"), 0o644)
	_ = os.WriteFile(filepath.Join(compDir, "VueWidget.kiw"), []byte("<div>Vue SSR</div>"), 0o644)

	pageDir := filepath.Join(dir, "pages")
	_ = os.MkdirAll(pageDir, 0o755)
	pageKiw := `---
title: DSL Islands Test
layout: Base
---
<Counter client:svelte client:load count=5 />
<VueWidget:vue client:visible theme="light" />
`
	_ = os.WriteFile(filepath.Join(pageDir, "index.kiw"), []byte(pageKiw), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	out := t.TempDir()
	_, err = site.Build(out)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	html := string(body)

	if !strings.Contains(html, `data-kiw-framework="svelte"`) {
		t.Errorf("expected svelte framework marker in HTML: %s", html)
	}
	if !strings.Contains(html, `data-kiw-framework="vue"`) {
		t.Errorf("expected vue framework marker in HTML: %s", html)
	}
	if !strings.Contains(html, `/assets/kiw-islands.js`) {
		t.Errorf("expected kiw-islands.js loader in HTML: %s", html)
	}
}
