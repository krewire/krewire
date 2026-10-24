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

## Main API

- `kiw.ParseKiw(src string) (*KiwModule, error)`: Parses a `.kiw` source string.
- `kiw.ParseKiwFile(path string) (*KiwModule, error)`: Reads and parses a file.
- `kiw.DesugarTemplate(src string) (string, error)` / `kiw.DesugarTemplateWithScope(src, loopVars)`: Rewrites component tags into template calls.
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
