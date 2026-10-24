# Krewire Packages — Package Catalog

Documentation catalog for all 37 domain packages in `github.com/krewire/krewire`.

| Module | Documentation | Focus |
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
| `orm` | [`orm.md`](./orm.md) | Pure-Go multi-driver ORM & SQL Query Builder (SQLite, Postgres, MySQL) |
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

Repository architecture is documented in [`foundation/architecture.md`](../../foundation/architecture.md), while the high-level domain guide is in [`pkg/index.md`](../index.md).