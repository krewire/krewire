# Philosophy — Krewire Libraries

## Philosophy

**Small, composable, stdlib-aligned — with a central domain and kernel.** Libraries are narrow, focused, and versioned independently. `core` holds the business rules (workload matrix, `SpecID`, `Project` invariants) so every repo shares one truth; `kern` holds the generic executor (Kernel/Module/Supervisor) so `framework` and `krewire` share one boot lifecycle. `core` errors, `term` output, and `config` loading stay thin orchestration layers on top.

**Principles:**

- **Do one thing well.** Each package maps to one concern; no kitchen-sink utils. `core` = declarative (what is valid), `kern` = imperative (how it runs).
- **Modular at every Scope — SRP/SoC/High Cohesion (Industry Standard).** The **Single Responsibility Principle** (SOLID, Robert C. Martin), **Separation of Concerns** (Dijkstra/Parnas 1972), **High Cohesion & Low Coupling** (Constantine & Yourdon), **Unix "Do one thing well"** are the names for not stacking many unrelated functions in one module. One `Scope` = one concern, one reason to change; God Module is the anti-pattern. In libs this is `core.Scope` → `core.Kind` → `core.SpecID` (each small, cohesive); even `Unit` (`ScopeUnit`) is a module.
- **Why Go as the one language (architectural).** Only Go gives single-binary ownership (`embed` + `go build`), near-zero hosting, `gofmt`/`go vet`/`go test` as `kiw` quality gates, `net/http`/`html/template` without `npm`, and `GOOS=js` WASM islands from the same `ui` types — so `site`→`mesh` is one `Scope`-typed `go build`, not a context switch. See ADR-F4F0E.
- **Stdlib-first.** Prefer `flag`/`slog`/`os` over custom wrappers. `core` is stdlib-only; `kern` is stdlib + `core`.
- **Typed config for all kinds.** `config` decodes `krewire.yaml` into structs validated by `validate` + `core` — the single source for `app` through `infra`.
- **Spec traceability.** `KWL-*` specs (`KWL-K1N2Q`, `KWL-KERN-X8P3L`, `KWL-ARCH-J2K9Q`, `KWL-TEST-P8M4L`) declare `Scope` (`Workspace→Unit`) and trace to tests via `// Tests for <SpecID>`.
- **Progressive support.** `core.Scope` levels (`Workspace→Unit`) and `core.Kind` matrix exist to make the progressive pipeline (`KWF-ARCH-P7L2Q`) typable: batteries activate only at their stage, never by transitive import.
- **Zero-cost when unused.** `core` types are zero-value usable; `kern` is opt-in — importing only `core` adds no supervisor overhead.


## Contribution

- Read `project-vision.md` and `docs/specs/index.md` before changing behavior.
- Add/update tests matching project patterns; keep suite green.
- Update `README.md` / `docs/` and specs when public behavior changes; follow ecosystem spec conventions.
