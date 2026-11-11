package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	kiwdsl "github.com/krewire/krewire/pkg/kiw"
)

func TestModularComponentsAndPackages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ssg-modular-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create directories
	dirs := []string{
		"components",
		"components/Card",
		"components/marketing",
		"layouts",
		"pages",
		".krewire/pkg/acme-ui/components",
		".krewire/pkg/acme-ui/styles",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Create Base layout
	baseLayout := `<!DOCTYPE html><html><head><title>{{.Title}}</title>{{assetLinks ""}}</head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(filepath.Join(tempDir, "layouts", "Base.kiw"), []byte(baseLayout), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Create standalone package manifest & component in .krewire/pkg/acme-ui
	manifest := &kiwdsl.PackageManifest{
		Name:        "@acme/ui",
		Version:     "1.0.0",
		Description: "Acme corporate UI kit",
		Components: map[string]string{
			"Button": "./components/Button.kiw",
		},
		Styles: "./styles/acme.css",
	}
	if err := kiwdsl.SavePackageManifest(filepath.Join(tempDir, ".krewire", "pkg", "acme-ui", kiwdsl.PackageManifestFile), manifest); err != nil {
		t.Fatal(err)
	}
	btnKiw := `<button class="acme-btn" class:primary={variant == 'primary'}><slot>Default Acme Button</slot></button>
<script>
  export let variant: string = "primary";
</script>`
	if err := os.WriteFile(filepath.Join(tempDir, ".krewire", "pkg", "acme-ui", "components", "Button.kiw"), []byte(btnKiw), 0o644); err != nil {
		t.Fatal(err)
	}
	acmeCSS := `.acme-btn { padding: 8px 16px; }`
	if err := os.WriteFile(filepath.Join(tempDir, ".krewire", "pkg", "acme-ui", "styles", "acme.css"), []byte(acmeCSS), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Create project-specific nested component in components/marketing/Hero.kiw and components/Card/Header.kiw
	heroKiw := `<div class="hero">
  <h1><slot name="title">Hero Title</slot></h1>
  <p><slot>Hero Content</slot></p>
</div>`
	if err := os.WriteFile(filepath.Join(tempDir, "components", "marketing", "Hero.kiw"), []byte(heroKiw), 0o644); err != nil {
		t.Fatal(err)
	}

	cardHeaderKiw := `<div class="card-header"><h3><slot>Default Header</slot></h3></div>`
	if err := os.WriteFile(filepath.Join(tempDir, "components", "Card", "Header.kiw"), []byte(cardHeaderKiw), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. Create local helper component in pages/LocalBadge.kiw
	localBadgeKiw := `<span class="badge">{label}</span>
<script>
  export let label: string = "New";
</script>`
	if err := os.WriteFile(filepath.Join(tempDir, "pages", "LocalBadge.kiw"), []byte(localBadgeKiw), 0o644); err != nil {
		t.Fatal(err)
	}

	// 5. Create index page importing local component and invoking package component
	indexPage := `---
title: Modular Home
---
<script>
  import LocalBadge from './LocalBadge.kiw';
</script>

<marketing:Hero>
  <template slot="title">Welcome to Modular Kiw</template>
  <LocalBadge />
  <acme:Button />
  <Card.Header>Dotted Compound Header</Card.Header>
</marketing:Hero>
`
	if err := os.WriteFile(filepath.Join(tempDir, "pages", "index.kiw"), []byte(indexPage), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load site from directory
	site, err := LoadFromDir(tempDir)
	if err != nil {
		t.Fatalf("LoadFromDir error: %v", err)
	}

	// Build site
	outDir := filepath.Join(tempDir, "dist")
	if _, err := site.Build(outDir); err != nil {
		t.Fatalf("site.Build error: %v", err)
	}

	// Read generated index.html
	htmlBytes, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	htmlStr := string(htmlBytes)

	// Verify rendered components
	if !strings.Contains(htmlStr, "Welcome to Modular Kiw") {
		t.Errorf("expected hero title rendered, got:\n%s", htmlStr)
	}
	if !strings.Contains(htmlStr, "Default Acme Button") {
		t.Errorf("expected fallback slot in acme button rendered, got:\n%s", htmlStr)
	}
	if !strings.Contains(htmlStr, "Dotted Compound Header") {
		t.Errorf("expected dotted compound Card.Header rendered, got:\n%s", htmlStr)
	}
	if !strings.Contains(htmlStr, `class="badge"`) {
		t.Errorf("expected LocalBadge rendered, got:\n%s", htmlStr)
	}
	if !strings.Contains(htmlStr, `pkg-acme.css`) {
		t.Errorf("expected acme package stylesheet injected, got:\n%s", htmlStr)
	}
}
