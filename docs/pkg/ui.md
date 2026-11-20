# Package Reference: `ui`

Import: `github.com/krewire/krewire/pkg/ui`

## 1. Purpose & Architecture

The `pkg/ui` package implements Krewire's **Pop-Brutalist** UI design system: high-contrast borders (`2.5px solid`), vibrant primary accents (`#39D353`), hard geometric drop-shadows (`4px 4px 0px var(--border)`), strongly typed widgets, declarative form builders, and accessible dark-mode inversion.

For full component specifications and design tokens, see [Web UI Component Toolkit Guide](./web-runtime/ui-tui.md).

---

## 2. Main API Reference

- **`widget.NewButton(label)`**: Creates button with variants (`Primary`, `Secondary`, `Destructive`, `Outline`, `Ghost`).
- **`widget.NewCard()`**: Pop-Brutalist card layout with Header, Body, and Footer slots.
- **`widget.NewModal(title)`**: Accessible modal dialog with backdrop blur and keyboard dismissal.
- **`widget.NewAlert(message, level)`**: Status alerts (`Success`, `Warning`, `Error`, `Info`).
- **`form.New(action, method)`**: Declarative form builder with CSRF tokens and field validation state.
- **`tokens.CSS()`**: Returns standard Pop-Brutalist CSS tokens and theme variables.

---

## 3. Usage Example

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/ui/widget"
)

func RenderUserProfileCard(username string) string {
    btn := widget.NewButton("Edit Profile").
        Variant(widget.VariantPrimary)

    card := widget.NewCard().
        Title("User Profile").
        AddChild(widget.NewText("Active Member: " + username)).
        Footer(btn)

    return card.Render()
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Automatic CSRF token injection in forms; strict contextual HTML escaping on all text widgets.
- **Sustainable (S2)**: Zero npm/Node.js runtime dependencies; components compile via pure Go `embed.FS`.
- **Scalable (S3)**: Components follow the "Copy & Own" model or direct Go module imports.
