# Philosophy — Krewire Ecosystem

Guiding principles, architectural rationales, design ethics, and contribution discipline underpinning the Krewire platform.

---

## 1. Core Philosophy

> **Small, composable, stdlib-aligned — with one CLI for every workload.**

Modern software development often suffers from tool fragmentation: separate runtimes for backend, frontend, documentation, CLI tools, and background workers; sprawling `node_modules` dependencies; brittle glue scripts; and disconnected configuration formats.

Krewire rejects this fragmentation. We believe that a cohesive, type-safe, compiled language with first-class concurrency and single-binary deployment can orchestrate every software workload — from static documentation books to distributed cloud workers — under a unified, elegant development model.

---

## 2. Core Principles

### 1. Modular at Every Scope (SRP, SoC & High Cohesion)
We adhere rigorously to the **Single Responsibility Principle** (Robert C. Martin), **Separation of Concerns** (Dijkstra/Parnas), and the **Unix philosophy ("Do one thing well")**. 

In Krewire, modularity applies at every level of the scope hierarchy:
```text
Workspace  ───►  Module  ───►  Domain  ───►  Service  ───►  Unit
```
Even an individual `Unit` (Go package, function, or CLI command) has one clearly defined responsibility and one reason to change. The anti-pattern of "God Modules" or "kitchen-sink utility packages" is rejected across the codebase.

### 2. Why Go as the Unified Language
The choice of Go as the primary language across the entire Krewire ecosystem is an architectural decision, not a matter of syntax preference:
- **Single-Binary Ownership:** `go:embed` combined with `go build` produces self-contained, dependency-free executables.
- **Zero Hosting Friction:** Near-zero runtime overhead, minimal memory footprints, and instant cold starts compared to interpreted VM stacks.
- **Unified Toolchain Quality Gates:** Standard tooling (`gofmt`, `go vet`, `go test`) provides deterministic verification with zero third-party setup.
- **Full-Stack Type Sharing:** The same Go types and components (`pkg/ui`) drive server-side rendering (SSR), static generation (SSG), and interactive client-side WebAssembly islands (`GOOS=js`) without context switching between TypeScript and Go.

### 3. Standard-Library-First (Stdlib-First)
Where Go's standard library provides a robust capability (`flag`, `log/slog`, `os`, `net/http`, `io/fs`, `html/template`), Krewire builds upon it rather than wrapping or re-implementing it. 
- `pkg/errs` is 100% stdlib-only.
- `pkg/model` uses stdlib plus `errs` (no external framework dependency to prevent cyclic references).
- Custom abstractions exist only where domain logic, type safety, or security defaults require formalization.

### 4. Declarative Definition vs. Imperative Execution
Control flow in Krewire separates **what is valid** from **how it runs**:
- **Declarative Core (`pkg/model`):** Pure data models, invariant validation, scope hierarchies, and business rules. Free of side effects and external dependencies.
- **Imperative Runtime (`pkg/lifecycle`, `pkg/app`):** Execution engines, dependency injection, supervised processes, and event dispatchers that operate upon declarative models.

### 5. Single Source of Truth (SSOT)
Duplication breeds drift. Across the entire ecosystem:
- **Configuration SSOT:** [`krewire.yaml`](../../pkg/model/project.go) is the single configuration authority for all eight workload kinds. Ad-hoc or tool-specific configuration formats are prohibited.
- **Workload Matrix SSOT:** Canonical kinds and matrix entries are defined once in `pkg/model/workload.go`.
- **Specification SSOT:** Requirements and progress tracking live in GitHub Issues tagged `spec`, eliminating out-of-sync markdown checkboxes.

### 6. Zero-Cost Abstractions & Opt-In Batteries
Importing a base model or kernel must never inflict transitive bloat.
- Core types are zero-value usable without complex factory ceremonies.
- **Opt-In Rules (`OptInRule`):** Batteries (such as cloud storage or worker queues) belong to their owning workload kind and are blocked from unintended kinds unless explicitly opted in. A lightweight CLI tool or static site remains ultra-fast and lightweight.

### 7. Progressive Support & Pipeline Scaling
Projects evolve. A Krewire project starts from a clean pre-init bootstrap layout and progressively equips new workload capabilities (`app`, `cli`, `site`, `book`, `worker`, `service`, `gateway`, `infra`) via `kiw init`. At each stage, the project's configuration and toolchain adapt seamlessly without requiring a rewrite or migration to different frameworks.

### 8. Local-First & Dogfooding Discipline
Developers must be able to develop, preview, test, and audit their entire stack locally, offline, and rapidly:
- The developer CLI (`kiw`) dogfoods the monorepo's own terminal UI toolkit (`pkg/tui`) and error handling (`pkg/errs`).
- `kiw dev` and `kiw serve` deliver instantaneous feedback loops via local servers and WebSocket-driven Hot Module Replacement (HMR).
- Releases are verified locally against the built binary (`bin/kiw`) before being published.

### 9. AI-Native Engineering (Boost Constitution)
AI coding agents are treated as first-class team members:
- The Krewire Boost template (`pkg/boost`, `templates/boost`) codifies an autonomous, multi-agent development workflow directly into the repository structure (`.agents/`).
- Agents operate under an explicit constitution (`AGENTS.md`) with role specializations (planning, designing, building, decomposing, testing, auditing).
- Agents respect project boundaries, test-driven requirements, and automated quality gates without human babysitting.

---

## 3. Contribution & Engineering Discipline

To maintain high code quality and architectural integrity:

1. **Strict Quality Gates:**
   Never commit code or conclude a task without passing:
   - `gofmt -l .` — Zero unformatted files.
   - `go vet ./...` — Zero static analysis warnings.
   - `go test ./...` — Zero failing tests.
   - `go build -o bin/kiw ./tools/kiw/cmd/kiw` — CLI builds cleanly.

2. **Commit Discipline:**
   - Use Conventional Commits: `feat|fix|docs|test|refactor|perf|security|chore(scope): description`.
   - Maintain atomic commits ($\le$ ~300 LOC per commit).
   - Reference corresponding GitHub Issues and requirement IDs in commit messages.

3. **Documentation Integrity:**
   - Store all repository documentation, commit messages, and specifications in **English**.
   - Keep documentation synchronized with code changes; eliminate doc-code drift proactively.
