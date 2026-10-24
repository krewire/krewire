package ssg

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// names extracts base names from a URL list for readable assertions.
func names(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		out = append(out, path.Base(u))
	}
	return out
}

// A fully populated site must resolve the canonical cascade order, not
// alphabetical order. This is the regression guard for the original bug, where
// book.css loaded before tailwind.css and therefore could not override it.
func TestInjectedAssetsUsesCascadeOrderNotAlphabetical(t *testing.T) {
	s := New().
		Asset("assets/book.css", "").
		Asset("assets/tailwind.css", "").
		Asset("assets/ui.css", "").
		Asset("assets/theme.css", "").
		Component(Component{Name: "c", Body: "x", Style: ".c{}"})
	css, _ := s.InjectedAssets()
	got := names(css)
	want := []string{"style.css", "tailwind.css", "ui.css", "theme.css", "book.css"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("css order = %v, want %v", got, want)
	}
	if len(got) > 0 && got[0] == "book.css" {
		t.Error("book.css must not load first; it exists to override the site")
	}
}

// A vendor stylesheet registered after the theme must still load before it when
// its layer says so.
func TestInjectedAssetsRespectsExplicitLayer(t *testing.T) {
	s := New().
		AssetWith("assets/print.css", "", WithLayer(LayerScoped)).
		Asset("assets/theme.css", "")
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) < 2 {
		t.Fatalf("css = %v, want at least print.css and theme.css", got)
	}
	if got[0] != "print.css" {
		t.Errorf("css order = %v, want print.css first (LayerScoped)", got)
	}
}

// WithOrder breaks ties inside one layer without moving between layers.
func TestInjectedAssetsOrderBreaksTiesWithinLayer(t *testing.T) {
	s := New().
		AssetWith("assets/a.css", "", WithLayer(LayerTheme), WithOrder(2)).
		AssetWith("assets/b.css", "", WithLayer(LayerTheme), WithOrder(1))
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) < 2 || got[0] != "b.css" {
		t.Errorf("css order = %v, want b.css before a.css (lower order first)", got)
	}
}

// A plugin can declare its stylesheet and still land in a chosen layer, so
// third-party output does not have to sort by filename.
func TestDeclareAssetWithLayerOrdersPluginOutput(t *testing.T) {
	s := New().
		DeclareAssetWith([]string{"assets/tailwind.css"}, WithLayer(LayerVendor)).
		Asset("assets/theme.css", "")
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) != 2 || got[0] != "tailwind.css" || got[1] != "theme.css" {
		t.Errorf("css order = %v, want tailwind.css then theme.css", got)
	}
}

// A builtin that the build never produces must not be linked: a dead <link>
// would ship a 404 into every page.
func TestInjectedAssetsOmitsAbsentBuiltins(t *testing.T) {
	css, js := New().InjectedAssets()
	if len(css) != 0 || len(js) != 0 {
		t.Errorf("empty site plan = css %v js %v, want both empty", css, js)
	}
}

// The scoped stylesheet is generated during the build rather than registered, so
// it is the one builtin that may appear without being registered.
func TestInjectedAssetsIncludesGeneratedScopedCSS(t *testing.T) {
	s := New().Component(Component{Name: "c", Body: "x", Style: ".c{}"})
	css, _ := s.InjectedAssets()
	if len(css) == 0 || path.Base(css[0]) != "style.css" {
		t.Errorf("css = %v, want style.css first", css)
	}
}

