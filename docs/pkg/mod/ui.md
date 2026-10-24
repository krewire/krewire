# `ui`

Import: `github.com/krewire/krewire/pkg/ui`

## Purpose

Web UI component library, layout panels, design tokens, and CSS theme palettes for SSR templates and client components.

## Components

Every component is a plain struct with a `Render() template.HTML` method, so it composes in Go templates and in server-rendered pages.

- **Layout & chrome**: `Header` (`HeaderConfig`), `Footer` (`FooterConfig`), `Nav` (`NavConfig`), `Hero`, `Section`, `Container`, `Grid`, `Layout` (`LayoutConfig`).
- **Content**: `Button` (`Text`, `URL`, `Variant`, `Target`, `Rel`), `Badge` (`Text`, `Variant`), `Card` (`Header`, `Body`, `Footer`), `ThemeToggle`.
- **`ui/form`**: Form controls, text input, select, checkbox, and validation error binding.
- **`ui/panel`**: Header, sidebar, modal, and drawer panels.
- **Theme & tokens**: `Theme` (`Default` = `auto|light|dark`, `Light`/`Dark` `Palette` overrides), `DefaultLightPalette`/`DefaultDarkPalette`, and per-component CSS constants (`ButtonCSS`, `CardCSS`, `HeroCSS`, …).
- `ui.ComponentsCSS()` returns the combined built-in stylesheet; `ui.AllBuiltinNames()` lists registered components.
- `ui.New(title)` returns an `App` that serves a dashboard, pages, and forms over HTTP with optional Tailwind integration.

## Example

```go
package main

import (
	"fmt"

	"github.com/krewire/krewire/pkg/ui"
)

func main() {
	btn := ui.Button{
		Text:    "Submit",
		URL:     "/submit",
		Variant: "primary",
	}
	fmt.Println(btn.Render())
}
```

## Design boundary

Pure Go UI definition. Standard HTML/CSS output. See the full [UI Reference](../web-runtime/ui-tui.md).
