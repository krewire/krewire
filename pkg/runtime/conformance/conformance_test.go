package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/runtime/mount"
	ftest "github.com/krewire/krewire/pkg/testing"
	"github.com/krewire/krewire/pkg/testing/browser"
	"github.com/krewire/krewire/pkg/web/ssg"
	"github.com/krewire/ui/kiw"
)

// ── 1. Matrix Catalog Self-Verification ─────────────────────────────────────

func TestConformanceMatrix_AllFeaturesDocumentedAndTested(t *testing.T) {
	if len(ConformanceMatrix) == 0 {
		t.Fatal("ConformanceMatrix must not be empty")
	}

	seenIDs := make(map[string]bool)
	for _, f := range ConformanceMatrix {
		if f.ID == "" {
			t.Errorf("feature has empty ID: %+v", f)
		}
		if seenIDs[f.ID] {
			t.Errorf("duplicate feature ID: %s", f.ID)
		}
		seenIDs[f.ID] = true

		if f.Name == "" || f.Description == "" || f.TestFunc == "" {
			t.Errorf("feature %s has missing metadata: %+v", f.ID, f)
		}
		if f.Tier != TierSupported && f.Tier != TierProgressive && f.Tier != TierUnsupported {
			t.Errorf("feature %s has invalid tier: %s", f.ID, f.Tier)
		}
	}

	summary := FormatMatrixSummary()
	if !strings.Contains(summary, "Krewire .kiw Conformance Matrix:") {
		t.Errorf("FormatMatrixSummary invalid: %s", summary)
	}

	// Verify Browser baseline catalog
	if len(BrowserMatrix) == 0 {
		t.Fatal("BrowserMatrix must not be empty")
	}
	hasLegacyUnsupported := false
	for _, b := range BrowserMatrix {
		if b.Platform == "" || b.Tier == "" {
			t.Errorf("incomplete browser matrix entry: %+v", b)
		}
		if b.Tier == TierUnsupported {
			hasLegacyUnsupported = true
		}
	}
	if !hasLegacyUnsupported {
		t.Error("BrowserMatrix must explicitly document TierUnsupported for legacy engines")
	}
}

// ── 2. Grammar & Parsing Conformance ────────────────────────────────────────

func TestConformance_Grammar_Frontmatter(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	if title, ok := mod.Frontmatter["title"].(string); !ok || title != "Full Grammar Conformance" {
		t.Errorf("expected title 'Full Grammar Conformance', got %v", mod.Frontmatter["title"])
	}
	if layout, ok := mod.Frontmatter["layout"].(string); !ok || layout != "BaseLayout" {
		t.Errorf("expected layout 'BaseLayout', got %v", mod.Frontmatter["layout"])
	}
}

func TestConformance_Grammar_PropsAndDefaults(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	if len(mod.Props) < 3 {
		t.Fatalf("expected at least 3 props, got %d", len(mod.Props))
	}

	headline, ok := mod.Props["headline"]
	if !ok || headline.Type != "string" || !headline.Required || headline.Default != "Hello World" {
		t.Errorf("invalid headline prop spec: %+v", headline)
	}

	count, ok := mod.Props["count"]
	if !ok || count.Type != "number" || count.Default != "42" {
		t.Errorf("invalid count prop spec: %+v", count)
	}

	active, ok := mod.Props["active"]
	if !ok || active.Type != "boolean" || active.Default != "true" {
		t.Errorf("invalid active prop spec: %+v", active)
	}
}

func TestConformance_Grammar_Slots(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// Slot names declared in frontmatter or discovered in template
	hasHeader := false
	hasFooter := false
	for _, s := range mod.Slots {
		if s.Name == "header" {
			hasHeader = true
		}
		if s.Name == "footer" {
			hasFooter = true
		}
	}
	if !hasHeader || !hasFooter {
		t.Errorf("expected header and footer slots in module slots: %+v", mod.Slots)
	}
}

