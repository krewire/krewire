package kiw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompoundAndNamespacedComponents(t *testing.T) {
	src := `
<Card.Header title="Card Title" />
<Card.Body>
  <p>Inside body</p>
</Card.Body>
<ui:Button variant="primary" size="lg">Submit</ui:Button>
<acme:DataGrid.Cell value=42 />
`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	expectedComponents := []string{
		`{{component "Card.Header" (dict "title" "Card Title")}}`,
		`{{component "Card.Body" (dict "Body" "<p>Inside body</p>")}}`,
		`{{component "ui:Button" (dict "variant" "primary" "size" "lg" "Body" "Submit")}}`,
		`{{component "acme:DataGrid.Cell" (dict "value" 42)}}`,
	}

	for _, expected := range expectedComponents {
		if !strings.Contains(mod.Body, expected) {
			t.Errorf("expected body to contain %q, but got:\n%s", expected, mod.Body)
		}
	}
}

func TestComponentImports_FrontmatterAndScript(t *testing.T) {
	src := `---
title: Modular Component
imports:
  Hero: "./components/marketing/Hero.kiw"
  Card: "@krewire/ui/Card"
  ui: "@krewire/ui"
props:
  headline:
    type: string
    default: "Default Headline"
    required: true
  count:
    type: number
    default: 10
slots:
  - default
  - header
exports:
  Header: "./Header.kiw"
---
<script>
  import Button from './Button.kiw';
  import { Badge, Modal as CustomModal } from '@krewire/ui';

  export let variant: string = "primary";
  export let disabled: boolean = false;
</script>

@import Nav from "./Nav.kiw"

<Hero title={headline} />
<ui:Button variant={variant} />
<Button>Click</Button>
`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	// Verify imports
	if len(mod.Imports) < 6 {
		t.Fatalf("expected at least 6 imports, got %d: %+v", len(mod.Imports), mod.Imports)
	}

	importMap := make(map[string]ImportDecl)
	for _, imp := range mod.Imports {
		importMap[imp.LocalName] = imp
	}

	if imp, ok := importMap["Hero"]; !ok || imp.Source != "./components/marketing/Hero.kiw" {
		t.Errorf("missing or invalid Hero import: %+v", imp)
	}
	if imp, ok := importMap["ui"]; !ok || !imp.IsNamespace {
		t.Errorf("missing or invalid ui namespace import: %+v", imp)
	}
	if imp, ok := importMap["Button"]; !ok || imp.Source != "./Button.kiw" {
		t.Errorf("missing or invalid Button import: %+v", imp)
	}
	if imp, ok := importMap["CustomModal"]; !ok || imp.ImportedName != "Modal" {
		t.Errorf("missing or invalid CustomModal import: %+v", imp)
	}
	if imp, ok := importMap["Nav"]; !ok || imp.Source != "./Nav.kiw" {
		t.Errorf("missing or invalid Nav directive import: %+v", imp)
	}

	// Verify props & defaults
	if len(mod.Props) < 4 {
		t.Fatalf("expected at least 4 props, got %d: %+v", len(mod.Props), mod.Props)
	}
	if mod.Props["headline"].Default != "Default Headline" || !mod.Props["headline"].Required {
		t.Errorf("invalid headline prop: %+v", mod.Props["headline"])
	}
	if mod.Props["variant"].Default != "primary" {
		t.Errorf("invalid variant prop: %+v", mod.Props["variant"])
	}
	if mod.Defaults["variant"] != "primary" {
		t.Errorf("invalid variant default: %v", mod.Defaults["variant"])
	}

	// Verify exports
	if len(mod.Exports) == 0 {
		t.Fatalf("expected exports, got none")
	}
}

