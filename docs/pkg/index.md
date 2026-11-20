# Krewire Packages — Technical Reference

Comprehensive catalog and architectural guide for all 37 Go domain packages in `github.com/krewire/krewire`.

---

## 1. Architectural Scope Levels: Go Modules vs. Go Packages

To adhere strictly to standard Golang idioms and the Krewire Scope Model (`pkg/model/scope.go`), the ecosystem enforces explicit distinctions across architectural levels:

$$\text{Workspace} \supset \text{Module} \supset \text{Domain / Package} \supset \text{Service} \supset \text{Unit}$$

- **Workspace (`go.work`)**:
  - The multi-repository development workspace at the repository hub root (`/home/krewire/workspace/krewire`).
  - Unifies all 10 ecosystem repositories with seamless local cross-module dependency resolution.
- **Go Module (`go.mod`)**:
  - The canonical unit of versioning (Semantic Versioning `vX.Y.Z`), external dependency tracking, and independent distribution.
  - Each ecosystem platform repository (`krewire`, `krew`, `cloud`, `auth`, `dash`, `ui`, `internal`, `krewire.com`, `krewire.github.io`) is an independent Go Module with its own `go.mod`.
- **Go Package (`package <name>`)**:
  - A directory of Go source files sharing the same `package <name>` declaration.
  - Every library under `krewire/pkg/*` (`pkg/web`, `pkg/orm`, `pkg/cache`, `pkg/model`, `pkg/errs`, etc.) is a **Go Package**, *not* a separate Go module. Consumers import them under the `github.com/krewire/krewire` module namespace:
    ```go
    import (
        "github.com/krewire/krewire/pkg/web"
        "github.com/krewire/krewire/pkg/orm"
        "github.com/krewire/krewire/pkg/model"
    )
    ```
- **Service (`ScopeService`)**:
  - An executable application or deployable runtime component (e.g. `cmd/kiw`, `apps/server`).
- **Unit (`ScopeUnit`)**:
  - The smallest cohesive code element inside a package (a Go function, struct, interface, or single file).

---

## 2. Architecture & Layering Invariants

All domain packages strictly conform to the downward-only dependency hierarchy described in [Architecture & Layering](../foundation/architecture.md):

```text
tools/kiw ──► pkg/* ──► pkg/model, pkg/errs ──► Go Standard Library
```

1. **Strict Downward-Only**: Packages at higher layers import lower layers; lower layers (`pkg/model`, `pkg/errs`) never import domain packages.
2. **Standard-Library-First**: Foundational packages depend primarily on the Go standard library.
3. **Pure-Go Invariant**: All packages compile with `CGO_ENABLED=0` without native C dependencies.

---

## 3. Package Domains & Reference Guides

### A. [Core & Control Plane](./core/model.md)
The bedrock runtime executor, declarative domain models, application container, and diagnostics:
- [Execution & Lifecycle Supervision (`lifecycle`)](./lifecycle.md)
- [Declarative Models & Workload Matrix (`model`)](./model.md)
- [Errors & Diagnostic Exit Codes (`errs`)](./errs.md)
- [Environment Resolution & Structured Logging (`env`, `log`)](./log.md)
- [Ecosystem Versioning & Signature (`version`)](./version.md)
- [Application Assembly & Container (`app`)](./app.md)

### B. [Web & Client Runtime](./web-runtime/web.md)
Fullstack web routing, component templates, static site generation, and interactive client WebAssembly:
- [HTTP Router, SSG, SSR & Gateway (`web`)](./web.md)
- [The `.kiw` DSL Engine & Templates (`kiw`)](./kiw.md)
- [WebAssembly VDOM Engine (`runtime`)](./runtime.md)
- [Web Components & Design Tokens (`ui`)](./ui.md)
- [Terminal UI Toolkit (`tui`)](./tui.md)
- [Asset Management & Bundling Pipeline (`assets`)](./assets.md)
- [Markdown Content & Book Compiler (`book`)](./book.md)

### C. [System, Security & Storage](./system/security-and-auth.md)
Defensive security, configuration parsing, and persistence abstractions:
- [Security Defaults & Authentication (`sec`, `auth`)](./sec.md)
- [Declarative Config & Struct Validation (`config`, `validation`)](./config.md)
- [Object Storage & Testable Filesystem (`storage`, `fs`, `file`)](./storage.md)

### D. [Platform, AI & Utilities](./platform/cloud.md)
Cloud orchestration, AI capabilities, testing infrastructure, and modular utilities:
- [Native AI SDK (`ai`)](./ai.md)
- [Cloud Orchestration & Infra Blueprints (`cloud`)](./cloud.md)
- [Test Harness & Browser Automation (`testing`)](./testing.md)
- [Caching, Redis & ORM (`cache`, `redis`, `orm`)](./orm.md)
- [Mail Delivery & Commerce (`mail`, `commerce`)](./mail.md)
- [Team Collaboration & Policies (`collab`)](./collab.md)
- [Hub, Resilience, Markdown, Functional, i18n, Term, Boost (`utilities`)](./platform/utilities.md)

