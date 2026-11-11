# `kiw` (DSL Engine)

Import: `github.com/krewire/krewire/pkg/kiw`

## Purpose

The parser and desugaring engine for the `.kiw` component template dialect.

## Features

- **YAML Frontmatter**: Extracted into `KiwModule.Frontmatter` (route metadata, title, layout, props, imports).
- **JSX-like Component Desugaring**: Transforms `<Card title="Hello" />` into Go template calls `{{component "Card" ...}}`; `{{mount "Name" "mode" ...}}` is also recognized.
- **Compound & Vendor Tags**: Seamlessly parses compound dotted tags (`<Card.Header />`) and vendor namespaces (`<ui:Button />` or `<acme:Header />`).
- **Slot Mechanics with Fallback**: Supports `<slot name="header">Fallback</slot>` and `<slot>Default Body</slot>` with automatic resolution.
- **Inline Snippets / Macros**: Authored with `{#snippet name(args)}...{/snippet}` and invoked with `{@render name(args)}`.
- **Declarative Imports**: Supports `<script>` block ES-like imports, frontmatter `imports:`, and `@import` directives.
- **Standalone Component Packages**: Built-in `.kpkg` pack/unpack engine with `kiw.pkg.json` manifests and `sec.SafePath` traversal guards.
- **Scoped `<style>`**: Captured as `StyleBlock` values with `Lang`, `Scoped`, and `Content`.
- **Multi-language `<script>`**: Captured as `ScriptBlock` values with `Lang` (`js|ts|go|rust`), `Hydrate` (`load|idle|visible`), `Server`, and `Compute` flags.
- **`<markdown>` Blocks**: Rendered through `pkg/markdown` at parse time.
- **Component inventory**: `KiwModule.Components` and `ComponentNames` record every invocation for asset and hydration planning.

## Props and casing convention

Props are the attributes of a component tag. Inside a component, **read props
lower-camel** — `{variant}`, `{ariaLabel}`, `{#if title}` — which desugar to
`.Variant`, `.AriaLabel`, `.Title`. Callers may spell a prop however is natural;
the renderer stores every prop under all aliases from `kiw.PropAliases`:

| Written by the caller | Also visible as |
|---|---|
| `variant="x"` / `Variant="x"` | `variant`, `Variant` |
| `aria-label="x"`, `aria_label="x"` | `ariaLabel`, `AriaLabel` |
| `id="x"`, `url="x"` | `Id`/`ID`, `Url`/`URL` |

An alias never overwrites a prop that was passed explicitly. The child content
of a paired tag is the `body` prop (`{body}`); `<slot name="footer">` passes the
`footer` prop. `kiw make component <Name>` generates a component that follows
this convention.

## Loops

`{#each items as item, i}…{/each}` blocks may nest. An inner block sees every
enclosing loop variable (`{#each row as cell}` ranges over `$row`), and outer
props stay reachable (`{title}` inside a loop reads `$.Title`).

## Compound & Namespaced Components

The parser desugars dotted and colon notations into Go template component invocations:

```html
<!-- Dotted compound notation -->
<Card>
  <Card.Header title="Dashboard Overview" />
  <Card.Body>Metrics content...</Card.Body>
</Card>

<!-- Namespaced / Vendor notation -->
<ui:Button variant="primary">Save Changes</ui:Button>
<acme:Badge status="live" />
```

Dotted compound components resolve to subdirectories in `components/` (e.g. `<Card.Header />` resolves to `components/Card/Header.kiw` or `components/CardHeader.kiw`), while namespaced tags resolve to installed component packages in `.krewire/pkg/` or embedded UI components.

## Fallback Slot Mechanics

Slot outlets in component templates can declare fallback content that renders automatically when the caller provides no slot data:

```html
<!-- components/Card.kiw -->
<div class="card">
  <div class="card-header">
    <slot name="header"><h3>Default Card Header</h3></slot>
  </div>
  <div class="card-body">
    <slot><p>Default card body content goes here.</p></slot>
  </div>
</div>
```

Callers can populate slots via `<template slot="header">`, `<div slot="header">`, or `<slot:header>`.

## Inline Snippets (Local Macros)

Reusable template fragments can be declared and rendered inside a single `.kiw` file without creating external files:

```html
{#snippet tagBadge(label, color)}
  <span class="badge badge-{color}">{label}</span>
{/snippet}

<div class="tags">
  {@render tagBadge("Go", "blue")}
  {@render tagBadge("SSG", "green")}
</div>
```

