# Package: `ui` & `tui` (Web Components & Terminal UI)

The `packages/ui` and `packages/tui` libraries provide composable UI primitives for the browser and the terminal respectively.

---

## 1. Web UI Toolkit (`packages/ui`)

Shared components and design tokens usable in both SSR templates and WASM components:
- **`ui/components`**: Button, Alert, Badge, Card, Avatar, Breadcrumb.
- **`ui/form`**: Form, Input, Checkbox, Select, Textarea with validation error binding.
- **`ui/panel`**: Header, Sidebar, Drawer, Modal, Container layout panels.

---

## 2. Terminal UI Toolkit (`packages/tui`)

Powers the `kiw` CLI application:
- **Command Routing**: Declarative subcommand trees with group categorizations and examples (`tui.NewApp`, `tui.NewCommand`).
- **Interactive Prompts**: Text input, password masking, single select menus, multi-select checkboxes.
- **Terminal Formatting (`packages/term`)**: Colorized text output, ASCII art, status spinners, and progress meters.

```go
app := tui.NewApp("my-tool", "v1.0.0").
	WithDescription("Developer CLI").
	Command(tui.NewCommand("run", "Execute task", nil, runHandler).
		WithGroup("develop").
		WithExample("my-tool run"))
```
