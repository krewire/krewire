# Krewire — Core Monorepo

Official monorepo for the Krewire ecosystem (`github.com/krewire/krewire`), providing modular domain packages, developer tooling, first-party applications, and agentic templates.

## Architecture & Layout

The monorepo contains four main structural areas:

```text
krewire/
├── docs/         # Ecosystem documentation (foundation, pkg, tools, templates, guides)
├── pkg/          # Domain libraries and control plane (37 modular packages)
│   ├── model/    # Declarative domain models, workload matrix, scopes, tiers, opt-in rules
│   ├── lifecycle/# Workload execution dispatch, registry resolution, and supervisor
│   ├── errs/     # POSIX-compatible exit codes, diagnostic attributes, and stack traces
│   ├── env/      # Target environment resolution (local, production, testing)
│   ├── version/  # Semantic versioning, compatibility checks, and build watermark signature
│   ├── log/      # Structured slog logging tailored to environments and source tagging
│   ├── web/      # HTTP router, middleware, SSR, SSG, gateway, HMR
│   ├── app/      # Dependency injection container & application assembly
│   ├── ui/       # UI components, layout, styling tokens, CSS theme palettes
│   ├── tui/      # Terminal UI toolkit powering the kiw CLI
│   ├── runtime/  # WebAssembly client runtime (VDOM, mounting, widgets)
│   ├── ai/       # Native AI SDK: providers, streaming, tools, schema, object generation
│   ├── sec/      # Secure defaults (headers, CSRF, CORS, SSRF, PII redaction)
│   ├── auth/     # Authentication & token verification primitives (Basic Auth, JWT)
│   ├── assets/   # Asset management, pipeline, bundling, and hash busting
│   ├── cloud/    # Runner, service, worker, storage, infra, deploy targets
│   ├── config/   # Declarative configuration loader (krewire.yaml)
│   ├── validation/ # Schema and struct validation
│   ├── storage/  # Persistent object storage abstractions (disk, memory, cloud)
│   ├── fs/       # Testable filesystem abstraction boundary
│   ├── file/     # Atomic file operations and safe replacement
│   ├── functional/ # Generic collection and functional utilities (Map, Filter, Reduce)
│   ├── hub/      # Plugin registry and package resolver
│   ├── i18n/     # Internationalization, message bundles, pluralization, middleware
│   ├── kiw/      # .kiw DSL compiler, parser, AST desugaring, component engine
│   ├── markdown/ # Markdown parsing & rendering with link prefixing
│   ├── book/     # Markdown content compiler (chapters, TOC, search, book sites)
│   ├── resilience/ # Retry policies with exponential backoff & circuit breakers
│   ├── term/     # Terminal detection, ANSI colors, and output formatting
│   ├── testing/  # Test harnesses and browser automation utilities
│   ├── boost/    # Boost coordination package & embedded AI agent template
│   ├── cache/    # Multi-driver cache (memory, file, Redis, Memcached)
│   ├── redis/    # Pure-Go Redis client (RESP protocol)
│   ├── orm/      # Pure-Go multi-driver ORM & SQL query builder
│   ├── mail/     # Multi-driver email delivery (SMTP, Resend, SendGrid, Postmark)
│   ├── commerce/ # Payment gateway integration (Midtrans)
│   └── collab/   # Team roles and per-environment collaboration policies
├── apps/         # Deployable services and applications
├── tools/        # Developer tooling
│   └── kiw/      # Unified developer CLI for project scaffolding, build, and deploy
└── templates/    # Starter templates and scaffolding blueprints
    ├── boost/    # AI-agent workflow template and constitution
    ├── init/     # In-place project initialization templates
    └── new/      # Fresh project scaffolding blueprints
```

## Role in the 8-Repository Ecosystem

Krewire is the core monorepo in the ecosystem:
1. **krewire** (this repository): Core monorepo containing packages, apps, CLI tooling (`kiw`), and starters.
2. **krew**: Autonomous AI agent monorepo (in-process embed, CLI, HTTP/SSE, MCP).
3. **cloud**: Private enterprise hybrid PaaS engine and control plane.
4. **auth**: Private enterprise identity, SSO, MFA, and RBAC platform.
5. **dash**: Krewire Dash — private enterprise AI-powered virtual workspace.
6. **internal**: Documentation hub: ADRs, roadmaps, contributor guides.
7. **krewire.com**: Official product website and documentation portal.
8. **krewire.github.io**: Community portal and guides hub.

## Prerequisites

- [Go](https://go.dev/) 1.27.1 or higher
- [Git](https://git-scm.com/)

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/krewire/krewire.git
   cd krewire
   ```

2. Build the CLI:
   ```bash
   go build -o bin/kiw ./tools/kiw/cmd/kiw
   ```

3. Run quality gates:
   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   ```

## License

[MIT](LICENSE) © 2026 Krewire Contributors
