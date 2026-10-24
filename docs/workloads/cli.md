# Workload: Command-Line Tool (`cli`)

The `cli` workload targets standalone developer utilities, administrative commands, and interactive terminal applications.

---

## 1. Key Features

- **`pkg/tui`**: Declarative command tree structure, subcommands, flag binding, and interactive prompts.
- **`pkg/term`**: ANSI color styling, cursor control, spinners, tables, and box drawing.
- **`pkg/errs`**: Standard exit code mapping (`0` success, `1` failure, `2` usage).

---

## 2. Project Layout

The `kiw init --cli` scaffold writes:

```text
my-cli/
├── krewire.yaml
├── main.go
├── internal/
│   ├── commands/         # Command definitions
│   └── config/           # Typed configuration
└── README.md
```

---

## 3. Implementation Example

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/tui"
)

func main() {
	app := tui.NewApp("my-cli", "v1.0.0").
		WithDescription("A high-performance Krewire CLI utility").
		Command(tui.NewCommand("hello", "Say hello", nil, func(fs *flag.FlagSet) errs.ExitCode {
			fmt.Println("Hello from Krewire CLI!")
			return errs.ExitCodeSuccess
		}))

	os.Exit(int(app.Run(os.Args[1:])))
}
```

---

## 4. Workload Commands

```bash
# Test the CLI locally
kiw run hello

# Build standalone binary into .krewire/build/bin/<name>
kiw build

# Run linting and vet
kiw vet
```