func TestSlotFallbacksDesugaring(t *testing.T) {
	componentSrc := `
<div class="card">
  <div class="header">
    <slot name="header">
      <h3>Default Card Title</h3>
    </slot>
  </div>
  <div class="body">
    <slot>
      <p>Default card body</p>
    </slot>
  </div>
  <div class="footer">
    <slot name="footer" />
  </div>
</div>
`
	out, err := DesugarTemplate(componentSrc)
	if err != nil {
		t.Fatalf("DesugarTemplate error: %v", err)
	}

	// Check named slot fallback
	if !strings.Contains(out, `{{if .Header}}{{.Header}}{{else}}`) || !strings.Contains(out, `<h3>Default Card Title</h3>`) {
		t.Errorf("expected named slot header with fallback, got:\n%s", out)
	}
	// Check default slot fallback
	if !strings.Contains(out, `{{if .Body}}{{.Body}}{{else}}`) || !strings.Contains(out, `<p>Default card body</p>`) {
		t.Errorf("expected default slot body with fallback, got:\n%s", out)
	}
	// Check self-closing named slot
	if !strings.Contains(out, `{{.Footer}}`) {
		t.Errorf("expected self-closing slot footer to desugar to {{.Footer}}, got:\n%s", out)
	}
}

func TestSnippetsAndRender(t *testing.T) {
	src := `
{#snippet userBadge(name, role)}
  <span class="badge {role}">{name}</span>
{/snippet}

<div class="team">
  {@render userBadge("Alice", "admin")}
  {@render userBadge("Bob", "editor")}
</div>
`
	out, err := DesugarTemplate(src)
	if err != nil {
		t.Fatalf("DesugarTemplate error: %v", err)
	}

	if !strings.Contains(out, `{{define "userBadge"}}`) {
		t.Errorf("expected snippet define, got:\n%s", out)
	}
	if !strings.Contains(out, `{{template "userBadge" (dict "Name" "Alice" "Role" "admin")}}`) {
		t.Errorf("expected template render call with dict, got:\n%s", out)
	}
}

func TestStandalonePackage_PackAndUnpack(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiw-pkg-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	pkgSrcDir := filepath.Join(tempDir, "my-ui")
	if err := os.MkdirAll(filepath.Join(pkgSrcDir, "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pkgSrcDir, "styles"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := &PackageManifest{
		Name:        "@acme/my-ui",
		Version:     "1.2.0",
		Description: "Acme modular UI library",
		Author:      "Acme Engineers",
		License:     "MIT",
		Components: map[string]string{
			"Button":      "./components/Button.kiw",
			"Card.Header": "./components/CardHeader.kiw",
		},
		Styles: "./styles/theme.css",
	}

	manifestPath := filepath.Join(pkgSrcDir, PackageManifestFile)
	if err := SavePackageManifest(manifestPath, manifest); err != nil {
		t.Fatalf("SavePackageManifest error: %v", err)
	}

	btnContent := `<button class="btn"><slot /></button>`
	if err := os.WriteFile(filepath.Join(pkgSrcDir, "components", "Button.kiw"), []byte(btnContent), 0o644); err != nil {
		t.Fatal(err)
	}
	cssContent := `.btn { background: blue; }`
	if err := os.WriteFile(filepath.Join(pkgSrcDir, "styles", "theme.css"), []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pack package
	archivePath := filepath.Join(tempDir, "acme-my-ui-1.2.0.kpk")
	outPath, sha, err := PackPackage(pkgSrcDir, archivePath)
	if err != nil {
		t.Fatalf("PackPackage error: %v", err)
	}
	if outPath != archivePath {
		t.Errorf("outPath = %q, want %q", outPath, archivePath)
	}
	if sha == "" {
		t.Errorf("expected non-empty sha256")
	}

	// Unpack package
	unpackDir := filepath.Join(tempDir, "installed", "acme-my-ui")
	unpackedManifest, err := UnpackPackage(archivePath, unpackDir)
	if err != nil {
		t.Fatalf("UnpackPackage error: %v", err)
	}

	if unpackedManifest.Name != "@acme/my-ui" || unpackedManifest.Version != "1.2.0" {
		t.Errorf("manifest mismatch: %+v", unpackedManifest)
	}

	readBtn, err := os.ReadFile(filepath.Join(unpackDir, "components", "Button.kiw"))
	if err != nil {
		t.Fatalf("read unpacked button: %v", err)
	}
	if string(readBtn) != btnContent {
		t.Errorf("unpacked button content mismatch: got %q want %q", string(readBtn), btnContent)
	}
}
