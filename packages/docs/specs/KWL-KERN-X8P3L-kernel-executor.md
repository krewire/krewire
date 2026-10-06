# Specification — Kernel Executor & Supervisor

| Field  | Value                          |
|--------|--------------------------------|
| SpecID | KWL-KERN-X8P3L                  |
| Title  | Kernel Executor & Supervisor   |
| Status | Draft                          |
| Date   | 2026-08-21                     |
| Author | Krewire Contributors            |
| Domain | Libraries — Kernel             |

## 1. Context

The ecosystem has a declarative center (`libs/core` — *what* is valid) but no imperative center (*how* it runs). Boot logic is duplicated in `framework/app` (`NewApp(...).Build()`) and `krewire/internal/commands` (`run`/`dev`/`deploy` dispatch). `worker`, `service`, and `infra` each expose their own `Run`/`Plan`/`Apply` loops without a shared lifecycle. The unified vision needs a single executor that boots from `krewire.yaml`, registers modules, supervises workers/services, and handles signals — the Kernel.

## 2. Problem Statement

- No shared boot sequence; `krewire` and `framework` duplicate config loading and validation.
- No supervisor for long-running processes (`krewire dev` hot-reload, `krewire worker`, `service` registry); each invents signal handling and graceful shutdown.
- No plugin system for opt-in batteries (`worker`/`service`/`infra`) to register with the monolith without importing `framework` in `libs`.

## 3. Goals

- G1 — `libs/kern` provides generic `Kernel`, `Module`, `Registry`, `Supervisor`, and `Executor` — stdlib-only plus `libs/core`.
- G2 — Boot from `core.Project`: load `krewire.yaml` via `libs/config` (caller-provided, not hard-coded), validate via `libs/validation` + `core`, register modules in dependency order.
- G3 — Supervision: manage lifecycle of registered modules (start, health, stop) with signal-aware graceful shutdown and hot-reload hook.
- G4 — Zero coupling: `libs/kern` never imports `framework`; `framework` provides modules that satisfy `kern.Module`.

## 4. Non-Goals

- NG1 — Not re-implementing business rules; types and validation come from `libs/core`.
- NG2 — Not a full OTP supervisor; v1 is a simple Go supervisor with `context` cancellation, not process isolation.
- NG3 — No network or provider SDKs; `framework/infra` and `framework/service` supply those behind `Module` implementations.

## 5. Requirements

### 5.1 Module & Registry

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-KERN-001 | `type Module interface { Name() string; Init(*Kernel) error }` — modules are parameterised by the kernel they initialize against. | Must |
| KWL-KERN-002 | `type Registry struct` with `Register(Module) error`, `Resolve(name string) (Module, bool)`, `Ordered() []Module` returning topologically sorted modules by declared `DependsOn() []string` (if module implements it) or registration order. | Must |
| KWL-KERN-003 | Duplicate `Name()` registration returns `*core.Error` with `ExitCodeUsage`. | Must |

### 5.2 Kernel & Boot

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-KERN-010 | `type Kernel struct { Project core.Project; Registry *Registry; Supervisor *Supervisor; Executor Executor }` + `func New(core.Project) (*Kernel, error)` validating project via `core.Project.Validate()`. | Must |
| KWL-KERN-011 | `func (k *Kernel) Use(modules ...Module) *Kernel` registers modules and returns receiver for chaining. | Must |
| KWL-KERN-012 | `func (k *Kernel) Boot(ctx context.Context) error` iterates `Registry.Ordered()`, calls `Module.Init`, and aggregates errors as `*core.Error` with `ExitCodeFailure`. | Must |
| KWL-KERN-013 | `Loader` abstraction for config: `type Loader interface { Load(path string) (core.Project, error) }` — callers inject `libs/config` loader; `kern` does not hard-code YAML parsing. | Should |

### 5.3 Executor & Supervisor

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-KERN-020 | `type Executor interface { Execute(ctx context.Context, workload core.Workload) core.ExitCode }` — `Kernel.Execute` dispatches to the registered module that handles the workload's `Kind`; unknown kind returns `ExitCodeUsage`. | Must |
| KWL-KERN-021 | `type Supervisor struct` with `Start(ctx context.Context) error`, `Stop(ctx context.Context) error`, `Health() map[string]error`; manages long-running modules (those implementing `Startable interface { Start(context.Context) error; Stop(context.Context) error }`). | Must |
| KWL-KERN-022 | Signal-aware shutdown: `Supervisor` listens for `ctx.Done()` and calls `Stop` with timeout (default 10s, configurable via `WithShutdownTimeout`). | Should |
| KWL-KERN-023 | Hot-reload hook: `Supervisor.OnReload(func())` — called by `krewire dev` when sources change; no file watching inside `kern` itself. | Should |

### 5.4 Module Contracts for Framework Consumers

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-KERN-030 | `framework/app` provides `AppModule` satisfying `kern.Module`; `framework/worker` provides `WorkerModule`; `framework/service` provides `ServiceModule`; `framework/infra` provides `InfraModule` — each `Init` registers its workload handler with the kernel's `Executor`. | Should |
| KWL-KERN-031 | Example usage is documented in `go doc` for `kern.Kernel` (see §7). | Must |

## 6. Non-Functional Requirements

- NFR1 — Stdlib plus `libs/core` only; optional `libs/config`/`libs/validation` via `Loader` injection, not hard dependency.
- NFR2 — `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` with in-memory fakes; no external services.
- NFR3 — Backward compatible with existing `framework/app` boot — `kern` is additive, not a breaking replacement.

## 7. Success Criteria

- S1 — Example in `go doc` passes `gofmt` and compiles:
  ```go
  project := core.Project{Name: "demo", Kind: core.KindService}
  k, _ := kern.New(project)
  k.Use(service.Module{}, worker.Module{})
  k.Boot(ctx); defer k.Shutdown(ctx)
  k.Execute(ctx, core.Workloads[core.KindService])
  ```
- S2 — `libs/kern` tests cover `Registry` duplicate detection, `Kernel.Boot` error aggregation, `Supervisor` start/stop with `context` cancellation, and `Executor` unknown-kind handling.
- S3 — `krewire` can be refactored to delegate `run`/`dev`/`worker` dispatch to `kern.Kernel` without behavior change (follow-up, not in this spec).

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| [KWL-K1N2Q](./KWL-CORE-K1N2Q-core-business-rules.md) | Core Business Rules (declarative counterpart) |
| [KWL-W0J2X](./KWL-CORE-W0J2X-errors-exit-codes.md) | Errors & Exit Codes (extends) |
| [KWF-ARCH-M8K2Q](https://github.com/krewire/framework/blob/main/docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified Vision |
| [KWF-5ZHQV](https://github.com/krewire/framework/blob/main/docs/specs/KWF-ARCH-5ZHQV-modular-monolith-architecture.md) | Modular Monolith (extraction path that kern supervises) |

## 9. References

- Erlang/OTP supervisor: https://www.erlang.org/doc/design_principles/sup_princ.html
- `internal/docs/project-vision.md` — workload matrix
