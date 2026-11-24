# Reusable Library Contracts & Capability Boundaries

This specification documents the public contracts, capability boundaries, and stability expectations of all 37 foundational and domain modules in `github.com/krewire/krewire`.

---

## 1. Complete Package Inventory & Responsibility Boundaries

The foundational modules provide reusable primitives without embedding application-specific business logic or vendor locks:

### Layer 1: Declarative Core & Base Primitives (Zero Domain Coupling)

| Package | Layer | Responsibilities & Boundary | Stability |
| :--- | :--- | :--- | :--- |
| `errs` | Layer 1 | Structured error categories, exit codes (`0`, `1`, `2`), diagnostic attributes, and stack traces. Zero external dependencies. | Stable |
| `model` | Layer 1 | Core ecosystem domain entities, workload kinds, scope hierarchy (`Workspace` → `Unit`), licensing tiers, and validation models. | Stable |
| `version` | Layer 1 | Semantic versioning, build signatures, and compatibility verification (`IsCompatible`). | Stable |
| `file` | Layer 1 | Safe file-oriented operations, parent directory creation, and atomic file replacements (`WriteAtomic`). | Stable |
| `fs` | Layer 1 | Testable filesystem abstraction boundary (`FileSystem`, `OS`, `Memory`) on top of `io/fs`. | Stable |
| `functional` | Layer 1 | Type-safe collection transformations (`Map`, `Filter`, `Reduce`, `Chunk`) and closure composition. | Stable |
| `tui/term` | Layer 1 | Terminal detection (`IsTerminal`), ANSI colors, styles, and console presentation. | Stable |
| `validation` | Layer 1 | Reflection-based struct-tag validation engine (`required`, `email`, `min`, `max`, `pattern`). | Stable |

### Layer 2: Execution & Supervision

| Package | Layer | Responsibilities & Boundary | Stability |
| :--- | :--- | :--- | :--- |
| `lifecycle` | Layer 2 | Process lifecycle states, registry topological resolution, supervisor, and graceful shutdown traps. | Stable |
| `env` | Layer 2 | Canonical environment resolution (`local`, `production`, `testing`) and `.env` parsing. | Stable |
| `log` | Layer 2 | Structured `slog` logging with environment-specific formatting and source tags (`log.WithSource`). | Stable |

### Layer 3: Modular Domain Libraries & Engines

| Package | Layer | Responsibilities & Boundary | Stability |
| :--- | :--- | :--- | :--- |
| `web` | Layer 3 | HTTP routing, middleware, security headers (CSP, HSTS), SSG, SSR, HMR, and API gateway routing. | Stable |
| `runtime` | Layer 3 | Client-side WebAssembly Virtual DOM (VDOM) engine, component mounting, reactive widget islands (`GOOS=js`). | Stable |
| `ui/kiw` | Layer 3 | `.kiw` template parsing, AST desugaring, component runtime, and template compilers. | Stable |
| `ui` | Layer 3 | Pop-Brutalist UI component library, layout panels, and CSS styling tokens. | Stable |
| `tui` | Layer 3 | Terminal UI application builder powering the `krew` CLI (`tui.NewApp`, `tui.NewCommand`). | Stable |
| `sec` | Layer 3 | Defensive security middleware, PBKDF2 password hashing, SSRF guards (`SafeHTTPClient`), and `SafePath`. | Stable |
| `auth` | Layer 3 | Basic Authentication, HS256 JWT, claims verification, and caller identity binding. | Stable |
| `assets` | Layer 3 | Static asset management, pipeline bundling, minification, and cache-busting fingerprinting. | Stable |
| `config` | Layer 3 | Declarative YAML configuration loader (`krewire.yaml`), environment overlays, and variable expansion. | Stable |
| `storage` | Layer 3 | Backend-independent object storage abstraction (disk, memory, cloud). | Stable |
| `markdown` | Layer 3 | Fast Markdown-to-HTML rendering and relative link prefixing. | Stable |
| `book` | Layer 3 | Technical documentation book compiler, table of contents generator, and search indexing. | Stable |
| `cache` | Layer 3 | Multi-driver cache (Memory, File, Redis/Valkey, Memcached) with TTL eviction. | Stable |
| `hub/redis` | Layer 3 | Pure-Go RESP protocol Redis client (Strings, Hashes, Lists, Sets, Pub/Sub). | Stable |
| `orm` | Layer 3 | Pure-Go multi-driver ORM & SQL query builder (SQLite, PostgreSQL, MySQL) with bounded connection pools. | Stable |
| `resilience` | Layer 3 | Concurrency bulkheads, token-bucket rate limiters, exponential backoff retries, and circuit breakers. | Stable |
| `mail` | Layer 3 | Multi-driver transactional email delivery (SMTP, Resend, SendGrid, Postmark, Memory). | Stable |
| `commerce` | Layer 3 | Payment gateway client (Midtrans) with constant-time webhook verification. | Stable |
| `i18n` | Layer 3 | Localization, translation message catalogs, pluralization rules, and HTTP locale middleware. | Stable |
| `hub` | Layer 3 | Plugin registry, package resolver, and semantic version constraint solver. | Stable |
| `cloud` | Layer 3 | Cloud execution plane: task runners, microservices, background queues, and infra blueprints. | Stable |
| `collab` | Layer 3 | Team workspace membership, hierarchical roles, and per-environment deployment policies. | Stable |
| `testing` | Layer 3 | Test assertions, HTTP testing harnesses, and headless browser automation. | Stable |
| `app` | Layer 3 | Dependency injection container, service provider lifecycle, and application assembly. | Stable |

---

## 2. Library Design Discipline

1. **Stdlib-First Boundary**:
   - Prefer standard library packages (`net/http`, `crypto`, `context`, `os`, `io/fs`, `log/slog`) over third-party packages.
2. **Anti-God-Module Rule**:
   - Each module has a single reason to change. Specific application logic belongs in products or apps, never in foundational core modules.
3. **Downward-Only Imports**:
   - Foundational domain modules never import `krew` or downstream product applications.
4. **Contract Verification**:
   - All critical shared contracts have automated unit and invariant tests (`go test ./modules/...`).
