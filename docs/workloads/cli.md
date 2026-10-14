# Workload: Command-Line Tool (`cli`)

The `cli` workload targets standalone developer utilities, administrative commands, and interactive terminal applications.

---

## 1. Key Features

- **`packages/tui`**: Declarative command tree structure, subcommands, flag binding, and interactive prompts.
- **`packages/term`**: ANSI color styling, cursor control, spinners, tables, and box drawing.
- **`packages/kern/errs`**: Standard exit code mapping (`0` for success, `1` for general error, `2` for usage error, `78` for config error).

---

## 2. Project Layout

```text
my-cli/
├── krewire.yaml          # Project configuration (kind: cli)
├── main.go               # Command setup & application execution
└── cmd/                  # Subcommand definitions
    ├── root.go
    └── run.go
```

---

## 3. Implementation Example

```go
package main

import (
	"os"
	"github.com/krewire/krewire/packages/tui"
)

func main() {
	app := tui.NewApp("my-cli", "v1.0.0").
		WithDescription("A high-performance Krewire CLI utility").
		Command(tui.NewCommand("hello", "Say hello", nil, func(ctx *tui.Context) int {
			ctx.Println("Hello from Krewire CLI!")
			return 0
		}))

	os.Exit(int(app.Run(os.Args[1:])))
}
```

---

## 4. Workload Commands

```bash
# Test the CLI locally
kiw run hello

# Build standalone binary
kiw build

# Run linting and vet
kiw vet
```