func TestConformance_Grammar_Conditionals(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// Desugared body must have transformed {#if ...} into {{if ...}}
	if strings.Contains(mod.Body, "{#if") {
		t.Errorf("desugared body still contains raw {#if: %s", mod.Body)
	}
	if !strings.Contains(mod.Body, "{{if") {
		t.Errorf("desugared body missing {{if: %s", mod.Body)
	}
	if !strings.Contains(mod.Body, "{{else if") && !strings.Contains(mod.Body, "{{else}}") {
		t.Errorf("desugared body missing else / else if: %s", mod.Body)
	}
	if !strings.Contains(mod.Body, "{{end}}") {
		t.Errorf("desugared body missing {{end}}: %s", mod.Body)
	}
}

func TestConformance_Grammar_Loops(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// Desugared body must have transformed {#each items as item, index} into range loop
	if strings.Contains(mod.Body, "{#each") {
		t.Errorf("desugared body still contains raw {#each: %s", mod.Body)
	}
	if !strings.Contains(mod.Body, "{{range") {
		t.Errorf("desugared body missing {{range: %s", mod.Body)
	}
}

func TestConformance_Grammar_StylesAndClasses(t *testing.T) {
	content, err := os.ReadFile("testdata/valid_grammar.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/valid_grammar.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	if len(mod.StyleBlocks) == 0 {
		t.Fatal("expected StyleBlocks to be extracted")
	}
	if !mod.StyleBlocks[0].Scoped {
		t.Error("expected first style block to be scoped")
	}
	if !strings.Contains(mod.StyleBlocks[0].Content, ".headline") {
		t.Errorf("style block content missing .headline: %s", mod.StyleBlocks[0].Content)
	}
}

func TestConformance_Grammar_CompoundAndNamespaced(t *testing.T) {
	content, err := os.ReadFile("testdata/slots_composition.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/slots_composition.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// Must find compound components: Card.Container, Card.Header, Card.Body, Card.Footer, ui:Button
	foundNames := make(map[string]bool)
	for _, c := range mod.Components {
		foundNames[c.Name] = true
	}

	expected := []string{"Card.Container", "Card.Header", "Card.Body", "Card.Footer", "ui:Button"}
	for _, exp := range expected {
		if !foundNames[exp] {
			t.Errorf("expected component %q in parsed calls, got: %+v", exp, foundNames)
		}
	}
}

// ── 3. Diagnostics & Error Recovery ─────────────────────────────────────────

func TestConformance_Diagnostics_LineColAccurate(t *testing.T) {
	content, err := os.ReadFile("testdata/invalid_syntax.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/invalid_syntax.kiw: %v", err)
	}

	src := string(content)
	mod, _ := kiw.ParseKiw(src)

	// Attempt to parse desugared body with html/template to trigger syntax failure
	tmpl := template.New("test")
	_, tmplErr := tmpl.Parse(mod.Body)
	if tmplErr == nil {
		t.Fatal("expected template parse error on unclosed {#if} block")
	}

	explained := kiw.Explain("invalid_syntax.kiw", "test", src, nil, tmplErr)
	var diag *kiw.Diagnostic
	if !errors.As(explained, &diag) || diag == nil {
		t.Fatalf("Explain did not produce a *kiw.Diagnostic: %v", explained)
	}
	if diag.Line <= 0 {
		t.Errorf("diagnostic line must be > 0, got %d", diag.Line)
	}
	if diag.Col < 0 {
		t.Errorf("diagnostic col must be >= 0, got %d", diag.Col)
	}
	if diag.Message == "" {
		t.Errorf("diagnostic message is empty")
	}
}

func TestConformance_Diagnostics_MalformedBlocks(t *testing.T) {
	malformed := "---\ntitle: [unclosed yaml list\n---\n<p>test</p>"
	mod, _ := kiw.ParseKiw(malformed)
	if mod == nil {
		t.Fatal("ParseKiw should not return nil on malformed frontmatter")
	}
	// Verify that body is still parsed safely
	if !strings.Contains(mod.Body, "<p>test</p>") {
		t.Errorf("expected body to survive frontmatter error, got: %s", mod.Body)
	}
}

// ── 4. SSR / SSG Rendering & Escaping ───────────────────────────────────────

