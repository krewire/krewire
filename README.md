# Krewire — Core Monorepo

Official monorepo for the Krewire ecosystem (`github.com/krewire/krewire`), providing modular domain packages, developer tooling, first-party applications, and agentic templates.

## Architecture & Layout

The monorepo contains four main structural areas:

```text
krewire/
├── packages/     # Domain libraries and control plane
│   ├── kern/     # Kernel, lifecycle, errors, environment, model (workload & kinds)
│   ├── web/      # HTTP router, middleware, SSR, SSG, gateway
│   ├── app/      # Dependency injection container & application assembly
│   ├── ui/       # UI components, layout, styling tokens
│   ├── tui/      # Terminal UI toolkit powering the kiw CLI
│   ├── runtime/  # WebAssembly client runtime (VDOM, mounting, widgets)
│   ├── sec/      # Secure defaults (headers, CSRF, CORS, SSRF, PII redaction)
│   ├── auth/     # Authentication & token verification primitives
│   ├── cloud/    # Runner, service, worker, storage, infra deployment targets
│   ├── config/   # Declarative configuration loader
│   ├── validation/ # Schema and struct validation
│   ├── storage/  # Persistent storage abstractions
│   ├── markdown/ # Markdown parsing & rendering
│   ├── testing/  # Test harnesses and browser test utilities
│   ├── hub/      # Plugin registry and package resolver
│   ├── boost/    # Boost coordination package
│   └── docs/     # Internal package specifications and guides
├── apps/         # Deployable services and applications
│   ├── auth/     # Krewire authentication and SSO service
│   └── krewire/  # Self-hosted project and license management
├── tools/        # Developer tooling
│   └── kiw/      # Unified developer CLI for project scaffolding, build, and deploy
└── templates/    # Starter templates and scaffolding blueprints
    ├── boost/    # AI-agent workflow template and constitution
    ├── init/     # In-place project initialization templates
    └── new/      # Fresh project scaffolding blueprints
```

## Role in the 5-Repository Ecosystem

Krewire is one of 5 repositories in the ecosystem:
1. **krewire** (this repository): Core monorepo containing packages, apps, CLI tooling, and templates.
2. **mdbind**: Standalone Markdown book & docs builder powering the `book` workload.
3. **internal**: Private docs, ADRs, roadmaps, and contributor guides.
4. **krewire.com**: Product website source built with the `site` workload.
5. **krewire.github.io**: Community portal and sponsorship hub.

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
