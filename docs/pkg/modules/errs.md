# Error & Diagnostic Primitives (`packages/errs`)

The `packages/errs` package provides POSIX-compatible exit codes, structured diagnostic attributes, and stack traces with standard-library-only dependencies.

## Key Types & Functions

```go
import "github.com/krewire/krewire/pkg/errs"

// Exit codes
const (
    ExitCodeSuccess = 0
    ExitCodeFailure = 1
    ExitCodeUsage   = 2
)

// Constructors
err := errs.UsageError("missing required parameter")
err = errs.FailureError("failed to connect to upstream")

// Diagnostics
err = errs.WithAttrs(err, errs.Attr{Key: "endpoint", Value: "/api/v1"})
err = errs.WithHint(err, "run `kiw init` to initialize the project")
err = errs.WithStack(err)

// Formatting
fmt.Println(errs.FormatTree(err))
```
