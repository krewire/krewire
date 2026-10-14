# Architecture — Krewire Ecosystem

System architecture, layering rules, module hierarchy, and design decisions across the Krewire monorepo and its constituent subsystems.

---

## 1. Ecosystem Overview

Krewire is an integrated, full-stack application and developer ecosystem built in Go (1.27.1+). It unifies eight distinct software workload kinds under a single declarative control plane, standard-library-aligned domain packages, a consolidated developer CLI, and an AI-native agent constitution.

The ecosystem spans five repositories:
1. **`krewire`** (Core Monorepo): Domain packages, kernel executor, developer CLI (`kiw`), first-party apps, and templates.
2. **`mdbind`**: Standalone Markdown book & documentation engine powering the `book` workload.
3. **`internal`**: Private documentation, ADRs, roadmaps, and project vision.
4. **`krewire.com`**: Public marketing and documentation website built with the `site` workload.
5. **`krewire.github.io`**: Community portal and distribution hub.

---

## 2. Monorepo Structure

```text
krewire/
├── docs/                 # Ecosystem documentation (Foundation, Packages, Boost, Devtool)
│   ├── foundation/       # Unified Architecture & Philosophy specifications
│   ├── packages/         # Domain packages and module specifications
│   └── tools/kiw/        # Developer CLI documentation
├── packages/             # Domain libraries and control plane
│   ├── kern/             # Kernel, lifecycle, errors, environment, model (workload & kinds)
│   ├── web/              # HTTP router, middleware, SSR, SSG, gateway, HMR
│   ├── app/              # Dependency injection container & application assembly
│   ├── ui/               # UI components, layout, styling tokens
│   ├── tui/              # Terminal UI toolkit powering the kiw CLI
│   ├── runtime/          # WebAssembly client runtime (VDOM, mounting, widgets)
│   ├── sec/              # Secure defaults (headers, CSRF, CORS, SSRF, PII redaction)
│   ├── auth/             # Authentication & token verification primitives
│   ├── cloud/            # Runner, service, worker, storage, infra deployment targets
│   ├── config/           # Declarative configuration loader (krewire.yaml)
│   ├── validation/       # Schema and struct validation
│   ├── storage/          # Persistent storage abstractions
│   ├── markdown/         # Markdown parsing & rendering
│   ├── testing/          # Test harnesses and browser test utilities
│   ├── hub/              # Plugin registry and package resolver
│   └── boost/            # Boost coordination package & embedded agentic template
├── apps/                 # Deployable services and applications
├── tools/                # Developer tooling
│   └── kiw/              # Unified developer CLI for scaffolding, build, and deploy
└── templates/            # Starter templates and scaffolding blueprints
    ├── boost/            # AI-agent workflow template and constitution (.agents/)
    ├── init/             # In-place project initialization templates (app, book, cli, static)
    └── new/              # Fresh project scaffolding blueprints (kernel)
```

---

## 3. Layering Invariants & Dependency Rules

The monorepo enforces **strict downward-only dependencies** (`KWL-LAYER-001`, verified in [`layering_test.go`](../../packages/kern/layering_test.go)):

```text
templates/boost (Guild)
       │
       ▼
tools/kiw (CLI)
       │
       ▼
packages/* (Domain Packages)
       │
       ▼
packages/kern (Kernel Executor & Model)
       │
       ▼
Go Standard Library
```

### Layering Rules:
1. **Kern is the Floor (`KWL-LAYER-002`):** `packages/kern` sits at layer 0. It requires no other ecosystem module. `packages/kern/model` imports exclusively from standard library packages and its internal `errs` package.
2. **No Upward or Lateral Leaks:** No package or library may ever import `tools/kiw`. Domain packages (`packages/*`) may only import downwards.
3. **No Ecosystem Names in Kernel (`KERN-LAYER-001`):** The kernel must never hardcode ecosystem module names or paths in constants, lookup tables, or strings. The module roster is dynamically derived from `go.work` and module metadata to ensure true zero-friction scalability.
4. **CLI Reuses Packages:** `tools/kiw` imports domain packages directly (e.g. `packages/hub/resolver`, `packages/tui`). Logic is never duplicated between CLI and domain packages.

---

## 4. Domain Model & Workload Spectrum

### The Eight Workload Kinds (`model.Kind`)
Every project in the ecosystem belongs to one of eight canonical kinds:
- **`app`**: Full-stack dynamic web applications (HTTP router, DI container, SSR, client WASM).
- **`cli`**: Standalone terminal commands and developer utilities.
- **`site`**: Content and marketing sites with static site generation (SSG) and HMR.
- **`book`**: Structured multi-chapter technical documentation books (integrated with `mdbind`).
- **`worker`**: Background processing queues, event consumers, and scheduled jobs.
- **`service`**: Headless backend microservices and RPC/REST APIs.
- **`infra`**: Declarative cloud infrastructure blueprints and state management.
- **`kernel`**: Minimal initial seed project (`go.mod` + `krewire.yaml` + `main.go`).

