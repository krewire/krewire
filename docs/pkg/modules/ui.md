# `ui`

Import: `github.com/krewire/krewire/pkg/ui`

## Purpose

Web UI component library, layout panels, design tokens, and CSS theme palettes for SSR templates and client components.

## Subpackages & Components

- **Components**: `Button`, `Badge`, `Card`, `Drawer`, `Hamburger`, `Header`, `Hero`, `Footer`, `Nav`, `Section`, `ThemeToggle`.
- **`ui/form`**: Form controls, text input, select, checkbox, and validation error binding.
- **`ui/panel`**: Header, sidebar, modal, and drawer panels.
- **Theme & Tokens**: CSS variables, theme switching (`auto`, `light`, `dark`), and semantic color palettes.

## Example

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/pkg/ui"
)

func main() {
	btn := ui.Button(ui.ButtonProps{
		Label:   "Submit",
		Variant: "primary",
	})
	fmt.Println(btn.Render())
}
```

## Design boundary

Pure Go UI definition. Standard HTML/CSS output. See the full [UI Reference](../web-runtime/ui-tui.md).
