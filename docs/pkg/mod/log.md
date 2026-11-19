# Structured Logging (`pkg/log`)

Import: `github.com/krewire/krewire/pkg/log`

## Purpose

The `pkg/log` package configures structured `slog` logging tailored to target execution environments (`env.Env`) with source tagging for multi-runtime architectures and direct integration with `pkg/errs` diagnostics.

---

## Environment Formatting Behavior

| Target Environment | Handler Format | Default Level | Features |
| :--- | :--- | :--- | :--- |
| **`EnvProduction`** | JSON (`slog.NewJSONHandler`) | `Info` | Optimized for log ingestion engines (FluentBit, Datadog, CloudWatch) |
| **`EnvLocal`** | Human-readable text (`slog.NewTextHandler`) | `Info` | Clean console output for interactive terminals |
| **`EnvTesting`** | Human-readable text (`slog.NewTextHandler`) | `Info` | Clear diagnostic output during `go test` runs |
| **`debug: true`** | Configured format | `Debug` | Adds caller source code annotations (`AddSource: true`) |

---

## Main API

- **`Setup(e env.Env, debug bool) *slog.Logger`**: Creates and configures a new `slog.Logger` instance writing to `os.Stderr`.
- **`Install(e env.Env, debug bool) *slog.Logger`**: Configures the logger and sets it as the process-wide default via `slog.SetDefault()`.
- **`WithSource(l *slog.Logger, source string) *slog.Logger`**: Decorates the logger with a structured `source` attribute (e.g. `"go"`, `"js"`, `"laravel"`).
- **`SourceTag(source string) string`**: Returns a bracketed console tag like `"[go]"` for terminal output.
- **`ErrAttrs(err error) []slog.Attr`**: Extracts structured diagnostic attributes from an error created via `errs.WithAttrs`.
- **`LogError(l *slog.Logger, msg string, err error)`**: Logs an error message at `Error` level, automatically extracting attributes and user hints.

---

## Example Usage

```go
package main

import (
	"log/slog"
	"github.com/krewire/krewire/pkg/env"
	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/log"
)

func main() {
	// Configure production logger
	logger := log.Install(env.EnvProduction, false)

	// Decorate with multi-runtime source tag
	logger = log.WithSource(logger, "go")

	logger.Info("server starting", "port", 8080)

	// Create diagnostic error
	err := errs.WithAttrs(
		errs.FailureError("failed to connect to upstream database"),
		errs.Attr{Key: "host", Value: "db.krewire.internal"},
		errs.Attr{Key: "retries", Value: 3},
	)
	err = errs.WithHint(err, "Verify database credentials in krewire.yaml")

	// Logs message, error string, all attributes, and the hint
	log.LogError(logger, "database initialization failed", err)
}
```

---

## Design Boundaries

- **Standard Library `log/slog`**: Wraps Go's standard structured logging library without external third-party logging engines.
- **Fail-Safe**: All functions are nil-safe and fallback to `slog.Default()` if given a nil logger.
