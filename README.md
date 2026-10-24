# Krewire — Core Monorepo

Official monorepo for the Krewire ecosystem (`github.com/krewire/krewire`), providing modular domain packages, developer tooling, first-party applications, and agentic templates.

## Architecture & Layout

The monorepo contains four main structural areas:

```text
krewire/
├── docs/         # Ecosystem documentation (packages, boost, devtool)
├── packages/     # Domain libraries and control plane (29 modular packages)
│   ├── errs/     # POSIX-compatible exit codes, diagnostic attributes, and stack traces
│   ├── model/    # Declarative domain models, workload matrix, scopes, tiers, opt-in rules
│   ├── env/      # Target environment resolution (local, production, testing)
│   ├── version/  # Semantic versioning, compatibility checks, and build watermark signature
│   ├── log/      # Structured slog logging tailored to environments and source tagging
│   ├── lifecycle/# Workload execution dispatch, registry resolution, and supervisor
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
│   ├── file/     # Atomic file operations and safe replacement
│   ├── fs/       # Testable filesystem abstraction boundary
│   ├── functional/ # Generic collection and functional utilities (Map, Filter, Reduce)
│   ├── hub/      # Plugin registry and package resolver
│   ├── i18n/     # Internationalization, message bundles, pluralization, middleware
│   ├── kiw/      # .kiw DSL compiler, parser, AST desugaring, component engine
│   ├── markdown/ # Markdown parsing & rendering with link prefixing
│   ├── resilience/ # Retry policies with exponential backoff & circuit breakers
│   ├── storage/  # Persistent object storage abstractions (disk, memory, cloud)
│   ├── term/     # Terminal detection, ANSI colors, and output formatting
│   ├── testing/  # Test harnesses and browser automation utilities
│   ├── validation/ # Schema and struct validation
│   └── boost/    # Boost coordination package & embedded AI agent template
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
5. **workspace**: Private SaaS AI-assisted virtual workspace.
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
   go build -o ../bin/kiw ./tools/kiw/cmd/kiw
   ```

3. Run quality gates:
   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   ```

## License

[MIT](LICENSE) © 2026 Krewire Contributors