func TestConformance_SSR_ContextualEscaping(t *testing.T) {
	content, err := os.ReadFile("testdata/contextual_escaping.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/contextual_escaping.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	tmpl, err := template.New("escaping").Parse(mod.Body)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	data := map[string]any{
		"PayloadScript":   "<script>alert('XSS-1')</script>",
		"PayloadAttr":     "\" onmouseover=\"alert('XSS-2')\" foo=\"",
		"PayloadURI":      "javascript:alert('XSS-3')",
		"PayloadEntities": "<b>&\"'</b>",
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execution error: %v", err)
	}

	rendered := buf.String()

	// 1. Script tag in body must be escaped
	if strings.Contains(rendered, "<script>alert('XSS-1')</script>") {
		t.Errorf("raw script tag leaked unescaped in body:\n%s", rendered)
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Errorf("expected &lt;script&gt; escaped text:\n%s", rendered)
	}

	// 2. Attribute breakout must be prevented
	if strings.Contains(rendered, " onmouseover=\"alert('XSS-2')\"") {
		t.Errorf("attribute quote breakout succeeded:\n%s", rendered)
	}

	// 3. Javascript: URI must be sanitized or escaped
	if strings.Contains(rendered, "href=\"javascript:alert('XSS-3')\"") {
		t.Errorf("javascript: URI executed unsanitized:\n%s", rendered)
	}
}

func TestConformance_SSR_DeterministicRendering(t *testing.T) {
	tmplSrc := `<div><h1>{{.Title}}</h1><ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul></div>`
	data := map[string]any{
		"Title": "Deterministic Test",
		"Items": []string{"Alpha", "Bravo", "Charlie", "Delta"},
	}

	var firstHash string
	for i := 0; i < 10; i++ {
		tmpl, err := template.New("determ").Parse(tmplSrc)
		if err != nil {
			t.Fatalf("run %d parse error: %v", i, err)
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Fatalf("run %d exec error: %v", i, err)
		}
		output := buf.String()
		h := sha256.Sum256([]byte(output))
		currentHash := hex.EncodeToString(h[:])

		if i == 0 {
			firstHash = currentHash
		} else if currentHash != firstHash {
			t.Fatalf("non-deterministic rendering at iteration %d: %s != %s", i, currentHash, firstHash)
		}
	}
}

// ── 5. Client Island Hydration & Directives ─────────────────────────────────

func TestConformance_Hydration_Directives(t *testing.T) {
	content, err := os.ReadFile("testdata/hydration_islands.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/hydration_islands.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// In mod.Components we must observe hydrate directives
	componentMap := make(map[string]kiw.ComponentCall)
	for _, c := range mod.Components {
		componentMap[c.Name] = c
	}

	counter, ok := componentMap["Counter"]
	if !ok || counter.Hydrate != "load" {
		t.Errorf("expected Counter with hydrate=load, got: %+v", counter)
	}

	searchBar, ok := componentMap["SearchBar"]
	if !ok || searchBar.Hydrate != "idle" {
		t.Errorf("expected SearchBar with hydrate=idle, got: %+v", searchBar)
	}

	dataGrid, ok := componentMap["DataGrid"]
	if !ok || dataGrid.Hydrate != "visible" {
		t.Errorf("expected DataGrid with hydrate=visible, got: %+v", dataGrid)
	}
}

func TestConformance_Hydration_FrameworkMarkers(t *testing.T) {
	content, err := os.ReadFile("testdata/hydration_islands.kiw")
	if err != nil {
		t.Fatalf("failed to read testdata/hydration_islands.kiw: %v", err)
	}

	mod, err := kiw.ParseKiw(string(content))
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	componentMap := make(map[string]kiw.ComponentCall)
	for _, c := range mod.Components {
		componentMap[c.Name] = c
	}

	if counter := componentMap["Counter"]; counter.Framework != "svelte" {
		t.Errorf("expected Counter framework=svelte, got: %q", counter.Framework)
	}
	if searchBar := componentMap["SearchBar"]; searchBar.Framework != "vue" {
		t.Errorf("expected SearchBar framework=vue, got: %q", searchBar.Framework)
	}
	if dataGrid := componentMap["DataGrid"]; dataGrid.Framework != "wasm" && dataGrid.Framework != "" {
		t.Errorf("expected DataGrid framework=wasm or default, got: %q", dataGrid.Framework)
	}

	// Verify Mount Wrap and Scan roundtrip
	wrapped := mount.WrapWithFramework("Counter", "svelte", mount.Load, `{"initial":10,"step":1}`, `<button>10</button>`)
	scanned, err := mount.Scan(wrapped)
	if err != nil {
		t.Fatalf("mount.Scan failed: %v", err)
	}
	if len(scanned) != 1 {
		t.Fatalf("expected 1 scanned mount, got %d", len(scanned))
	}
	m := scanned[0]
	if m.Name != "Counter" || m.Framework != "svelte" || m.Hydrate != mount.Load {
		t.Errorf("unexpected scanned mount: %+v", m)
	}
	if !strings.Contains(m.Props, `"initial":10`) {
		t.Errorf("props JSON lost or corrupted: %s", m.Props)
	}
}

func TestConformance_Hydration_LoaderInjection(t *testing.T) {
	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("project:\n  kind: site\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "Base.kiw"), []byte("<!DOCTYPE html><html><head></head><body>{{.Content}}</body></html>"), 0o644)

	_ = os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "components", "Widget.kiw"), []byte("<div><span>Widget</span></div>"), 0o644)

	_ = os.MkdirAll(filepath.Join(dir, "pages"), 0o755)
	pageContent := `---
title: Island Page
layout: Base
---
<Widget:svelte hydrate="load" />
`
	_ = os.WriteFile(filepath.Join(dir, "pages", "index.kiw"), []byte(pageContent), 0o644)

	site, err := ssg.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	out := t.TempDir()
	created, err := site.Build(out)
	if err != nil {
		t.Fatalf("site.Build failed: %v", err)
	}

	indexHTML, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile index.html failed: %v", err)
	}

	htmlStr := string(indexHTML)
	loaderScript := `<script type="module" src="/assets/kiw-islands.js"></script>`
	if !strings.Contains(htmlStr, loaderScript) {
		t.Errorf("expected kiw-islands.js loader script injected before </body>, got:\n%s", htmlStr)
	}

	// Verify no double injection
	if strings.Count(htmlStr, "kiw-islands.js") != 1 {
		t.Errorf("expected exactly 1 kiw-islands.js reference, found %d", strings.Count(htmlStr, "kiw-islands.js"))
	}

	// Verify asset file was written
	foundAsset := false
	for _, c := range created {
		if c == ssg.DefaultIslandsAssetPath {
			foundAsset = true
			break
		}
	}
	if !foundAsset {
		t.Errorf("expected %s in created asset list: %v", ssg.DefaultIslandsAssetPath, created)
	}
}

