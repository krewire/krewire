# Package: `pkg/errs`, `pkg/env` & `pkg/log` (Errors, Exit Codes & Logging)

This foundational library trio provides structured error categorization, diagnostic stack traces, POSIX-aligned process exit codes, target environment resolution, and structured `slog` logging.

---

## 1. Exit Codes & Error Types (`pkg/errs`)

Import: `github.com/krewire/krewire/pkg/errs`

Krewire maps errors deterministically to standard POSIX process exit codes with zero external dependencies:

| Exit Code Constant | Value | Purpose | Constructor / Mapping |
| :--- | :--- | :--- | :--- |
| `ExitCodeSuccess` | `0` | Successful command/process termination. | Process terminates normally |
| `ExitCodeFailure` | `1` | Generic runtime execution failure, upstream failure, or internal error. | `errs.FailureError(msg)` |
| `ExitCodeUsage` | `2` | Invalid command line arguments, unrecognized flags, or malformed input. | `errs.UsageError(msg)` |

### Creating and Inspecting Errors

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/pkg/errs"
)

func run() error {
	// Create an error carrying an explicit exit code
	return errs.UsageError("project name is required")
}

func main() {
	err := run()
	if err != nil {
		if e, ok := err.(*errs.Error); ok {
			fmt.Printf("Exiting with code %d: %s\n", e.ExitCode(), e.Error())
		}
	}
}
```

### Structured Diagnostic Attributes & Hints

`pkg/errs` allows attaching structured metadata and actionable hints without altering the underlying error message or breaking `errors.Is` / `errors.As`:

```go
// Attach structured diagnostic key/value pairs
err = errs.WithAttrs(err,
	errs.Attr{Key: "endpoint", Value: "/api/v1/users"},
	errs.Attr{Key: "status", Value: 503},
)

// Attach an actionable user hint
err = errs.WithHint(err, "Run 'krew doctor' to verify local toolchain prerequisites")

// Attach captured call stack trace
err = errs.WithStack(err)

// Render full diagnostic tree
fmt.Println(errs.FormatTree(err))
```

---

## 2. Target Environment Resolution (`pkg/env`)

Import: `github.com/krewire/krewire/pkg/env`

`pkg/env` defines the canonical environments supported across the Krewire ecosystem:

- **`env.EnvLocal`** (`"local"`): Default for local developer machines.
- **`env.EnvProduction`** (`"production"`): Live production deployment environment.
- **`env.EnvTesting`** (`"testing"`): Automated testing environment (`go test`).

### Resolving Environments

```go
import "github.com/krewire/krewire/pkg/env"

// Parse from string (case-insensitive, trims whitespace)
targetEnv, err := env.ParseEnv("production")
if err != nil {
    // Returns errs.UsageError if invalid
    return err
}

// Canonical list of all environments
validEnvs := env.Envs() // []Env{EnvLocal, EnvProduction, EnvTesting}
```

---

## 3. Structured Logging (`pkg/log`)

Import: `github.com/krewire/krewire/pkg/log`

`pkg/log` configures the process-wide structured logger based on Go's standard `log/slog`:

- **Environment-Aware Handlers**:
  - In `EnvProduction`: Emits machine-readable JSON records (`slog.NewJSONHandler`) at `Info` level.
  - In `EnvLocal` / `EnvTesting`: Emits human-readable text records (`slog.NewTextHandler`) at `Info` level.
  - When `debug: true`: Lowers log level to `Debug` and enables source file/line annotations (`AddSource: true`).
- **Multi-Runtime Source Tagging**: Decorates logs with source markers (`source="go"`, `source="js"`, `source="laravel"`) for unified aggregation in hybrid projects.
- **Diagnostic Error Integration**: Bridges `errs.AttrsOf` and `errs.HintOf` directly into structured log fields.

### Example Setup and Usage

```go
package main

import (
	"log/slog"
	"github.com/krewire/krewire/pkg/env"
	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/log"
)

func main() {
	// Initialize and set process-wide default logger
	logger := log.Install(env.EnvProduction, false)

	// Decorate with multi-runtime source tag
	logger = log.WithSource(logger, "go")

	logger.Info("service listening", "port", 8080)

	// Structured error logging with diagnostic attributes
	err := errs.WithAttrs(errs.FailureError("database connection timeout"),
		errs.Attr{Key: "db_host", Value: "db.internal"},
	)
	log.LogError(logger, "startup failed", err)
}
```