### Scope Hierarchy as Code (`model.Scope`)
System boundaries are codified through five explicit scope tiers:
```text
Workspace  ───►  Module  ───►  Domain  ───►  Service  ───►  Unit
```
- **Workspace:** Multi-module Go workspace rooted at `go.work`.
- **Module:** Deployable or distributable Go module (e.g. `github.com/krewire/krewire`).
- **Domain:** Functional subsystem or package group (e.g. `web`, `cloud`, `ui`).
- **Service:** Deployable service or autonomous process.
- **Unit:** Individual Go package, file, or component adhering to Single Responsibility Principle (SRP).

All specifications, tests, and requirement IDs tag their corresponding Scope level (`KWL-ARCH-J2K9Q`), allowing granular test execution (e.g. `kiw test --spec <SpecID>`).

### Single Configuration Authority (`krewire.yaml`)
- `krewire.yaml` is the sole declarative configuration authority for every workload. Obsolete formats (e.g., `ssg.yaml`) are strictly disallowed.
- Project name must follow kebab-case `^[a-z][a-z0-9-]*$`.
- Module path is validated against directory traversal risks (`CWE-22`).

### Opt-In Batteries (`OptInRule`)
Batteries are opt-in per workload kind. A project of kind `For` cannot import packages owned by kind `Owner` unless explicitly opted in. This guarantees zero bloat and zero-cost abstraction for lightweight kinds (such as `cli` or `kernel`).

---

## 5. Subsystem Architecture

### A. Kernel & Lifecycle Control Plane (`packages/kern`)
- **Kernel & Module Registry (`KWL-KERN-X8P3L`):** Provides structured lifecycle phases (`Init`, `Start`, `Stop`) with supervisor capabilities, graceful shutdown, and context propagation.
- **Error Handling & Exit Codes (`KWL-W0J2X`, `KWL-P8W2N`):** Structured error categorization (`UsageError`, `SystemError`, `ConfigError`), stack trace capture, and deterministic process exit codes.
- **Environment & Logging (`KWL-K4T7W`):** Standard-library-based structured logging (`log/slog`) with environment-aware formatting (JSON in production, styled text in local development) and automatic PII redaction.

### B. Domain Libraries (`packages/*`)
- **Web (`packages/web`):** High-throughput HTTP routing, pipeline middleware, SSR templating, SSG asset pipeline, WebSocket-driven HMR, and API gateway routing.
- **Runtime (`packages/runtime`):** WebAssembly client engine compiled with `GOOS=js`, providing a lightweight Virtual DOM (VDOM), declarative component mounting, and reactive widget islands.
- **Security (`packages/sec`):** Secure defaults out-of-the-box: OWASP-compliant security headers, CSRF token generation/validation, CORS configuration, URL SSRF defense, and structured log sanitization.
- **Storage & Cloud (`packages/storage`, `packages/cloud`):** Pluggable object storage abstractions (local filesystem and cloud object stores) with deployment runners, service discovery, and worker queues.
- **Config & Validation (`packages/config`, `packages/validation`):** Type-safe YAML unmarshaling with struct tag validation (`validate:"required"`).

### C. Developer Tooling — `kiw` (`tools/kiw`)
- **Unified Entry Point:** A single binary (`kiw`) replacing fragmented shell scripts or separate CLI binaries.
- **Pipeline Dispatch:** `kiw` inspects project markers and `krewire.yaml` to detect project kind (`shape`) and route commands:
  - `build` / `serve`: Dispatches SSG pipeline for `site`, `mdbind` pipeline for `book`, or Go binary builds for `app`/`cli`.
  - `run` / `dev`: Executes local apps, standalone scripts, or defined tasks with hot-reload.
  - `deploy`: Prepares and stages production artifacts into `dist/`.
  - `new` / `init`: Scaffolds projects from kernel blueprints and equips them with workload variants.
- **Dogfooding:** Built entirely on `packages/tui` for interactive terminal UI and `packages/kern` for status and error codes.

### D. AI-Native Boost Engine (`packages/boost` & `templates/boost`)
- **Embedded Agent Constitution:** Embedded via `//go:embed` into `packages/boost`, providing a pre-configured multi-agent orchestration team (`.agents/`).
- **20 Specialized Agents & Workflows:** Covers planning, architecture, implementation, refactoring, security auditing, and continuous learning.
- **Project Governance:** Standardized rules and templates installed via `kiw boost install` into any new or existing repository.

---

## 6. Specifications, Traceability & Quality Gates

### GitHub Issues as SSOT
Active specifications are maintained exclusively as GitHub Issues under `krewire/krewire` labeled `spec` with title format `[Spec: <SpecID>] <Title>`. Every issue contains verifiable requirement checklists (`- [ ] RND-IF-001 [Must] ...`).

### Code Traceability
- Test files cite `// Tests for <SpecID>`.
- Individual test cases follow `Test<SpecID>_<ReqID>_<Scenario>`.

### Required Quality Gates
Every contribution must cleanly pass the monorepo quality gates:
```bash
gofmt -l .        # Must output nothing
go vet ./...      # Must pass without lint warnings
go test ./...     # All unit and integration tests must pass
go build -o bin/kiw ./tools/kiw/cmd/kiw # CLI binary must build cleanly
```
