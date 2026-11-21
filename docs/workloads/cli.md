# Workload: Command-Line Tool (`cli`)

The `cli` workload targets standalone developer utilities, administrative tools, system automation scripts, and interactive terminal applications powered by `pkg/tui`, `pkg/term`, and `pkg/errs`.

It produces static, single-binary executables with zero external shared library dependencies (`CGO_ENABLED=0`), sub-millisecond startup times, and deterministic POSIX exit codes.

---

## 1. Architectural Capabilities

- **Declarative Command Tree**: Nested subcommands, positional argument validation, and typed flags (`StringFlag`, `BoolFlag`, `IntFlag`).
- **ANSI Terminal Styling & TTY Detection**: Respects `isatty`, `NO_COLOR`, and CI environments, auto-disabling color escape codes when stdout is redirected.
- **Interactive Terminal Widgets**: Built-in progress spinners, multi-step progress bars, table formatters, and prompts.
- **Typo Suggestions**: Levenshtein distance matching automatically recommends valid commands when a user makes a typo (`Did you mean "build"?`).
- **POSIX Standard Exit Codes**: Typed exit code contracts (`0` success, `1` runtime error, `2` invalid argument/usage).

---

## 2. Standard Directory Layout

Scaffolded via `kiw new <tool-name> --cli` or `kiw init --cli`:

```text
my-tool/
├── krewire.yaml          # Project configuration manifest
├── main.go               # Application entrypoint & command registration
├── cmd/                  # (Optional) Alternative binary entrypoints
└── internal/
    ├── commands/         # Command handler implementations
    │   ├── deploy.go
    │   └── status.go
    └── config/           # CLI configuration & local cache
```

---

## 3. Configuration Contract (`krewire.yaml`)

```yaml
project:
  name: "my-tool"
  kind: "cli"
  version: "1.0.0"

output: ".krewire/build"

build:
  flags: "-ldflags='-s -w'" # Strip debug symbols for minimal binary size
```

---

## 4. Complete CLI Application Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/krewire/krewire/pkg/errs"
    "github.com/krewire/tui/term"
    "github.com/krewire/tui"
)

func main() {
    app := tui.NewApp("my-tool", "1.0.0", "Developer Infrastructure CLI")

    // Subcommand: status
    statusCmd := tui.NewCommand("status").
        Description("Check server cluster connectivity").
        Aliases("st").
        Flag(tui.StringFlag{
            Name:    "host",
            Short:   "H",
            Default: "127.0.0.1",
            Usage:   "Target host address",
        }).
        Action(func(c *tui.Context) error {
            host := c.String("host")

            spinner := c.NewSpinner("Pinging cluster...")
            spinner.Start()
            // simulate ping
            spinner.Success(fmt.Sprintf("Cluster at %s is healthy!", host))

            c.Banner("Cluster Status: HEALTHY", tui.ColorGreen)
            return nil
        })

    app.AddCommand(statusCmd)

    if err := app.Run(context.Background(), os.Args[1:]); err != nil {
        tui.ErrorExit(err)
    }
}
```

---

## 5. Build & Cross-Compilation Pipelines

```bash
# 1. Run command during development
kiw run status --host 192.168.1.10

# 2. Compile static binary for current platform into .krewire/build/bin/my-tool
kiw build

# 3. Cross-compile static Linux and Windows binaries
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/my-tool-linux-amd64 .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/my-tool-windows-amd64.exe .
```

---

## 6. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Terminal escape injection prevention; arguments and flags are validated against injection patterns; secret parameters (passwords, tokens) are hidden during interactive prompts.
- **Sustainable (S2)**: Single static binary (typically 5–10 MB); zero external ncurses or C libraries; runs cleanly on bare Linux without dependencies.
- **Scalable (S3)**: Command tree scales additively from a single command utility to an enterprise multi-subcommand suite.
