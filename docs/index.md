# Krewire Documentation

Central technical documentation portal for the Krewire core monorepo (`github.com/krewire/krewire`).

---

## 1. [Foundation & Architecture](./foundation/architecture.md)
Core architectural blueprint and system philosophy:
- **[Architecture Specification](./foundation/architecture.md)** — Monorepo structure, strict downward-only layering invariants, domain models, and quality gates.
- **[Philosophy & Guiding Principles](./foundation/philosophy.md)** — Core tenets, Go language rationale, stdlib-first policy, and single source of truth (SSOT).

---

## 2. [Workload Spectrum (8 Kinds)](./workloads/index.md)
Dedicated guides for all eight unified software workload targets:
- [Workload Overview & Dispatch](./workloads/index.md)
- [Fullstack App (`app`)](./workloads/app.md)
- [Command-Line Tool (`cli`)](./workloads/cli.md)
- [Static Website (`site`)](./workloads/site.md)
- [Technical Book & Docs (`book`)](./workloads/book.md)
- [Background Worker (`worker`)](./workloads/worker.md)
- [Microservice (`service`)](./workloads/service.md)
- [Declarative Infrastructure (`infra`)](./workloads/infra.md)
- [Minimal Kernel Seed (`kernel`)](./workloads/kernel.md)

---

## 3. [Domain Packages Reference](./packages/index.md)
Comprehensive technical documentation for all modular libraries:
- **[Core & Control Plane](./packages/core/kern.md)**: Kernel supervision, declarative models, exit codes, and logging.
- **[Web & Client Runtime](./packages/web-runtime/web.md)**: HTTP router, SSG/SSR pipelines, client WASM engine, and UI/TUI toolkits.
- **[System, Security & Storage](./packages/system/security-and-auth.md)**: Security defaults, authentication, configuration, validation, and object storage.
- **[Platform & Support Utilities](./packages/platform/cloud.md)**: Cloud runners, test harnesses, plugin hub, resilience, and markdown rendering.

---

## 4. [Developer Tooling (`kiw`)](./tools/kiw/index.md)
The unified developer CLI for scaffolding, building, running, and shipping:
- [CLI Overview](./tools/kiw/index.md)
- [Command Reference](./tools/kiw/commands.md)
- [Configuration Specification (`krewire.yaml`)](./tools/kiw/configuration.md)
- [Development & Build Pipelines](./tools/kiw/dev-and-build.md)

---

## 5. [Templates & AI-Agent Boost](./templates/index.md)
Scaffolding blueprints and autonomous multi-agent engineering workflows:
- [Blueprints & Initializers](./templates/blueprints.md)
- [Krewire Boost AI Overview](./templates/boost/index.md)
- [20 Autonomous Agent Roles](./templates/boost/agents.md)
- [Agentic SDLC & Workflows](./templates/boost/workflow.md)
- [Agent Conventions & Safety Rules](./templates/boost/conventions.md)

---

## 6. [Guides & Best Practices](./guides/security-best-practices.md)
Operational and quality assurance playbooks:
- [Multi-Runtime Coexistence (Node.js & Laravel)](./guides/multi-runtime-coexistence.md)
- [Security Best Practices](./guides/security-best-practices.md)
- [Testing & Quality Assurance](./guides/testing-and-qa.md)
- [Contributing Guide](./guides/contributing.md)

---

## Specifications & Progress Tracking
Active specifications and progress tracking are managed directly via [GitHub Issues SSOT](https://github.com/krewire/krewire/issues?q=is%3Aissue+label%3Aspec).
Refer to [`README.md`](../README.md) and [`AGENTS.md`](../AGENTS.md) for quick setup and quality gates.
