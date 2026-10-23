# `kiw` (DSL Engine)

Import: `github.com/krewire/krewire/packages/kiw`

## Purpose

The compiler, parser, AST builder, and desugaring engine for the `.kiw` component template dialect.

## Features

- **YAML Frontmatter**: Extracts route metadata, page title, layout definitions, and props.
- **JSX-like Component Desugaring**: Transforms `<Card title="Hello" />` into Go template calls `{{component "Card" (dict "title" "Hello")}}`.
- **Scoped `<style>`**: Generates deterministic CSS scope hashes (`data-kiw-*`).
- **Multi-language `<script>`**: Isolates browser client logic and WebAssembly bridge scripts.
- **`<markdown>` Blocks**: Compiles embedded Markdown into HTML at build time.

## Example

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/packages/kiw"
)

func main() {
	source := `---
title: Welcome
---
<Card title="Greeting">
  <p>Hello from .kiw DSL</p>
</Card>`

	page, err := kiw.Parse(source)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Title: %s\nHTML: %s\n", page.Frontmatter["title"], page.HTML)
}
```

## Design boundary

Pure Go compiler. It is imported by `packages/web/ssg` and `tools/kiw`. It never imports `tools/kiw`.
