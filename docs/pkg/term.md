# Package Reference: `term`

Import: `github.com/krewire/krewire/pkg/term`

## 1. Purpose & Architecture

The `pkg/term` package provides terminal I/O detection, ANSI color palettes, text styles, and output stream separation for Krewire CLI developer tools (`tools/kiw`, `pkg/tui`).

It automatically detects interactive TTY terminals (`isatty`) and suppresses color escape codes when stdout is redirected to a pipe, file, or non-interactive CI runner.

---

## 2. Main API Reference

### Colors & Styles

- **Colors**: `term.ColorBlack`, `ColorRed`, `ColorGreen`, `ColorYellow`, `ColorBlue`, `ColorMagenta`, `ColorCyan`, `ColorWhite`, `ColorDefault`.
- **Styles**: `term.StyleBold`, `StyleDim`, `StyleUnderline`.
- **`term.Paint(text string, color Color, styles []Style) string`**: Wraps text in ANSI escape codes, resetting with `\x1b[0m`.

### Terminal Abstraction (`term.Terminal`)

- **`term.New(stdout, stderr io.Writer) *Terminal`**: Creates a terminal wrapper that respects TTY capabilities.
- **`term.IsTerminal(w io.Writer) bool`**: Reports whether the writer is an interactive terminal.
- **`term.DetectColorSupport() bool`**: Checks `TERM`, `NO_COLOR`, and `CI` environment variables.

---

## 3. Usage Example

```go
package main

import (
    "os"
    "github.com/krewire/krewire/pkg/term"
)

func main() {
    t := term.New(os.Stdout, os.Stderr)

    // Automatically prints plain text if redirected to a file
    t.Println(t.Paint("Build Succeeded!", term.ColorGreen, []term.Style{term.StyleBold}))
    t.Errorln(t.Paint("Warning: missing configuration", term.ColorYellow, nil))
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Sanitizes ANSI control characters to prevent terminal escape injection attacks.
- **Sustainable (S2)**: Zero CGO dependencies; does not link against ncurses or external terminal libraries.
- **Scalable (S3)**: Respects standard conventions: `NO_COLOR=1`, `CLICOLOR_FORCE=1`, and CI detection.