// ── 6. Progressive Enhancement & Fallback ───────────────────────────────────

func TestConformance_Progressive_ZeroJSFallback(t *testing.T) {
	// Without JS execution, server-rendered text inside data-kiw-mount must be accessible
	markup := mount.WrapWithFramework("UserProfile", "svelte", mount.Visible, `{"id":42}`, `<div class="user-card"><h3>Alice Doe</h3><p>Engineer</p></div>`)

	if !strings.Contains(markup, `<h3>Alice Doe</h3>`) || !strings.Contains(markup, `<p>Engineer</p>`) {
		t.Fatalf("server rendered content must remain intact inside mount container: %s", markup)
	}

	// Scanned props and markers wrap but never destroy DOM text
	mounts, err := mount.Scan(markup)
	if err != nil {
		t.Fatalf("mount.Scan failed: %v", err)
	}
	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
}

func TestConformance_Progressive_FaultTolerance(t *testing.T) {
	// Malformed mount containers should not crash the scanner or renderer
	brokenHTML := `<div data-kiw-mount="" data-kiw-hydrate="unknown"><p>fallback</p>`
	scanned, err := mount.Scan(brokenHTML)
	if err != nil {
		t.Fatalf("mount.Scan must gracefully recover from broken HTML: %v", err)
	}
	// Nameless marker must be safely skipped
	if len(scanned) != 0 {
		t.Errorf("nameless marker should be skipped, got: %+v", scanned)
	}
}

// ── 7. Headless Browser Simulation ──────────────────────────────────────────

