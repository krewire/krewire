# Error & Diagnostic Primitives (`pkg/errs`)

Import: `github.com/krewire/krewire/pkg/errs`

## Purpose

The `pkg/errs` package provides POSIX-compatible exit codes, structured diagnostic attributes, caller stack trace capture, actionable user hints, and diagnostic tree rendering with zero third-party dependencies.

---

## Architectural Position

- **Floor Invariant (Layer 1)**: Imports standard library packages only (`errors`, `fmt`, `runtime`, `strings`).
- **Standard Unwrapping**: All wrapped error types implement `Unwrap() error`, ensuring full compatibility with standard `errors.Is` and `errors.As`.

---

## Process Exit Codes

| Constant | Value | Purpose |
| :--- | :--- | :--- |
| **`ExitCodeSuccess`** | `0` | Successful execution |
| **`ExitCodeFailure`** | `1` | Generic runtime execution failure or unhandled exception |
| **`ExitCodeUsage`** | `2` | Command-line syntax error, missing argument, or invalid flag |

### Functions:
- `(c ExitCode) Int() int` — Returns the integer exit code.
- `ExitCodeFromInt(code int) ExitCode` — Maps a raw process status code to the closest known `ExitCode`.

---

## Error Constructors & Diagnostics

### Basic Error Types
- `NewError(message, code)` — Constructs an error with an explicit exit code.
- `UsageError(message)` — Shortcut for `NewError(message, ExitCodeUsage)`.
- `FailureError(message)` — Shortcut for `NewError(message, ExitCodeFailure)`.

### Attaching Diagnostics
- **`WithAttrs(err, attrs...)`**: Attaches structured key/value attributes (`Attr{Key, Value}`) without mutating the underlying error message.
- **`AttrsOf(err)`**: Traverses the wrap chain and extracts all accumulated attributes from outermost to innermost.
- **`WithHint(err, hint)`**: Attaches an actionable user recommendation (e.g. `"Run 'krew init' to initialize the workspace"`).
- **`HintOf(err)`**: Extracts the nearest hint from the error wrap chain.
- **`WithStack(err)`**: Captures the current execution call stack (up to 32 frames).
- **`FormatTree(err)`**: Formats the complete error chain, attached attributes, stack frames, and hints into a human-readable diagnostic report.

---

## Example Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/krewire/krewire/pkg/errs"
)

func validateConfig(path string) error {
	if path == "" {
		err := errs.UsageError("configuration file path is required")
		err = errs.WithAttrs(err, errs.Attr{Key: "env", Value: "local"})
		err = errs.WithHint(err, "Specify --config path/to/krewire.yaml")
		return errs.WithStack(err)
	}
	return nil
}

func main() {
	if err := validateConfig(""); err != nil {
		// Print formatted diagnostic tree
		fmt.Fprintln(os.Stderr, errs.FormatTree(err))

		// Check exit code
		if e, ok := err.(*errs.Error); ok {
			os.Exit(e.ExitCode().Int())
		}
		os.Exit(errs.ExitCodeFailure.Int())
	}
}
```
