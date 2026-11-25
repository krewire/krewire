# Krewire — Foundational Core Monorepo

> **Krewire is an end-to-end, Go-first digital SDLC ecosystem engineered to build, run, and scale secure, sustainable, and scalable software products from specification to production.**

Official open-source foundational monorepo for the Krewire ecosystem (`github.com/krewire/krewire`), providing modular domain packages, foundational kernel primitives, official application runtimes, and autonomous agentic engineering blueprints.

The monorepo operates as a **Pure Multi-Module Monorepo** comprising **35 independent domain Go modules** with no root `go.mod`, linked locally via `go.work`.

For the canonical brand definition, communication variants, and taxonomical guide, see **[Canonical Definition & Identity](docs/foundation/canonical-definition.md)**.

## Architecture & Layout

The monorepo contains foundational and reusable domain modules organized at the repository root:

```text
krewire/
├── go.work       # Workspace linking 37 internal modules and sibling ecosystem checkouts
├── docs/         # Ecosystem documentation (foundation, modules, tools, templates, guides)
├── ai/           # Native AI SDK: providers, streaming, tools, schema, object generation, ReAct agent
├── apicontract/  # Declarative API contracts, schema validation, and client generation
├── app/          # Dependency injection container & application assembly
├── assets/       # Asset management, pipeline, bundling, and hash busting
├── auth/         # Authentication & token verification primitives (Basic Auth, JWT)
├── book/         # Markdown content compiler (chapters, TOC, search, book sites)
├── cache/        # Multi-driver cache interfaces (memory, file)
├── cloud/        # Runner, service, worker, storage, infra, deploy targets
├── cmd/ai/       # Autonomous AI agent CLI entrypoint
├── collab/       # Team roles and per-environment collaboration policies
├── commerce/     # Payment gateway integration with constant-time webhook verification
├── config/       # Declarative configuration loader (krewire.yaml)
├── conformance/  # Conformance test suites, layering guards, and ecosystem benchmarks
├── env/          # Target environment resolution (local, production, testing)
├── errs/         # POSIX-compatible exit codes, diagnostic attributes, and stack traces
├── file/         # Atomic file operations and safe replacement
├── fs/           # Testable filesystem abstraction boundary
├── functional/   # Generic collection and functional utilities (Map, Filter, Reduce)
├── i18n/         # Internationalization, message bundles, pluralization, middleware
├── lifecycle/    # Workload execution dispatch, registry resolution, and supervisor
├── log/          # Structured slog logging tailored to environments and source tagging
├── mail/         # Multi-driver email delivery (SMTP, Resend, SendGrid, Postmark)
├── markdown/     # Markdown parsing & rendering with link prefixing
├── model/        # Declarative domain models, workload matrix, scopes, tiers, opt-in rules
├── observe/      # OpenTelemetry and structured metrics collection
├── orm/          # Pure-Go multi-driver ORM, SQL builder, bounded pool defaults, migrations engine
├── resilience/   # Bulkhead bounds, RateLimiter token bucket, retry with jitter, circuit breaker
├── router/       # High-performance HTTP trie router and middleware chain
├── runtime/      # Client WebAssembly VDOM engine, component mounting, island hydration
├── sec/          # SafePath traversal guard, PBKDF2 hashing, SafeHTTPClient (SSRF), CSRF/CORS
├── storage/      # Persistent object storage abstractions (disk, memory, cloud)
├── testing/      # Test harnesses and browser automation utilities
├── toolchain/    # Toolchain contracts, project context linter, adapter specifications
├── validation/   # Schema and struct validation
├── version/      # Semantic versioning, compatibility checks, and build watermark signature
├── web/          # HTTP router, middleware, SSR, SSG, gateway, HMR
├── workgraph/    # Workspace dependency graph and affected-task engine
└── templates/    # Starter templates and scaffolding blueprints
    ├── boost/    # AI-agent workflow template and constitution
    ├── init/     # In-place project initialization templates
    └── new/      # Fresh project scaffolding blueprints
```