func TestConformance_Browser_HeadlessSimulation(t *testing.T) {
	pageHTML := `<!doctype html>
<html>
<head><title>Conformance Island</title></head>
<body>
  <header><h1>App Header</h1></header>
  <main>
    <div id="live-counter" data-kiw-mount="LiveCounter" data-kiw-framework="svelte" data-kiw-hydrate="load" data-kiw-props="{&#34;count&#34;:42}">
      <button id="counter-btn">Count: 42</button>
    </div>
    <div id="lazy-feed" data-kiw-mount="LazyFeed" data-kiw-framework="vue" data-kiw-hydrate="visible">
      <p id="feed-msg">Feed content ready</p>
    </div>
  </main>
</body>
</html>`

	srv := ftest.Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(pageHTML))
	}))

	b := browser.New(t, srv.URL)
	b.Navigate("/")
	b.WaitVisible("header h1")
	b.WaitText("header h1", "App Header")
	b.WaitVisible("#counter-btn")
	b.WaitText("#counter-btn", "Count: 42")
	b.WaitVisible("#feed-msg")
	b.WaitText("#feed-msg", "Feed content ready")

	if btnText := b.Text("#counter-btn"); btnText != "Count: 42" {
		t.Errorf("expected counter text 'Count: 42', got %q", btnText)
	}

	b.HTMLAssert().Has("#live-counter", 1)
	b.HTMLAssert().Has("#lazy-feed", 1)
}

// ── 8. Regression Fixtures for Confirmed Defects ────────────────────────────

func TestConformance_Regression_Defects(t *testing.T) {
	t.Run("Regression_NullPropsOmission", func(t *testing.T) {
		// Confirmed defect: empty or null props emitted data-kiw-props="null" which caused client JSON parse warning
		wrappedEmpty := mount.Wrap("Badge", mount.Load, "", `<span>Static</span>`)
		if strings.Contains(wrappedEmpty, "data-kiw-props") {
			t.Errorf("empty props must omit data-kiw-props attribute: %s", wrappedEmpty)
		}

		wrappedNull := mount.Wrap("Badge", mount.Load, "null", `<span>Static</span>`)
		if strings.Contains(wrappedNull, "data-kiw-props") {
			t.Errorf("null props must omit data-kiw-props attribute: %s", wrappedNull)
		}
	})

	t.Run("Regression_NestedIslandsInDocumentOrder", func(t *testing.T) {
		// Confirmed defect: scanning nested islands dropped or reordered child components
		content, err := os.ReadFile("testdata/regression_nested_islands.kiw")
		if err != nil {
			t.Fatalf("read file failed: %v", err)
		}

		mod, err := kiw.ParseKiw(string(content))
		if err != nil {
			t.Fatalf("ParseKiw failed: %v", err)
		}

		// Desugaring must render both components
		if !strings.Contains(mod.Body, "DashboardShell") || !strings.Contains(mod.Body, "AnalyticsChart") {
			t.Errorf("expected both DashboardShell and AnalyticsChart in desugared body: %s", mod.Body)
		}

		// Scanning rendered HTML must preserve parent-then-child document order
		htmlMarkup := `<body>` +
			mount.WrapWithFramework("DashboardShell", "svelte", mount.Load, `{}`,
				`<div class="inner">`+
					mount.WrapWithFramework("AnalyticsChart", "vue", mount.Visible, `{"metric":"cpu"}`, `<canvas></canvas>`)+
					`</div>`) +
			`</body>`

		scanned, err := mount.Scan(htmlMarkup)
		if err != nil {
			t.Fatalf("mount.Scan failed: %v", err)
		}
		if len(scanned) != 2 {
			t.Fatalf("expected 2 scanned mounts, got %d", len(scanned))
		}
		if scanned[0].Name != "DashboardShell" || scanned[1].Name != "AnalyticsChart" {
			t.Errorf("document order violated: [0]=%s, [1]=%s", scanned[0].Name, scanned[1].Name)
		}
	})

	t.Run("Regression_SingleQuotePropsEscaping", func(t *testing.T) {
		// Confirmed defect: props JSON containing single quotes or HTML entities broke DOM parsing
		payload := `{"title":"O'Connor & Sons <Safe>"}`
		wrapped := mount.Wrap("Header", mount.Load, payload, `<h1>Hi</h1>`)

		scanned, err := mount.Scan(wrapped)
		if err != nil {
			t.Fatalf("mount.Scan failed: %v", err)
		}
		if len(scanned) != 1 {
			t.Fatalf("expected 1 mount, got %d", len(scanned))
		}
		if scanned[0].Props != payload {
			t.Errorf("props payload mismatch after roundtrip:\nwant: %s\ngot:  %s", payload, scanned[0].Props)
		}
	})
}