Snippets desugar into scoped Go template definitions and invocations.

## Declarative Imports

Components can declare imports via `<script>` blocks, YAML frontmatter, or top-level `@import`:

```html
<script>
  import Button from './Button.kiw';
  import { Badge } from '@krewire/ui';
</script>
```

## Standalone Component Packages (`.kpkg`)

`pkg/kiw` provides a complete standalone package packaging and unpacking pipeline:
- **Canonical Manifest (`kiw.pkg.json`)**: Declares package name, version, exported components mapping, and stylesheet.
- **Archive Format (`.kpkg`)**: Gzipped tarball with SHA-256 integrity verification.
- **Directory Layout**: Installed into `.krewire/pkg/` by default (`kiw.DefaultPackageDir`).
- **Security (`sec.SafePath`)**: Unpacking strictly enforces traversal guards to prevent ZipSlip/TarSlip vulnerabilities.

## Diagnostics

Template errors are reported at `.kiw` source positions instead of positions
in the generated Go template:

```text
pages/index.kiw:6:16: {#if on} inside an attribute of <Button>: component props cannot contain control blocks
  hint: pass the value as a prop (label={label}) and branch inside the component, or wrap the whole tag in {#if}…{:else}…{/if}
```

- `kiw.Lint(file, src)` reports, with exact line and column: control blocks
  inside component tag attributes, unbalanced or mismatched `{#if}`/`{#each}`
  blocks, and invalid YAML frontmatter. `kiw vet` runs it on every `.kiw` file
  of the project.
- `kiw.Explain(file, name, src, funcs, err)` maps a `text/template` or
  `html/template` parse or execution error back to the source line (via the
  line-preserving `kiw.AlignedBody`) and column, and adds a hint.
  `pkg/web/ssg` does this for every component, page and layout it loads from
  `.kiw` files, so `kiw build` and `kiw dev` print source positions.

## Editor support

A TextMate grammar and VS Code extension for `.kiw` live in
[`tools/kiw/editors/vscode`](../../../tools/kiw/editors/vscode): frontmatter,
template blocks, interpolation, component tags, scoped `<style>`, `<script>`
and `<markdown>`, plus bracket matching and folding.

## Main API

- `kiw.ParseKiw(src string) (*KiwModule, error)`: Parses a `.kiw` source string.
- `kiw.ParseKiwFile(path string) (*KiwModule, error)`: Reads and parses a file.
- `kiw.DesugarTemplate(src string) (string, error)` / `kiw.DesugarTemplateWithScope(src, loopVars)`: Rewrites component tags into template calls.
- `kiw.Lint(file, src string) []Diagnostic`, `kiw.Explain(file, name, src string, funcs map[string]any, err error) error`, `kiw.AlignedBody(src string) string`: diagnostics (see above).
- `kiw.PropAliases(name string) []string`: the lenient prop-casing aliases.
- `kiw.PackPackage(dir, outPath string) (archivePath, sha256Hex string, err error)`: Bundles a component directory into a `.kpkg` archive.
- `kiw.UnpackPackage(archivePath, destDir string) (*PackageManifest, error)`: Safely unpacks a `.kpkg` archive with `sec.SafePath` bounds.
- `kiw.LoadPackageManifest(path string) (*PackageManifest, error)`: Decodes and validates `kiw.pkg.json`.
- `kiw.SavePackageManifest(path string, m *PackageManifest) error`: Writes a validated `PackageManifest` to disk.
- Constants: `kiw.PackageExtension` (`.kpkg`), `kiw.DefaultPackageDir` (`.krewire/pkg`), `kiw.PackageManifestFile` (`kiw.pkg.json`).
- `kiw.KiwModule`: fields `Frontmatter`, `Body`, `Styles`, `Scripts`, `StyleBlocks`, `ScriptBlocks`, `Markdown`, `Components`, `ComponentNames`, and `Raw`.

## Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/krewire/krewire/pkg/kiw"
)

func main() {
	source := `---
title: Welcome
---
<Card title="Greeting">
  <p>Hello from .kiw DSL</p>
</Card>`

	mod, err := kiw.ParseKiw(source)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Title: %v\nComponents: %v\n", mod.Frontmatter["title"], mod.ComponentNames)
}
```

## Design boundary

Pure Go parser. It is imported by `pkg/web/ssg` and `tools/kiw`; it never imports `tools/kiw`.
