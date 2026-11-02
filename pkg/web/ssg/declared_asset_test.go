package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeclaredAssetBundlesPluginOutput guards the contract of DeclareAsset: an
// asset a plugin produces inside outDir must be bundled from that artifact, not
// from the in-repo source. For Tailwind the source is a CSS file full of
// @apply directives, so preferring it ships invalid CSS to the browser.
func TestDeclaredAssetBundlesPluginOutput(t *testing.T) {
	const (
		source   = "@tailwind utilities;\n.btn { @apply px-2; }\n"
		compiled = ".btn{padding-left:.5rem;padding-right:.5rem}"
	)

	outDir := t.TempDir()
	s := New().
		Asset("assets/tailwind.css", source).
		DeclareAsset("assets/tailwind.css").
		AutoAssets(&AutoAssetConfig{Bundle: boolPtr(true)}).
		Component(Component{Name: "root", Body: `<p>hello</p>`}).
		Layout(Layout{
			Name: "base",
			Body: `<!doctype html><html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`,
		}).
		Page(Page{Path: "/", Title: "Home", Layout: "base", Root: "root"})

	// Stand in for the Tailwind plugin: it writes the compiled stylesheet into
	// the output directory before the bundle is assembled.
	pluginOut := filepath.Join(outDir, "assets", "tailwind.css")
	if err := os.MkdirAll(filepath.Dir(pluginOut), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(pluginOut, []byte(compiled), 0o644); err != nil {
		t.Fatalf("write plugin output: %v", err)
	}

	if _, err := s.Build(outDir); err != nil {
		t.Fatalf("Build: %v", err)
	}

	bundle, err := os.ReadFile(filepath.Join(outDir, "assets", "krewire.css"))
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if strings.Contains(string(bundle), "@apply") {
		t.Errorf("bundle contains raw Tailwind source: %q", bundle)
	}
	if !strings.Contains(string(bundle), compiled) {
		t.Errorf("bundle missing compiled plugin output; got %q", bundle)
	}

	// Build must not clobber the plugin artifact with the in-repo source.
	kept, err := os.ReadFile(pluginOut)
	if err != nil {
		t.Fatalf("read plugin output: %v", err)
	}
	if string(kept) != compiled {
		t.Errorf("plugin output overwritten by source: got %q", kept)
	}
}
