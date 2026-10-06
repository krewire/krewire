# `kern`

Import: `github.com/krewire/libs/kern`

## Purpose

Imperative kernel control plane for booting, registering, executing, supervising, and shutting down workload modules.

## Main API

- `Kernel`, constructed with `kern.New(core.Project)`
- `Module`, the lifecycle contract implemented by consumers
- `Registry`, constructed with `kern.NewRegistry()`
- `Executor`, for workload execution
- `Supervisor`, constructed with `kern.NewSupervisor()`
- `DefaultShutdownTimeout` (`10s`)

## Example

```go
project := core.Project{/* loaded and validated project */}
kernel, err := kern.New(project)
if err != nil { return err }
_ = kernel
```

## Design boundary

`kern` is stdlib plus `core`; it must not import `framework` or `krewire`. Those repositories provide concrete `Module` implementations. Configuration loading and business validation are injected or delegated to `config`, `validation`, and `core`.
