# Workload Architecture Guide: Idiomatic Go Blueprints & Trade-Offs

| Field | Value |
|---|---|
| ID | KWF-ARCH-001 |
| Title | Workload Architecture Guide: Idiomatic Go Blueprints, Patterns, and Trade-Offs |
| Status | Approved (Canonical Architecture Guidance) |
| Date | 2026-11-21 |
| Owners | Krewire Core Architecture Maintainers |
| Precedence | Precedence 1 (Core Technical Baseline) |

---

## 1. Executive Summary

Starting a new Go project presents developers with critical architectural decisions before writing a single line of business logic: defining package boundaries, separating domain logic from infrastructure, structuring application entry points, managing errors, handling concurrency, and choosing configuration patterns.

Generic software engineering advice often forces unnatural patterns onto Go—such as deep OOP inheritance trees, sprawling dependency injection frameworks, or premature microservice decomposition.

This guide provides concrete, workload-specific architectural blueprints across all eight canonical Krewire workloads. It explicitly separates **Go language idioms**, **Krewire framework invariants**, and **optional project choices**, while explaining when to use a pattern and when to avoid it.

---

## 2. The Three Architectural Convention Tiers

To eliminate confusion between language best practices and framework requirements, Krewire organizes conventions into three distinct tiers:

```text
┌────────────────────────────────────────────────────────────────────────┐
│               TIER 1: IDIOMATIC GO STANDARDS (LANGUAGE LEVEL)          │
│  • Stdlib-first package design (net/http, crypto, os, context)        │
│  • Package naming: lowercase, single-word, named for what they provide │
│  • Explicit error wrapping with fmt.Errorf("...: %w", err)             │
│  • Context propagation (ctx context.Context as first parameter)       │
│  • Goroutine lifecycle ownership & graceful shutdown with sync.WaitGroup│
│  • Table-driven tests with t.Run and fixtures in testdata/             │
│  • Pure-Go compilation with CGO_ENABLED=0                             │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ foundation for
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│               TIER 2: KREWIRE ECOSYSTEM INVARIANTS (FRAMEWORK LEVEL)   │
│  • 6-Layer Pyramid Architecture: Downward-only dependencies            │
│  • Zero Peer Coupling: Lateral engines never import private peer trees │
│  • Single Configuration SSOT: Declarative krewire.yaml schema          │
│  • Machine-Readable Context: kiw context --json for developers & AI   │
│  • The Three Pillars: Secure, Sustain, and Scalable non-negotiable    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ guides
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│           TIER 3: OPTIONAL ARCHITECTURAL CHOICES (PROJECT LEVEL)       │
│  • Database: SQLite (local/edge) vs PostgreSQL/MySQL (distributed)    │
│  • UI Rendering: SSG (static content) vs SSR + WASM (reactive apps)   │
│  • Topology: Modular Monolith vs Headless Service decomposition        │
│  • Queue Driver: In-Memory Channels vs Clustered Redis Lists           │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Workload-by-Workload Reference Blueprints

Every workload can be scaffolded using `krew new <project> --<kind>` or equipped into an existing folder via `krew init --<kind>`.

### 3.1 Fullstack Web Application (`app`)
- **Primary Package**: `web`, `app`, `runtime`, `ui`.
- **Reference Layout**:
  ```text
  my-app/
  ├── krewire.yaml          # Project configuration (port: 8080)
  ├── main.go               # Entry point: initializes app.Container & routes
  ├── internal/
  │   ├── domain/           # Core domain entities & business rules (pure Go)
  │   ├── handlers/         # HTTP request/response adapters
  │   └── store/            # Storage repository interfaces & implementations
  ├── web/
  │   ├── pages/            # Page templates & layouts (.kiw DSL)
  │   └── components/       # Reusable UI components
  └── assets/               # Static images, styles, embed.go
  ```
- **When to Use**: Dynamic web applications requiring server-rendered HTML, user sessions, database persistence, and optional client-side WebAssembly reactivity.
- **When NOT to Use**: Pure API backends (use `service`), static documentation (use `book`), or simple CLI utilities (use `cli`).
- **Anti-Pattern Guard**: Do not put business domain logic inside HTTP handler functions. Handlers should unpack requests, invoke domain services, and return views or JSON.

---

### 3.2 Command-Line Tool (`cli`)
- **Primary Package**: `krew`, `tui` (`github.com/krewire/tui`), `errs`.
- **Reference Layout**:
  ```text
  my-cli/
  ├── krewire.yaml          # CLI tool configuration
  ├── main.go               # Command dispatcher & exit code handler
  ├── cmd/                  # Subcommands (flags, help texts, validation)
  └── internal/
      └── engine/           # Execution logic decoupled from terminal I/O
  ```
- **Idiomatic Pattern**: Use standard `flag.FlagSet` or `tui.App`. Return typed POSIX exit codes (`errs.ExitCode`) and print diagnostic trees via `errs.FormatTree`.
- **When to Use**: Developer devtools, administrative utilities, code generators, and automation scripts.
- **When NOT to Use**: Background daemon workers (use `worker`) or continuous network servers (use `service`).

---

### 3.3 Static Website (`site`)
- **Primary Package**: `web/ssg`, `markdown`.
- **Reference Layout**:
  ```text
  my-site/
  ├── krewire.yaml          # SSG configuration (output: .krewire/build)
  ├── pages/                # File-based routing pages (index.kiw, about.kiw)
  ├── components/           # Modular .kiw components (Navbar.kiw, Footer.kiw)
  └── public/               # Static assets (favicons, images, robots.txt)
  ```
- **When to Use**: Marketing websites, product landing pages, blogs, and public portals.
- **When NOT to Use**: Applications requiring user authentication, dynamic per-user sessions, or background queues (use `app`).

---

### 3.4 Documentation & Books (`book`)
- **Primary Package**: `book`, `markdown`.
- **Reference Layout**:
  ```text
  my-book/
  ├── krewire.yaml          # Book config (title: "API Reference")
  ├── content/              # Markdown chapters ordered by prefix (01-intro.md)
  └── assets/               # Diagrams, screenshots, and illustrations
  ```
- **When to Use**: Software documentation, technical manuals, architectural specifications, and books.
- **When NOT to Use**: Interactive web applications with dynamic dashboards.

---

### 3.5 Background Queue Worker (`worker`)
- **Primary Package**: `cloud/worker`, `lifecycle`.
- **Reference Layout**:
  ```text
  my-worker/
  ├── krewire.yaml          # Worker config (queues: ["emails", "reports"])
  ├── main.go               # Supervisor startup, queue consumer registration
  └── internal/
      └── jobs/             # Idempotent job execution functions
  ```
- **Idiomatic Pattern**: Jobs must be idempotent. Propagate `context.Context` to every job execution; respect shutdown signals (`SIGTERM`, `SIGINT`) via `lifecycle.Supervisor` with timeout drains.
- **When to Use**: Asynchronous tasks, webhook delivery, PDF/report generation, and scheduled batch jobs.
- **When NOT to Use**: Synchronous, user-facing request-response flows.

---

### 3.6 Headless Microservice (`service`)
- **Primary Package**: `cloud/service`, `resilience`.
- **Reference Layout**:
  ```text
  my-service/
  ├── krewire.yaml          # Service config (port: 8081)
  ├── main.go               # HTTP/JSON RPC server startup
  ├── internal/
  │   ├── api/              # Route endpoints & request schemas
  │   └── service/          # High-performance business logic
  ```
- **Idiomatic Pattern**: Keep payloads structured, enforce rate limits and circuit breakers (`resilience`), and inject correlation IDs into structured logging (`log`).
- **When to Use**: High-throughput backend services, internal REST APIs, and decoupled domain microservices.
- **When NOT to Use**: Simple monolithic applications where an `app` workload avoids unnecessary network boundaries.

---

### 3.7 API Gateway & Reverse Proxy (`gateway`)
- **Primary Package**: `web/gateway`.
- **Reference Layout**:
  ```text
  my-gateway/
  ├── krewire.yaml          # Gateway route tables, upstreams, rate limits
  └── main.go               # Gateway ingress initialization
  ```
- **When to Use**: Unified edge ingress, multi-service routing, token verification, and reverse proxying across microservices.
- **When NOT to Use**: Simple applications with only one backend service.

---

### 3.8 Declarative Cloud Infrastructure (`infra`)
- **Primary Package**: `cloud/infra`, `cloud/deploy`.
- **Reference Layout**:
  ```text
  my-infra/
  ├── krewire.yaml          # Infrastructure topology definition
  └── main.go               # Provisioning plan executor (krew build --plan)
  ```
- **When to Use**: Managing environment topology, container process definitions, and deployment artifacts.
- **When NOT to Use**: General application logic.

---

## 4. Anti-Patterns & Common Pitfalls

1. **Premature Microservice Decomposition**:
   - *Pitfall*: Splitting a new project into 5 microservices before domain boundaries are clear.
   - *Guidance*: Start with a Modular Monolith (`app`). Split services only when independent deployment or differing scaling constraints demand it.
2. **The "God Module" Utility Package**:
   - *Pitfall*: Creating a single `utils` or `common` package with hundreds of unrelated helpers.
   - *Guidance*: Follow the Single Responsibility Principle. Group code by functional domain (`file`, `sec`), not vague groupings.
3. **Ignoring Context Cancellation**:
   - *Pitfall*: Spawning background goroutines without passing or respecting `ctx.Done()`.
   - *Guidance*: Always propagate `ctx` and listen for cancellation to prevent memory leaks and orphaned database connections.
4. **Hardcoding Environment Secrets**:
   - *Pitfall*: Committing API keys, database credentials, or private certificates.
   - *Guidance*: Reference secrets dynamically via `${env:SECRET_NAME}`.

---

## 5. Escape Hatches: 100% Native Go Interoperability

Krewire projects never lock developers into the `krew` CLI. Every project compiles and runs using standard Go toolchain commands:

```bash
# Build binary directly with Go stdlib toolchain
CGO_ENABLED=0 go build -o bin/app .

# Run test suites with native Go test
go test -v ./...

# Format and vet code directly
gofmt -l -w .
go vet ./...
```

The `krew` tool acts as a developer accelerator—automating scaffolding, configuration validation, and release qualification—while leaving the underlying Go project 100% standard and dependency-free.

---

## 6. Preserving the Three Pillars (3S) Invariant

All architectural guidance in this document enforces the Three Pillars:

1. **Secure**: Safety by default, stdlib-first, SafePath directory guards, zero stored secrets.
2. **Sustain**: Near-zero operational cost, 10–30 MB lean RAM footprint, single static binary embed, permanent MIT open core.
3. **Scalable**: Additive growth across 7 progressive pipeline stages without rewrites, reversible provider swapping.

> [!IMPORTANT]
> The three pillars must always appear as **Secure**, **Sustain**, and **Scalable**. They serve as the definitive architectural constraints for all Krewire projects.