> **Third-Party Isolation**: In accordance with the Port & Adapter architectural invariant, all external third-party database drivers (`modernc.org/sqlite`, `github.com/jackc/pgx/v5`, `github.com/go-sql-driver/mysql`, `redis`) and frontend compilers (`svelte`, `tailwind`, `vue`) reside exclusively in [`krewire/hub`](https://github.com/krewire/hub). The core `krewire` monorepo remains strictly 100% Go Standard Library.

## Documentation Index

- [Documentation Hub](docs/index.md) — Architectural overview, module index, and guides.
- [Ecosystem Architecture](docs/foundation/architecture.md) — 6-Layer pyramid architecture, downward dependencies, and layering tests.
- [Canonical Definition](docs/foundation/canonical-definition.md) — Brand guidelines, naming standards, and taxonomy.
- [Workload Guides](docs/guides/workload-architecture-guide.md) — Architectural guide for all 8 software workload kinds.
- [Getting Started Guide](docs/getting-started.md) — Monorepo setup, Go workspace synchronization, and local testing.
- [Contributing Guidelines](CONTRIBUTING.md) — Code quality gates, pure-Go invariant, and commit standards.

## Role in the 11-Repository Ecosystem & Battery Architecture

All workloads, libraries, and platforms advance along the **Ecosystem Architectural Battery**, fully driven by the `krew` unified toolchain CLI and declarative `krewire.yaml` configuration:

```text
Go Stdlib ──► Krewire Libraries ──► Krewire UI | Krewire Hub | Krewire Agent ──► Krewire Cloud ──► Krewire Dash
                                                                                                 (Unified Workspace)
```

1. **krewire** (this repository): Foundational pure-Go monorepo (35 domain modules without root `go.mod`).
2. **hub**: Port & Adapter repository for third-party integrations, external database drivers, and frontend compilers.
3. **krew**: Unified developer toolchain & CLI driver (`cli`, `core`, `conformance`, `version`).
4. **tui**: Lightweight, pure-Go Terminal User Interface (TUI) framework and terminal design primitives (`core`, `term`).
5. **ui**: clean, modern `.kiw` design system (10 modular packages: `components`, `core`, `styles`, etc.).
6. **cloud**: Private enterprise hybrid PaaS engine and control plane (13 modules).
7. **auth**: Private enterprise identity, SSO, MFA, and RBAC platform (12 modules).
8. **dash**: AI-native virtual workspace, execution sandboxes, and collaboration console (13 modules).
9. **internal**: Private engineering governance hub: ADRs, roadmaps, technical specifications.
10. **krewire.com**: Official production website and documentation portal (`mcp`, `examples/todo-app`).
11. **krewire.github.io**: Community launchpad, release hub, and open-source sponsorship portal.

## The Three Pillars (Secure, Sustainable, and Scalable)

Every module, tool, and service in Krewire is engineered to pass the **Three Pillars**:
- **Secure**: Standard library first to eliminate supply-chain vulnerabilities, `SafePath` directory traversal guards, PBKDF2 password hashing, `SafeHTTPClient` SSRF defense, constant-time secret checks, secure-by-default HTTP primitives (OWASP/CWE-aligned), context-aware template auto-escaping, CSRF/CORS protection, and secrets referenced via `${env:NAME}` without hardcoding.
- **Sustainable**: Bounded database connection pools and bulkhead concurrency limits to maintain a 10–30 MB lean RAM footprint, modular packages where you only pay for what you import, single static binaries with embedded assets, boring proven technology, a permanent MIT-licensed open-core foundation, and near-zero operational costs allowing production workloads to run on a $5 VPS.
- **Scalable**: Progressive pipeline from static sites to distributed cloud mesh without rewrites, reversible provider swapping (SQLite to Postgres, Memory to Redis), 6-layer Pyramid architecture with Bridge/Gateway decoupling, and Zero Peer Coupling between lateral engines.

## Prerequisites

- [Go](https://go.dev/) 1.27.1 or higher
- [Git](https://git-scm.com/)

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/krewire/krewire.git
   cd krewire
   ```

2. Run quality gates:
   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   ```

## License

[MIT](LICENSE) © 2026 Krewire Contributors