---

## 4. Comprehensive Package Catalog

| Package | Documentation | Focus |
|---|---|---|
| `ai` | [`ai.md`](./ai.md) | Native AI SDK: providers, streaming, tools, schema, object generation |
| `app` | [`app.md`](./app.md) | Application container, dependency injection, and service provider lifecycle |
| `assets` | [`assets.md`](./assets.md) | Static asset pipeline, bundling, minification, and fingerprinting |
| `auth` | [`auth.md`](./auth.md) | Basic Authentication, JWT, identity, claims, and HTTP errors |
| `book` | [`book.md`](./book.md) | Markdown content compiler: chapters, TOC, search, and book-shaped sites |
| `boost` | [`boost.md`](./boost.md) | Boost AI agent coordination & embedded template installer |
| `cache` | [`cache.md`](./cache.md) | Multi-driver caching (Memory, File, Redis/Valkey, Memcached) with TTL eviction |
| `cloud` | [`cloud.md`](./cloud.md) | Cloud execution plane: runners, microservices, workers, infra blueprints |
| `collab` | [`collab.md`](./collab.md) | Team membership, hierarchical roles, and per-environment deploy policies |
| `commerce` | [`commerce.md`](./commerce.md) | Payment gateway integration (Midtrans), tokenization, and webhook verification |
| `config` | [`config.md`](./config.md) | Declarative configuration loader (`krewire.yaml`), `.env`, environment overlays |
| `env` | [`env.md`](./env.md) | Target environment resolution (`local`, `production`, `testing`) |
| `errs` | [`errs.md`](./errs.md) | POSIX exit codes, diagnostic attributes, and stack traces |
| `file` | [`file.md`](./file.md) | File-oriented operations, safe paths, and atomic replacement |
| `fs` | [`fs.md`](./fs.md) | Testable filesystem abstraction boundary |
| `functional` | [`functional.md`](./functional.md) | Generic closure-based utilities (`Map`, `Filter`, `Reduce`, `Chunk`, `Find`) |
| `hub` | [`hub.md`](./hub.md) | Dynamic capability loader, plugin interface, and package resolver |
| `i18n` | [`i18n.md`](./i18n.md) | Internationalization, localized message catalogs, pluralization, middleware |
| `kiw` | [`kiw.md`](./kiw.md) | Unified `.kiw` DSL compiler, parser, AST desugaring, component runtime |
| `lifecycle` | [`lifecycle.md`](./lifecycle.md) | Workload execution dispatch, registry resolution, and supervisor lifecycle |
| `log` | [`log.md`](./log.md) | Structured `slog` logging tailored to environments and source tagging |
| `mail` | [`mail.md`](./mail.md) | Multi-driver email delivery system (SMTP, Resend, SendGrid, Postmark, Memory) |
| `markdown` | [`markdown.md`](./markdown.md) | Markdown-to-HTML rendering and relative link prefixing |
| `model` | [`model.md`](./model.md) | Pure data models, workload matrix, scopes, tiers, and opt-in rules |
| `orm` | [`orm.md`](./orm.md) | Pure-Go multi-driver ORM & SQL Query Builder (SQLite, Postgres, MySQL); opt-in SQLite driver (`orm/sqlite`) and migration runner (`orm/migrate`) |
| `redis` | [`redis.md`](./redis.md) | Pure-Go Redis client: RESP protocol, Strings, Hashes, Lists, Sets, and Pub/Sub |
| `resilience` | [`resilience.md`](./resilience.md) | Retry policies with exponential backoff and circuit breakers |
| `runtime` | [`runtime.md`](./runtime.md) | Client WebAssembly VDOM engine, component mounting, reactive widgets |
| `sec` | [`sec.md`](./sec.md) | Security middleware, policies, SSRF validation, and PII masking |
| `storage` | [`storage.md`](./storage.md) | Backend-independent object storage (disk, memory, cloud) |
| `term` | [`term.md`](./term.md) | Terminal detection, ANSI colors, styles, and output formatting |
| `testing` | [`testing.md`](./testing.md) | Test assertions, HTTP testing harnesses, headless browser automation |
| `tui` | [`tui.md`](./tui.md) | Terminal UI toolkit powering the `kiw` CLI |
| `ui` | [`ui.md`](./ui.md) | Web UI component library, layout panels, and CSS design tokens |
| `validation` | [`validation.md`](./validation.md) | Struct-tag validation and extensible rules |
| `version` | [`version.md`](./version.md) | Semantic versioning, compatibility checks, and build watermark signature |
| `web` | [`web.md`](./web.md) | HTTP router, middleware stack, SSG, SSR, HMR, and API Gateway |
