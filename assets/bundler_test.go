package assets

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBundler_BundleCSS(t *testing.T) {
	b := NewBundler()

	items := []BundleItem{
		{
			Path: "assets/a.css",
			Content: `@charset "UTF-8";
@import url("https://fonts.googleapis.com/css2?family=Inter");
.a { color: red; }`,
		},
		{
			Path: "assets/b.css",
			Content: `@charset "UTF-8";
@import "vendor.css";
.b { color: blue; }`,
		},
	}

	bundled := b.BundleCSS(items)

	// 1. Single @charset at top
	if !strings.HasPrefix(strings.TrimSpace(bundled), `@charset "UTF-8";`) {
		t.Errorf("expected @charset at top, got: %s", bundled)
	}
	if strings.Count(bundled, "@charset") != 1 {
		t.Errorf("expected exactly 1 @charset, got: %d", strings.Count(bundled, "@charset"))
	}

	// 2. All @imports hoisted before rules
	impIdx := strings.Index(bundled, `@import "vendor.css";`)
	ruleIdx := strings.Index(bundled, `.a { color: red; }`)
	if impIdx == -1 || ruleIdx == -1 || impIdx > ruleIdx {
		t.Errorf("@import must be hoisted before rule bodies: imp=%d, rule=%d", impIdx, ruleIdx)
	}

	// 3. Comments identify files
	if !strings.Contains(bundled, "/* assets/a.css */") || !strings.Contains(bundled, "/* assets/b.css */") {
		t.Errorf("expected file header comments in bundled output: %s", bundled)
	}
}

func TestBundler_BundleJS_SemicolonIsolation(t *testing.T) {
	b := NewBundler()

	items := []BundleItem{
		{
			Path:    "assets/first.js",
			Content: "var a = 1", // no trailing semicolon
		},
		{
			Path:    "assets/second.js",
			Content: "(function(){ window.ready = true })()", // starts with parenthesis
		},
	}

	bundled := b.BundleJS(items)

	// Verify semicolon separator is inserted between scripts
	if !strings.Contains(bundled, "\n;\n") {
		t.Errorf("expected semicolon isolation between scripts, got: %s", bundled)
	}
}

func TestBundler_StoreIntegration(t *testing.T) {
	store := NewStore().Mount(fstest.MapFS{
		"assets/style.css": &fstest.MapFile{Data: []byte(".c { margin: 0; }")},
		"assets/theme.css": &fstest.MapFile{Data: []byte(":root { --primary: #39D353; }")},
		"assets/app.js":    &fstest.MapFile{Data: []byte("console.log('app');")},
	})

	bundler := NewBundler()
	res, err := bundler.Bundle(store, []string{"assets/style.css", "assets/theme.css"}, []string{"assets/app.js"})
	if err != nil {
		t.Fatalf("bundle failed: %v", err)
	}

	if res.CSSPath != "assets/krewire.css" {
		t.Errorf("expected CSSPath assets/krewire.css, got %s", res.CSSPath)
	}
	if res.JSPath != "assets/krewire.js" {
		t.Errorf("expected JSPath assets/krewire.js, got %s", res.JSPath)
	}
	if !strings.Contains(res.CSSContent, ".c { margin: 0; }") || !strings.Contains(res.CSSContent, "--primary: #39D353") {
		t.Errorf("missing css content in bundle: %s", res.CSSContent)
	}
	if !strings.Contains(res.JSContent, "console.log('app');") {
		t.Errorf("missing js content in bundle: %s", res.JSContent)
	}
	if res.CSSFingerprint == "" || res.JSFingerprint == "" {
		t.Errorf("expected fingerprints, got css=%s, js=%s", res.CSSFingerprint, res.JSFingerprint)
	}
}

func TestBundler_Minification(t *testing.T) {
	css := `
		/* Header */
		.card {
			color: red;
			background: white;
		}
	`
	minCSS := MinifyCSS(css)
	if strings.Contains(minCSS, "/* Header */") || strings.Contains(minCSS, "\n") {
		t.Errorf("expected minified css, got %q", minCSS)
	}

	js := `
		// single line
		/* multi line */
		var x = 1;
	`
	minJS := MinifyJS(js)
	if strings.Contains(minJS, "single line") || strings.Contains(minJS, "multi line") {
		t.Errorf("expected minified js, got %q", minJS)
	}
}
