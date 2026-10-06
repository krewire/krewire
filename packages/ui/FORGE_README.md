# Krewire Forge 🔨

**Programmatic App Builder for the Krewire Ecosystem.**

Forge provides a pure Go, zero-npm, zero-bloat toolkit for constructing interactive web applications, internal tools, administrative dashboards, forms, and custom UI components programmatically.

---

## Features

- **TailwindCSS by Default**: Zero-setup utility-first styling powered by TailwindCSS out-of-the-box, pre-configured with Krewire pop-brutalist theme tokens and dark mode support.
- **Built-in UI Component & Widget System**: Rich set of primitives (`Button`, `Badge`, `Card`, `Alert`, `Table`, `Modal`, `Divider`) and responsive layouts (`VStack`, `HStack`, `Grid`).

### Atomic `.kiw` Components

`components/` ships atomic components as `.kiw` modules. They are embedded
(`components.FS`) and loaded automatically into every file-based site, so a
layout can use them by name with no import:

```html
<Breadcrumb>
  <BreadcrumbItem Label="Home" Href="/" />
  <BreadcrumbItem Label="Docs" Current="true" />
</Breadcrumb>

<Progress Value="42" ShowValue="true" />
<Avatar Initials="RS" Alt="Rina" />
```

| Group | Components |
| ----- | ---------- |
| Text & inline | `Text`, `Code`, `Kbd`, `List` |
| Layout | `Box`, `Stack`, `Flex`, `Grid`, `Container`, `Section`, `Divider` |
| Display | `Card`, `Badge`, `Alert`, `Callout`, `Avatar`, `Progress`, `Spinner` |
| Navigation | `Navbar`, `NavMenu`, `Breadcrumb` + `BreadcrumbItem`, `Sidebar`, `SideMenu`, `Drawer` |
| Forms | `Form`, `Input`, `Textarea`, `Select`, `Checkbox`, `Radio`, `Toggle` |
| Overlays | `Modal`, `Tooltip`, `Collapse`, `Tabs` |
| Feedback | `Spinner`, `Alert`, `Progress` |

Components nest: a child tag inside a parent's body is resolved, not printed as
text. Disclosure (`Collapse`) is built on `<details>` and `Spinner` is pure CSS,
so both work with no JavaScript.
- **Form Builder**: Declarative field construction, type-safe validation, error messages, and request binding (`TextField`, `Email`, `Number`, `Select`, `Checkbox`, `Textarea`).
- **Panel & Dashboard Builder**: Clean metric stat cards (`Stat`), content containers (`Panel`), and responsive dashboards (`Dashboard`).
- **Application Engine**: Programmatic routing, page shell generation, and cohesive theming with light/dark palettes and toggle runtime.
- **Zero JS Fatigue**: Server-rendered HTML with progressive enhancement, zero client bundlers, and sub-millisecond cold starts.

---

## Installation

```bash
go get github.com/krewire/forge
```

---

## Quick Start

### 1. Simple Programmatic App

```go
package main

import (
	"net/http"

	"github.com/krewire/forge"
	"github.com/krewire/forge/widget"
)

func main() {
	app := forge.New("Operations Hub").
		AddNav("Home", "/").
		Page("/", "Dashboard", widget.NewCard("Welcome to Forge").
			WithDesc("Programmatic Go App Builder").
			WithBody(widget.NewText("Built with pure Go and zero runtime bloat.").Strong()))

	http.ListenAndServe(":8080", app)
}
```

### 2. Form Builder

```go
f := form.New("/users/create", "POST").
	Add(form.NewText("name", "Full Name").MakeRequired()).
	Add(form.NewText("email", "Email Address").MakeRequired().AsEmail()).
	Add(form.NewSelect("role", "Role",
		form.SelectOption{Value: "member", Label: "Member"},
		form.SelectOption{Value: "admin", Label: "Administrator"},
	))

app.Form("/users/create", f, func(vals map[string]string) (widget.Widget, error) {
	// Process submission
	return widget.NewAlert("User " + vals["name"] + " created successfully!").Success(), nil
})
```

### 3. Dashboard & Panels

```go
dash := panel.NewDashboard("Platform Metrics").
	AddStat(
		panel.NewStat("Active Users", "12,450").WithChange("14.2%", true).WithIcon("👥"),
		panel.NewStat("Monthly Revenue", "$94,300").WithChange("8.5%", true).WithIcon("💰"),
	).
	AddPanel(
		panel.New("System Status").
			WithActions(widget.NewButton("Refresh").Small()).
			WithBody(widget.NewBadge("All Systems Operational").Success()),
	)

app.Dashboard("/metrics", dash)
```

---

## Ecosystem Integration

Forge is an official sub-product in the Krewire ecosystem, designed to work seamlessly with:
- [`krewire/framework`](https://github.com/krewire/framework): Fullstack web monoliths, HTTP routing, and workers.
- [`krewire/libs`](https://github.com/krewire/libs): Core utilities, resilience, and logging.
- [`krewire/kiw`](https://github.com/krewire/kiw): CLI devtool, scaffolding, and builds.

---

## License

MIT © Krewire Contributors.
