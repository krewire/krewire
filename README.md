# Krewire — Core Monorepo

> **Krewire is an end-to-end, Go-first digital SDLC ecosystem engineered to build, run, and scale secure, sustainable, and scalable software products from specification to production.**

Official open-source core monorepo for the Krewire ecosystem (`github.com/krewire/krewire`), providing modular domain packages, foundational kernel primitives, official application runtimes, and agentic engineering blueprints.

For the canonical brand definition, communication variants, and taxonomical guide, see **[Canonical Definition & Identity](docs/foundation/canonical-definition.md)**.

## Architecture & Layout

The monorepo contains foundational and reusable domain packages:

```text
krewire/
├── docs/         # Ecosystem documentation (foundation, pkg, tools, templates, guides)
├── pkg/          # Reusable domain libraries and control plane (33 modular packages)
│   ├── model/    # Declarative domain models, workload matrix, scopes, tiers, opt-in rules
│   ├── lifecycle/# Workload execution dispatch, registry resolution, and supervisor
│   ├── errs/     # POSIX-compatible exit codes, diagnostic attributes, and stack traces
│   ├── env/      # Target environment resolution (local, production, testing)
│   ├── version/  # Semantic versioning, compatibility checks, and build watermark signature
│   ├── log/      # Structured slog logging tailored to environments and source tagging
│   ├── web/      # HTTP router, middleware, SSR, SSG, gateway, HMR
│   ├── app/      # Dependency injection container & application assembly
│   ├── runtime/  # Client WebAssembly VDOM engine, component mounting, island hydration (WASM, Svelte, Vue)
│   ├── ai/       # Native AI SDK: providers, streaming, tools, schema, object generation
│   ├── sec/      # SafePath traversal guard, PBKDF2 hashing, SafeHTTPClient (SSRF), headers, CORS, CSRF
│   ├── auth/     # Authentication & token verification primitives (Basic Auth, JWT)
│   ├── assets/   # Asset management, pipeline, bundling, and hash busting
│   ├── cloud/    # Runner, service, worker, storage, infra, deploy targets
│   ├── config/   # Declarative configuration loader (krewire.yaml)
│   ├── validation/ # Schema and struct validation
│   ├── storage/  # Persistent object storage abstractions (disk, memory, cloud)
│   ├── fs/       # Testable filesystem abstraction boundary
│   ├── file/     # Atomic file operations and safe replacement
│   ├── functional/ # Generic collection and functional utilities (Map, Filter, Reduce)
│   ├── hub/      # Plugin registry and package resolver (Tailwind, Svelte 5, Vue 3 modular plugins)
│   ├── i18n/     # Internationalization, message bundles, pluralization, middleware
│   ├── markdown/ # Markdown parsing & rendering with link prefixing
│   ├── book/     # Markdown content compiler (chapters, TOC, search, book sites)
│   ├── resilience/ # Bulkhead concurrency bounds, RateLimiter token bucket, retry with jitter, circuit breaker
│   ├── testing/  # Test harnesses and browser automation utilities
│   ├── boost/    # Boost coordination package & embedded AI agent template
│   ├── cache/    # Multi-driver cache (memory, file, Redis, Memcached)
│   ├── redis/    # Pure-Go Redis client (RESP protocol)
│   ├── orm/      # Pure-Go multi-driver ORM, SQL builder, bounded pool defaults, migrations engine
│   ├── mail/     # Multi-driver email delivery (SMTP, Resend, SendGrid, Postmark)
│   ├── commerce/ # Payment gateway integration (Midtrans) with constant-time webhook verification
│   └── collab/   # Team roles and per-environment collaboration policies
├── cmd/
│   └── ai/       # Autonomous AI agent CLI entrypoint
└── templates/    # Starter templates and scaffolding blueprints
    ├── boost/    # AI-agent workflow template and constitution
    ├── init/     # In-place project initialization templates
    └── new/      # Fresh project scaffolding blueprints
```

## Documentation Index

- [Documentation Hub](docs/index.md) — Architectural overview, package index, and guides.
- [Ecosystem Architecture](docs/foundation/architecture.md) — 6-Layer pyramid architecture, downward dependencies, and layering tests.
- [Canonical Definition](docs/foundation/canonical-definition.md) — Brand guidelines, naming standards, and taxonomy.
- [Package Inventory](docs/pkg/index.md) — Comprehensive API reference for all modular domain packages (`pkg/*`).
- [Workload Guides](docs/guides/workload-architecture-guide.md) — Architectural guide for all 8 software workload kinds.
- [Getting Started Guide](docs/getting-started.md) — Monorepo setup, Go workspace synchronization, and local testing.
- [Contributing Guidelines](docs/contributing.md) — Code quality gates, pure-Go invariant, and commit standards.

## Role in the 10-Repository Ecosystem & Battery Architecture

All workloads, libraries, and platforms advance along the **Ecosystem Architectural Battery**, fully driven by the `krew` unified toolchain CLI and declarative `krewire.yaml` configuration:

```text
Go Stdlib ──► Krewire Libraries ──► Krewire UI | Krewire Auth | Krewire Krew ──► Krewire Cloud ──► Krewire Dash
                                                                                                 (Unified Workspace)
```

Krewire is the foundational monorepo in the 10-repository ecosystem:
1. **krewire** (this repository): Foundational Go monorepo (kernel primitives and 33 modular domain libraries).
2. **krew**: Autonomous ReAct AI agent engine & unified developer toolchain (CLI, dev server, build, MCP, embed).
3. **tui**: Lightweight, pure-Go Terminal User Interface (TUI) framework and terminal design primitives.
4. **ui**: Pop-Brutalist `.kiw` design system, Go-native UI primitives, and `.kiw` template compiler & DSL engine.
5. **cloud**: Private enterprise hybrid PaaS engine and control plane.
6. **auth**: Private enterprise identity, SSO, MFA, and RBAC platform.
7. **dash**: AI-native virtual workspace, execution sandboxes, and collaboration console.
8. **internal**: Private engineering governance hub: ADRs, roadmaps, technical specifications.
9. **krewire.com**: Official production website and documentation portal.
10. **krewire.github.io**: Community launchpad, release hub, and open-source sponsorship portal.

## The Three Pillars (Secure, Sustain, and Scalable)

Every library, tool, and service in Krewire is engineered to pass the **Three Pillars**:
- **Secure**: Standard library first to eliminate supply-chain vulnerabilities, `SafePath` directory traversal guards, PBKDF2 password hashing, `SafeHTTPClient` SSRF defense, constant-time secret checks, secure-by-default HTTP primitives (OWASP/CWE-aligned), context-aware template auto-escaping, CSRF/CORS protection, and secrets referenced via `${env:NAME}` without hardcoding.
- **Sustain**: Bounded database connection pools and bulkhead concurrency limits to maintain a 10–30 MB lean RAM footprint, modular packages where you only pay for what you import, single static binaries with embedded assets, boring proven technology, a permanent MIT-licensed open-core foundation, and near-zero operational costs allowing production workloads to run on a $5 VPS.
- **Scalable**: Progressive pipeline from static sites to distributed cloud mesh without rewrites, reversible provider swapping (SQLite to Postgres, Memory to Redis), 6-layer Pyramid architecture with Bridge/Gateway decoupling, and Zero Peer Coupling between lateral engines.

## Prerequisites

- [Go](https://go.dev/) 1.27.1 or higher
- [Git](https://git-scm.com/)

## Getting Started

For a comprehensive walkthrough, empirical cohort benchmarks, and actionable diagnostics, see the **[First-Time Developer Onboarding Guide](docs/guides/first-time-developer-onboarding.md)**.

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
