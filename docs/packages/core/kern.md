# Kernel & Control Plane — `kern` Package Reference

## Description
The `packages/kern` library provides the foundational control plane, lifecycle supervision, error hierarchy, environment models, structured logging, and ecosystem build signature for the entire Krewire ecosystem.

---

## Zero-Dependency Floor

`packages/kern` sits at the very bottom of the dependency graph (`KWL-LAYER-002`, `ADR-B7D3A`):
- It depends **only** on the Go standard library and its internal subpackages.
- It never imports any higher-level ecosystem package or CLI tool.
- Every package in Krewire can safely import `packages/kern`.

---

## Build Signature (`build with krewire`)

The ecosystem provides a verifiable build signature embedded into compiled binaries via `-ldflags`:

```go
package kern

// DefaultSignature is the canonical signature embedded into all Krewire binaries.
const DefaultSignature = "build with krewire"

// Signature holds the active binary signature, overrideable via -ldflags.
var Signature = DefaultSignature

// BuildSignature returns the active ecosystem binary signature.
func BuildSignature() string {
    return Signature
}
```

When building or running with `kiw`, the devtool automatically passes:
`-ldflags="-X 'github.com/krewire/krewire/packages/kern.Signature=build with krewire'"`

---

## Subpackages & Subsystems

### 1. `kern/lifecycle` (Module Lifecycle & Supervision)
Orchestrates module boot sequences across three distinct phases (`Init`, `Start`, `Stop`):

```go
type Module interface {
    Name() string
    Init(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type Supervisor struct { /* ... */ }
func NewSupervisor() *Supervisor
func (s *Supervisor) Register(m Module)
func (s *Supervisor) Run(ctx context.Context) error
```

### 2. `kern/log` (Structured Logging & Multi-Runtime Tagging)
Configures the process-wide structured logger (`log/slog`) with environment awareness and multi-runtime source tagging:

```go
// Setup logger according to environment (local = text, production = JSON)
logger := log.Setup(env.EnvProduction, false)

// Decorate logger with distinguishable source tags (KWN-NDV-014, KWN-LVR-014)
goLogger := log.WithSource(logger, "go")
goLogger.Info("database pool initialized")

// Human-readable console tag: "[go]"
tag := log.SourceTag("go")

// Structured error logging with diagnostic attributes and actionable hints
log.LogError(logger, "connection failed", err)
```

### 3. `kern/errs` (Diagnostic Errors & Exit Codes)
Provides typed error categorization, structured diagnostic attributes, and user-actionable hints:
- Standard exit codes: `ExitCodeSuccess` (0), `ExitCodeFailure` (1), `ExitCodeUsage` (2).
- `errs.WithHint(err, "run 'kiw init' first")`
- `errs.WithAttrs(err, errs.Attr{Key: "port", Value: 8080})`

### 4. `kern/env` (Runtime Environment Invariants)
Defines the canonical environment rungs (`EnvLocal`, `EnvTesting`, `EnvStaging`, `EnvProduction`) with strict validation rules.

### 5. `kern/model` (Workload Model)
Declares the eight first-class workload kinds (`app`, `cli`, `site`, `book`, `worker`, `service`, `infra`, `kernel`).

---

## Quick References

- [Architecture Blueprint](../../foundation/architecture.md) — Downward-only dependency hierarchy.
- [Multi-Runtime Coexistence Guide](../../guides/multi-runtime-coexistence.md) — Cross-runtime logging and process bridges.
- [`kiw` Commands Reference](../../tools/kiw/commands.md) — CLI flags and operations.
