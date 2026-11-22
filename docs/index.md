# Krewire Documentation

> **Krewire is an end-to-end, Go-first digital SDLC ecosystem engineered to build, run, and scale secure, sustainable, and scalable software products from specification to production.**

Central technical documentation portal for the Krewire core monorepo (`github.com/krewire/krewire`).

---

## 1. [Foundation & Architecture](./foundation/architecture.md)
Core architectural blueprint, canonical definitions, and system philosophy:
- **[Canonical Definition & Identity](./foundation/canonical-definition.md)** — Canonical one-sentence definition, communication tiers (short, standard, extended), and taxonomy.
- **[Vision, Mission, and The Three Pillars](./foundation/vision-mission-and-pillars.md)** — Strategic hierarchy, approved vision & mission statements, and the three pillars (Secure, Sustain, Scalable).
- **[Market Positioning & Value Proposition](./foundation/market-positioning.md)** — Target audience segments, Fragmented SDLC Tax problem definition, and verifiable proof points.
- **[Developer Value Validation & Empirical Benchmarks](./foundation/developer-value-validation.md)** — 16-participant empirical cohort research, workflow benchmarks, customer segment validation, and roadmap decision log.
- **[Competitive Differentiation Framework](./foundation/competitive-differentiation.md)** — Objective comparison matrix, defensible differentiators, and capability gap analysis.
- **[Product & Brand Architecture](./foundation/product-and-brand-architecture.md)** — First-party component catalog, naming standards, and multi-repository boundaries.
- **[Open-Source & Commercial Boundaries](./foundation/open-source-and-commercial-boundaries.md)** — Licensing matrix, permanent open-core covenant, and self-hosting guarantees.
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
- [API Gateway (`gateway`)](./workloads/gateway.md)
- [Declarative Infrastructure (`infra`)](./workloads/infra.md)

---

## 3. [Domain Packages Reference](./pkg/index.md)
Comprehensive technical documentation for all modular libraries:
- **[Core & Control Plane](./pkg/core/model.md)**: Kernel supervision, declarative models, exit codes, and logging.
- **[Web & Client Runtime](./pkg/web-runtime/web.md)**: HTTP router, SSG/SSR pipelines, client WASM engine, and UI/TUI toolkits.
- **[System, Security & Storage](./pkg/system/security-and-auth.md)**: Security defaults, authentication, configuration, validation, and object storage.
- **[Platform & Support Utilities](./pkg/platform/cloud.md)**: Cloud runners, test harnesses, plugin hub, resilience, and markdown rendering.

---

## 4. [Developer Tooling & Agent Engine (`krew`)](./krew/index.md)
The unified developer CLI for scaffolding, building, running, and shipping:
- [CLI Overview](./krew/index.md)
- [Command Reference](./krew/commands.md)
- [Configuration Specification (`krewire.yaml`)](./krew/configuration.md)
- [Development & Build Pipelines](./krew/dev-and-build.md)

---

## 5. [Templates & AI-Agent Boost](./templates/index.md)
Scaffolding blueprints and autonomous multi-agent engineering workflows:
- [Blueprints & Initializers](./templates/blueprints.md)
- [Krewire Boost AI Overview](./templates/boost/index.md)
- [21 Autonomous Agent Roles](./templates/boost/agents.md)
- [Agentic SDLC & Workflows](./templates/boost/workflow.md)
- [Agent Conventions & Safety Rules](./templates/boost/conventions.md)

---

## 6. [Guides & Best Practices](./guides/security-best-practices.md)
Operational and quality assurance playbooks:
- **[First-Time Developer Onboarding](./guides/first-time-developer-onboarding.md)** — Clean-environment quickstart, empirical cohort benchmarks (5/5 personas, 100% completion, ~1,436ms TTFR), and actionable diagnostics.
- **[Reproducible AI Verification](./guides/reproducible-ai-verification.md)** — Verification evidence model, anti-fabrication invariants, execution sandboxing, and `krew verify`.
- **[Package Health & Compatibility](./guides/package-health-and-compatibility.md)** — Stability tiers, metadata catalog, SemVer compatibility guarantees, breaking change policies, and upgrade workflows.
- **[Operational Visibility & Recovery](./guides/operational-visibility-and-recovery.md)** — Production observability, distinct health/readiness probes, graceful shutdown, backup/restore, and rollback runbooks.
- **[Security Assurance Baseline](./guides/security-assurance-baseline.md)** — Canonical threat models, verified controls, fail-closed defaults, and regression audits.
- **[Progressive Workload Transitions](./guides/progressive-workload-transitions.md)** — Tested migration paths, modular monolith vs service trade-offs, and 7-stage pipeline.
- **[Workload Architecture Guide](./guides/workload-architecture-guide.md)** — Idiomatic Go blueprints, convention tiers, and anti-pattern avoidance for all 8 workloads.
- **[AI-Native Engineering Model](./guides/ai-engineering-model.md)** — Governed AI development model across Krew, Boost, and the `krew` toolchain.
- [Project Context Model & Precedence](./guides/project-context-model.md) — Authoritative machine-readable context discovery and source precedence hierarchy.
- [Conformance & Quality Matrices](./guides/kiw-conformance-matrix.md) — Browser compatibility and template rendering conformance.
- [Multi-Runtime Coexistence (Node.js & Laravel)](./guides/multi-runtime-coexistence.md)
- [Interoperability & Incremental Adoption](./guides/interoperability-and-migration.md)
- [Security Best Practices](./guides/security-best-practices.md)
- [Testing & Quality Assurance](./guides/testing-and-qa.md)
- [Contributing Guide](../CONTRIBUTING.md)

---

## Specifications & Progress Tracking
Active specifications and progress tracking are managed directly via [GitHub Issues SSOT](https://github.com/krewire/krewire/issues?q=is%3Aissue+label%3Aspec).
Refer to [`README.md`](../README.md) and [`AGENTS.md`](../AGENTS.md) for quick setup and quality gates.
