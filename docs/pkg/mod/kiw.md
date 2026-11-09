# `kiw` (DSL Engine)

Import: `github.com/krewire/krewire/pkg/kiw`

## Purpose

The parser and desugaring engine for the `.kiw` component template dialect.

## Features

- **YAML Frontmatter**: Extracted into `KiwModule.Frontmatter` (route metadata, title, layout, props).
- **JSX-like Component Desugaring**: Transforms `<Card title="Hello" />` into Go template calls `{{component "Card" ...}}`; `{{mount "Name" "mode" ...}}` is also recognized.
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
