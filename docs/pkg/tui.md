# Package Reference: `tui`

Import: `github.com/krewire/tui`

## 1. Purpose & Architecture

The `pkg/tui` package is the Terminal UI application builder powering the `krew` CLI tool. It provides command dispatching, flag parsing, subcommands, interactive terminal spinners, styled banners, progress bars, and Levenshtein-based typo suggestions.

For detailed TUI architecture and examples, see [Web UI & Terminal UI Guide](./web-runtime/ui-tui.md).

---

## 2. Main API Reference

- **`tui.NewApp(name, version, description) *App`**: Creates a CLI application entry point.
- **`tui.NewCommand(name) *Command`**: Creates a subcommand with flags, aliases, and action handlers.
- **`tui.StringFlag` / `tui.BoolFlag` / `tui.IntFlag`**: Strongly typed CLI flag definitions.
- **`ctx.NewSpinner(label)`**: Interactive terminal spinner for long-running tasks.
- **`ctx.Banner(title, color)` / `ctx.Success(...)` / `ctx.Error(...)`**: Styled ANSI terminal output.
- **`tui.Suggest(input, candidates)`**: Levenshtein distance typo suggestion for mistyped commands.

---

## 3. Usage Example

```go
package main

import (
    "context"
    "os"
    "github.com/krewire/tui"
)

func main() {
    app := tui.NewApp("tool", "1.0.0", "Custom Developer Tool")

    cmd := tui.NewCommand("build").
        Description("Build static assets").
        Flag(tui.BoolFlag{Name: "prod", Usage: "Production build"}).
        Action(func(c *tui.Context) error {
            c.Banner("Starting Build", tui.ColorGreen)
            return nil
        })

    app.AddCommand(cmd)
    _ = app.Run(context.Background(), os.Args[1:])
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Sanitizes ANSI control codes; prevents terminal escape sequence injection attacks.
- **Sustainable (S2)**: Zero CGO dependencies; compiles to static pure-Go binaries without ncurses.
- **Scalable (S3)**: Reusable command and flag models for all ecosystem CLI tools.
