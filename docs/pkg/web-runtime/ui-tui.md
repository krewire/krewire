# Package: `ui` & `tui` (Web UI Component Toolkit & Terminal UI Engine)

The `pkg/ui` and `pkg/tui` libraries provide user interface authoring surfaces across two mediums: Pop-Brutalist web interfaces with pure Go runtime efficiency, and expressive, interactive terminal UI applications.

---

## 1. Architectural Map & Responsibility Boundaries

```text
pkg/
├── ui/                 # Pop-Brutalist Web UI & Component Architecture
│   ├── tokens.go       # CSS design tokens, palettes, brutalist borders & hard drop-shadows
│   ├── theme.go        # Theme management & dark/light mode toggle contracts
│   ├── app_builder.go  # Fluent web layout and application assembler
│   ├── widget/         # Strongly typed HTML widget primitives (Button, Card, Modal, Table, Alert)
│   ├── form/           # Declarative form builder (TextField, Select, Checkbox, Textarea)
│   ├── panel/          # Structured dashboard layouts & sidebar navigation panels
│   └── node/           # UI virtual node tree & .kiw template compiler adapter
└── tui/                # Terminal UI application builder powering the krew CLI
    ├── tui.go          # App & Command runner, flag parsing, lifecycle
    ├── help.go         # Styled ANSI help screens and usage formatters
    ├── output.go       # Terminal banners, spinners, progress bars & color styles
    └── suggest.go      # Typo distance calculation (Levenshtein) for command suggestions
```

---

## 2. Pop-Brutalist Web UI Engine (`pkg/ui`)

`pkg/ui` embodies the **Pop-Brutalism** design system: high-contrast borders (`2.5px solid`), vibrant primary accents (`#39D353`), hard geometric drop-shadows (`4px 4px 0px var(--border)`), and accessible dark-mode inversion.

### Strongly Typed Widgets (`pkg/ui/widget`)

Developers can construct accessible HTML interfaces directly in Go with typed widgets:

```go
package main

import (
    "fmt"
    "github.com/krewire/ui/widget"
)

func BuildDashboardCard() string {
    // 1. Create a Primary Action Button
    saveBtn := widget.NewButton("Save Changes").
        Variant(widget.VariantPrimary).
        Size(widget.SizeMedium).
        Attr("type", "submit")

    // 2. Create an Alert Banner
    alert := widget.NewAlert("Deployment successful!", widget.AlertSuccess).
        Dismissible(true)

    // 3. Assemble Card Layout
    card := widget.NewCard().
        Title("System Status").
        AddChild(alert).
        AddChild(widget.NewText("All cluster runners are operating normally.")).
        Footer(saveBtn)

    return card.Render()
}
```

### Declarative Form Engine (`pkg/ui/form`)

Construct validated HTML forms with automatic CSRF fields and error state styling:

```go
package main

import (
    "github.com/krewire/ui/form"
)

func CreateLoginForm(csrfToken string) string {
    f := form.New("/login", "POST").
        AddHidden("csrf_token", csrfToken).
        AddField(form.NewTextField("email", "Email Address").
            Required(true).
            Placeholder("developer@example.com")).
        AddField(form.NewTextField("password", "Password").
            Type("password").
            Required(true)).
        AddField(form.NewCheckboxField("remember", "Remember me on this machine")).
        SubmitLabel("Sign In")

    return f.Render()
}
```

### CSS Tokens & Scoped Styling (`pkg/ui/tokens.go`)

Tokens are distributed as native CSS custom properties without requiring CSS preprocessors:

```css
:root {
  --primary: #39D353;
  --primary-hover: #2eb946;
  --bg: #FFFFFF;
  --surface: #F9FAFB;
  --text: #111827;
  --border: #111827;
  --border-width: 2.5px;
  --shadow-offset: 4px 4px 0px var(--border);
  --radius: 6px;
}

[data-theme="dark"] {
  --bg: #121212;
  --surface: #1E1E1E;
  --text: #F3F4F6;
  --border: #374151;
  --shadow-offset: 4px 4px 0px #000000;
}
```

---

## 3. Terminal UI Toolkit (`pkg/tui`)

`pkg/tui` powers the `krew` devtool CLI and terminal applications, providing argument parsing, flags, subcommands, interactive prompts, spinners, and help documentation.

### Core Contracts & Types

```go
type App struct {
    Name        string
    Version     string
    Description string
    Commands    []*Command
}

type Command struct {
    Name        string
    Aliases     []string
    Description string
    Run         func(ctx *Context) error
    Flags       []Flag
}
```

### Complete CLI Application Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/krewire/tui"
)

func main() {
    app := tui.NewApp("mycli", "1.0.0", "Developer Automation CLI")

    deployCmd := tui.NewCommand("deploy").
        Description("Deploy project artifacts to remote targets").
        Aliases("dp").
        Flag(tui.StringFlag{
            Name:    "env",
            Short:   "e",
            Default: "staging",
            Usage:   "Target deployment environment",
        }).
        Flag(tui.BoolFlag{
            Name:    "dry-run",
            Usage:   "Simulate deployment without mutations",
        }).
        Action(func(c *tui.Context) error {
            env := c.String("env")
            dryRun := c.Bool("dry-run")

            c.Banner("Starting Deployment Pipeline", tui.ColorGreen)
            c.Info(fmt.Sprintf("Target Environment: %s (Dry Run: %t)", env, dryRun))

            // Run with interactive spinner
            spinner := c.NewSpinner("Connecting to cloud gateway...")
            spinner.Start()
            // simulate work...
            spinner.Success("Gateway connected!")

            c.Success("Deployment successfully completed!")
            return nil
        })

    app.AddCommand(deployCmd)
    if err := app.Run(context.Background(), os.Args[1:]); err != nil {
        tui.ErrorExit(err)
    }
}
```

### Levenshtein Typo Suggestions (`pkg/tui/suggest.go`)

If a user enters an unrecognized command (e.g. `kiw buld`), `pkg/tui` computes the Levenshtein edit distance and suggests the closest valid command:

```text
Error: unknown command "buld"
Did you mean "build"?

Run 'kiw --help' for usage.
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**:
  - Form widgets automatically include anti-CSRF token fields and enforce strict input escaping.
  - TUI sanitizes ANSI control sequences to prevent terminal escape injection attacks.
- **Sustainable (S2)**:
  - Web UI compiles in pure Go; styles and widgets compile in pure Go (`embed.FS`).
  - Terminal applications require no external ncurses/C libraries, preserving static pure-Go binaries (`CGO_ENABLED=0`).
- **Scalable (S3)**:
  - UI widgets and forms share the exact same validation rules and schemas defined in `pkg/validation`.
