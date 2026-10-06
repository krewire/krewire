# Architecture — Krewire Libraries

## Module Structure

```
libs/
├── core/                 # Business rules — Kind/Workload registry, SpecID/RequirementID, Project invariants, DomainEvent + ExitCode/Error (KWL-K1N2Q, re-exports vein)
├── kern/                 # Kernel executor — Kernel, Module, Registry, Executor, Supervisor (KWL-KERN-X8P3L)
├── vein/                 # Krewire Vein — observability: logging (Setup/Install), diagnostics (Attr/Hint/FormatTree), stack traces (WithStack/StackOf), error handling (ExitCode/Error)
├── sec/                  # Krewire Security — auth integration, headers, CORS, CSRF, health, policies, SSRF URL validation, PII masking, and slog masking
├── term/                 # Terminal I/O, colors, formatting
├── config/               # Typed `krewire.yaml` loading for all 8 kinds (delegates business validation to core)
├── validation/           # Struct validation and extensible rules (`validate:"required"` etc.)
├── fs/                   # Testable filesystem boundary over io/fs and os
├── file/                 # File-oriented operations and atomic replacement
├── storage/              # Backend-independent object storage with local backend
├── functional/            # Generic closure-based utilities
├── resilience/            # Retry policies and circuit breakers
└── docs/
```

**Design decisions:**

- **Declarative + imperative control plane.** `core` (what is valid) + `kern` (how it runs) are the ecosystem's center; every repo imports `core` for types/rules, `framework`/`krewire` compose via `kern`. `core` is stdlib-only; `kern` is stdlib + `core` (no `framework` dependency, to avoid cycles).
- **Modular at every Scope (SRP/SoC).** Even `Unit` is a module — one concern per file/package, no God Module. Industry: SRP (SOLID), Separation of Concerns (Parnas), High Cohesion/Low Coupling, Unix "Do one thing well". Applies from `libs/core.Scope` → `libs/core.Kind` → `libs/core.ScopeUnit`.
- **Scope hierarchy as code.** `core.Scope` (`KWL-ARCH-J2K9Q`) codifies `Workspace → Module → Domain → Service → Unit` (Krewire Workspace = Go `go.work` at hub root; `Module ⊃ Service ⊃ Unit`, Unit = Go package / `pkg.Func`) with `ParseScope`/`Less()`; all specs/tests declare scope, enabling `kiw test --spec` filtering (KWL-TEST-P8M4L).
- **Monorepo, independent versioning.** Each package is importable alone; consumers pull only what they need.
- **Single config authority.** `config` + `validation` enforce the `krewire.yaml`-only rule for every workload; used by `framework`, `krewire`, and `mdbind`; business validation delegates to `core`.
- **No re-implementation.** Where stdlib covers it (`flag`, `log/slog`, `os`), libs does not duplicate.
- **Cross-repo replace for local dev.** `framework/go.mod` → `replace github.com/krewire/libs => ../libs` during development, removed before tag.


## Conventions

- Documentation in English, Markdown, spec-driven (`docs/specs/`); requirements declare `Scope: Workspace/Module/Domain/Service/Unit` (`KWL-ARCH-J2K9Q`), tests declare `// Tests for <SpecID>` (`KWL-TEST-P8M4L`).
- Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` in each Go repo; per-kind `kiw build` / `kiw build --plan` spot-checks.
- Cross-repo testing via `go.work` workspace (`./framework`, `./libs`, etc.) at hub root; `go work sync` updates `go.work.sum`.
