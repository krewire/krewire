# `tui`

Import: `github.com/krewire/krewire/packages/tui`

## Purpose

Terminal UI toolkit powering the `kiw` CLI: declarative command routing, subcommand trees, help text generation, and interactive terminal prompts.

## Main API

- `tui.NewApp(name, version)`: Initializes a CLI command application.
- `tui.NewCommand(name, desc, registerFlags, runHandler)`: Declares a CLI command with group and examples.
- `app.Command(cmd)`: Mounts a subcommand into the application.
- `app.Run(args)`: Dispatches command execution and maps exit codes.

## Example

```go
package main

import (
	"os"
	"github.com/krewire/krewire/packages/errs"
	"github.com/krewire/krewire/packages/tui"
)

func main() {
	app := tui.NewApp("mycli", "1.0.0").
		Command(tui.NewCommand("hello", "Print greeting", nil, func(fs *flag.FlagSet) errs.ExitCode {
			println("Hello, World!")
			return errs.ExitCodeSuccess
		}))

	os.Exit(int(app.Run(os.Args[1:])))
}
```

## Design boundary

Independent presentation layer for terminal applications. Powers `tools/kiw`.