func TestParseLayer(t *testing.T) {
	cases := []struct {
		in   string
		want AssetLayer
		ok   bool
	}{
		{"scoped", LayerScoped, true},
		{"base", LayerScoped, true},
		{"VENDOR", LayerVendor, true},
		{"plugin", LayerVendor, true},
		{"component", LayerComponent, true},
		{"ui", LayerComponent, true},
		{"theme", LayerTheme, true},
		{"book", LayerBook, true},
		{"last-mile", LayerPage, true},
		{"last_mile", LayerPage, true},
		{"page", LayerPage, true},
		{"  theme  ", LayerTheme, true},
		{"nope", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, err := ParseLayer(c.in)
		if c.ok && err != nil {
			t.Errorf("ParseLayer(%q) error = %v", c.in, err)
			continue
		}
		if !c.ok {
			if err == nil {
				t.Errorf("ParseLayer(%q) error = nil, want error", c.in)
			}
			continue
		}
		if got != c.want {
			t.Errorf("ParseLayer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// A misconfigured layer must be visible without failing the build.
func TestAutoAssetsReportsUnknownLayer(t *testing.T) {
	s := New().AutoAssets(&AutoAssetConfig{
		Order: map[string]AssetOrder{"assets/theme.css": {Layer: "nope"}},
	})
	errs := s.AutoAssetErrors()
	if len(errs) != 1 {
		t.Fatalf("AutoAssetErrors() = %v, want one error", errs)
	}
	if !strings.Contains(errs[0].Error(), "assets/theme.css") {
		t.Errorf("error = %q, want it to name the offending asset", errs[0])
	}
}

// A configured layer must actually move the asset, proving config reaches the plan.
func TestAutoAssetsOrderPinsLayer(t *testing.T) {
	s := New().
		Asset("assets/extra.css", "").
		Asset("assets/theme.css", "").
		AutoAssets(&AutoAssetConfig{
			Order: map[string]AssetOrder{"theme.css": {Layer: "last-mile"}},
		})
	if len(s.AutoAssetErrors()) != 0 {
		t.Fatalf("AutoAssetErrors() = %v, want none", s.AutoAssetErrors())
	}
	css, _ := s.InjectedAssets()
	if len(css) == 0 || path.Base(css[len(css)-1]) != "theme.css" {
		t.Errorf("css = %v, want theme.css last after pinning layer last-mile", css)
	}
}

// The plan must be stable across repeated calls: map iteration order must never
// leak into the output.
func TestInjectedAssetsIsDeterministic(t *testing.T) {
	build := func() []string {
		s := New()
		for _, n := range []string{"assets/z.css", "assets/a.css", "assets/m.css", "assets/b.css"} {
			s.Asset(n, "")
		}
		css, _ := s.InjectedAssets()
		return css
	}
	first := strings.Join(build(), ",")
	for i := 0; i < 25; i++ {
		if got := strings.Join(build(), ","); got != first {
			t.Fatalf("plan changed between runs:\n %s\n %s", first, got)
		}
	}
}

// Exclusions must survive the layered plan.
func TestInjectedAssetsHonorsExclude(t *testing.T) {
	s := New().
		Asset("assets/vendor.min.css", "").
		Asset("assets/theme.css", "").
		AutoAssets(&AutoAssetConfig{Exclude: []string{"*.min.css"}})
	css, _ := s.InjectedAssets()
	for _, u := range css {
		if strings.Contains(u, "vendor.min.css") {
			t.Errorf("css = %v, want vendor.min.css excluded", css)
		}
	}
}

// Page-scoped scripts are emitted per page and must never be site-wide.
func TestInjectedAssetsSkipsPageScopedScripts(t *testing.T) {
	s := New().
		ScriptAsset("assets/layout-Base-0.js", "console.log(1)").
		Asset("assets/app.js", "")
	_, js := s.InjectedAssets()
	for _, u := range js {
		if strings.Contains(u, "layout-Base-0.js") {
			t.Errorf("js = %v, want the page-scoped script excluded", js)
		}
	}
}

// The cache-busting rule has one owner (VersionToken/VersionQuery), so injection,
// {{assetLinks}} and a host embedding the plan can never spell the version
// differently. This is the guard against re-growing a duplicated literal.
func TestVersionQueryIsSingleSourced(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"v1.2.3", "?v=1.2.3"},
		{"1.2.3", "?v=1.2.3"},
		{" 1.2.3 ", "?v=1.2.3"},
		{"", ""},
		{"v", ""},
	} {
		if got := VersionQuery(c.in); got != c.want {
			t.Errorf("VersionQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := VersionedURL("/assets/app.js", "v0.1.0"); got != "/assets/app.js?v=0.1.0" {
		t.Errorf("VersionedURL = %q", got)
	}
	if got := VersionToken("v1.2.3"); got != "1.2.3" {
		t.Errorf("VersionToken = %q, want 1.2.3", got)
	}
}

// A page must never mix spellings of the same version: the injected tags and the
// assetLinks output have to agree on the query, because both can appear in one
// document when a layout links an asset by hand.
func TestInjectedAndAssetLinksAgreeOnVersionQuery(t *testing.T) {
	s := bareSite().Asset("assets/extra.css", `a{}`)
	page := renderIndex(t, s)
	injected := VersionedURL("/assets/extra.css", "v1.2.3")
	if !strings.Contains(page, injected) {
		t.Errorf("page missing %s\n%s", injected, page)
	}
	links := string(s.assetLinks(VersionToken("v1.2.3")))
	if !strings.Contains(links, injected) {
		t.Errorf("assetLinks missing %s\ngot: %s", injected, links)
	}
}

// The {{assetLinks}} helper and automatic injection must agree on order; they
// read the same plan, so they cannot drift.
func TestAssetLinksMatchesInjectedOrder(t *testing.T) {
	s := New().
		Asset("assets/book.css", "").
		Asset("assets/tailwind.css", "").
		Asset("assets/app.js", "")
	cssURLs, jsURLs := s.InjectedAssets()
	html := string(s.assetLinks("1.0.0"))

	last := -1
	for _, u := range cssURLs {
		idx := strings.Index(html, u+"?v=1.0.0")
		if idx < 0 {
			t.Fatalf("assetLinks missing %s\ngot: %s", u, html)
		}
		if idx < last {
			t.Errorf("assetLinks order differs from the plan for %s", u)
		}
		last = idx
	}
	for _, u := range jsURLs {
		if !strings.Contains(html, u+"?v=1.0.0") {
			t.Errorf("assetLinks missing %s", u)
		}
	}
}

// Static assets like images, fonts, icons, manifests must never be injected
// into HTML documents as stylesheets or scripts.
func TestInjectedAssetsNeverInjectsStaticFiles(t *testing.T) {
	s := New().
		Asset("assets/images/logo.png", "dummy-image-bytes").
		Asset("assets/images/banner.jpg", "dummy-image-bytes").
		Asset("assets/fonts/inter.woff2", "dummy-font-bytes").
		Asset("assets/icons/favicon.ico", "dummy-icon-bytes").
		Asset("assets/theme.css", "body{}").
		Asset("assets/app.js", "console.log(1)")

	css, js := s.InjectedAssets()
	for _, c := range css {
		if strings.HasSuffix(c, ".png") || strings.HasSuffix(c, ".jpg") ||
			strings.HasSuffix(c, ".woff2") || strings.HasSuffix(c, ".ico") {
			t.Errorf("static non-css file injected into css: %s", c)
		}
	}
	for _, j := range js {
		if strings.HasSuffix(j, ".png") || strings.HasSuffix(j, ".jpg") ||
			strings.HasSuffix(j, ".woff2") || strings.HasSuffix(j, ".ico") {
			t.Errorf("static non-js file injected into js: %s", j)
		}
	}
	if len(css) != 1 || !strings.Contains(css[0], "theme.css") {
		t.Errorf("expected only theme.css in css, got: %v", css)
	}
	if len(js) != 1 || !strings.Contains(js[0], "app.js") {
		t.Errorf("expected only app.js in js, got: %v", js)
	}
}

func TestBundleAssets_InjectedPlanReturnsBundles(t *testing.T) {
	s := New().
		WithBundle(true).
		Asset("assets/tailwind.css", "/* tw */").
		Asset("assets/ui.css", "/* ui */").
		Asset("assets/app.js", "console.log(1);")

	css, js := s.InjectedAssets()
	if len(css) != 1 || css[0] != "/assets/krewire.css" {
		t.Errorf("expected [/assets/krewire.css], got: %v", css)
	}
	if len(js) != 1 || js[0] != "/assets/krewire.js" {
		t.Errorf("expected [/assets/krewire.js], got: %v", js)
	}
}

func TestBundleAssets_BuildWritesBundledAndIndividualFiles(t *testing.T) {
	outDir := t.TempDir()
	s := New().
		WithBundle(true).
		Asset("assets/tailwind.css", "/* tw css */").
		Asset("assets/ui.css", "/* ui css */").
		Asset("assets/a.js", "var a = 1;").
		Asset("assets/b.js", "var b = 2;").
		Layout(Layout{
			Name: "base",
			Body: "<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>",
		}).
		Component(Component{Name: "home", Body: "<h1>Hello</h1>"}).
		Page(Page{Path: "/", Layout: "base", Root: "home"})

	created, err := s.Build(outDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify both bundled and individual files are created
	expectedFiles := []string{
		"assets/krewire.css",
		"assets/krewire.js",
		"assets/tailwind.css",
		"assets/ui.css",
		"assets/a.js",
		"assets/b.js",
	}
	for _, ef := range expectedFiles {
		fullPath := filepath.Join(outDir, ef)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("expected file %s to exist: %v", ef, err)
		}
	}

	// Check content of krewire.css: order matters (tailwind before ui)
	cssBytes, err := os.ReadFile(filepath.Join(outDir, "assets/krewire.css"))
	if err != nil {
		t.Fatalf("failed to read krewire.css: %v", err)
	}
	cssStr := string(cssBytes)
	twIdx := strings.Index(cssStr, "/* tw css */")
	uiIdx := strings.Index(cssStr, "/* ui css */")
	if twIdx == -1 || uiIdx == -1 || twIdx > uiIdx {
		t.Errorf("krewire.css does not contain assets in cascade order: %s", cssStr)
	}

	// Check content of krewire.js: separated properly
	jsBytes, err := os.ReadFile(filepath.Join(outDir, "assets/krewire.js"))
	if err != nil {
		t.Fatalf("failed to read krewire.js: %v", err)
	}
	jsStr := string(jsBytes)
	if !strings.Contains(jsStr, "var a = 1;") || !strings.Contains(jsStr, "var b = 2;") {
		t.Errorf("krewire.js missing contents: %s", jsStr)
	}

	_ = created
}

func TestAutoAssetConfig_Bundle(t *testing.T) {
	b := true
	cfg := &AutoAssetConfig{
		Bundle: &b,
	}
	s := New().AutoAssets(cfg)
	if !s.Bundle() {
		t.Errorf("expected s.Bundle() to be true")
	}

	bFalse := false
	cfgFalse := &AutoAssetConfig{
		Bundle: &bFalse,
	}
	s2 := New().AutoAssets(cfgFalse)
	if s2.Bundle() {
		t.Errorf("expected s2.Bundle() to be false")
	}
}
