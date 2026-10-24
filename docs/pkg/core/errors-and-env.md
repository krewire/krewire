# Package: `pkg/errs` & `pkg/env` (Errors, Exit Codes & Logging)

This library pair manages structured error categorization, diagnostic stack traces, process exit codes, and environment-aware logging (`KWL-W0J2X`, `KWL-P8W2N`, `KWL-K4T7W`).

---

## 1. Exit Codes & Error Types (`pkg/errs`)

Krewire maps errors deterministically to standard POSIX process exit codes:

| Error Type | Exit Code | Purpose |
|---|---|---|
| `OK` | `0` | Success |
| `GeneralError` | `1` | Unhandled runtime failure |
| `UsageError` | `2` | Invalid command line arguments, flags, or project names |
| `ConfigError` | `78` | Malformed `krewire.yaml` or missing configuration |

### Creating Errors
```go
import "github.com/krewire/krewire/pkg/errs"

// Usage error (exit code 2)
return errs.UsageError("missing required argument <target>")

// Config error (exit code 78)
return errs.ConfigError("krewire.yaml: 'port' must be positive integer")
```

---

## 2. Environment & Logging (`pkg/env`, `pkg/log`)

- **Environment Detection**: Detects `development`, `staging`, `production`, and `test` environments.
- **Structured Slog Integration**: Employs Go's `log/slog` with automatic PII masking (passwords, JWT tokens, credit card patterns) and context tracing.
- **Debug Mode**: Controlled via `DEBUG=1` or `DEBUG=true` to enable verbose diagnostic trees.
