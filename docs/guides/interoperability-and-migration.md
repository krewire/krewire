# Interoperability, Hybrid Coexistence & Incremental Adoption Guide

This document establishes Krewire's technical boundaries, multi-runtime coexistence guarantees, and incremental adoption paths alongside external development stacks, JavaScript/TypeScript frontends, and mixed repositories per the **Three Pillars (3S)**:
- **Secure (S1)**: Zero silent modification of host configuration files, hard isolation between Go modules and foreign runtimes, and sanitized inter-process boundaries.
- **Sustain (S2)**: Preserve existing investments in frontend toolchains, databases, and CI pipelines without forcing destructive rewrites.
- **Scalable (S3)**: Progressive integration allowing teams to adopt a single Krewire microservice or worker while retaining existing React, Vue, Next.js, or Laravel applications.

---

## 1. Architectural Philosophy: Coexistence Without Full-Stack Rewrite

Krewire is engineered for incremental adoption. Small teams and independent developers are never forced into an all-or-nothing framework migration or rewrite. Instead, Krewire coexists transparently with existing host stacks, such as Node.js fullstack applications, Laravel PHP backends, or static repositories.

```text
Host Project (Node.js / React / Next.js / Laravel)
├── package.json / composer.json       # Host manifests (100% untouched)
├── src/ or app/                       # Host application code (untouched)
├── vite.config.js / tsconfig.json     # Host toolchain configs (untouched)
└── [Co-located Krewire Workload]
    ├── api/                           # Krewire Go Service
    │   ├── go.mod
    │   ├── krewire.yaml
    │   └── main.go
    └── worker/                        # Krewire Queue Processor
        ├── go.mod
        ├── krewire.yaml
        └── main.go
```

---

## 2. Detection Capabilities & Ecosystem Identification

The `kiw info` devtool automatically inspects and identifies external runtime environments and co-located workloads without mutating project state:

| Runtime / Host | Manifest Detected | Package Managers Identified | Frameworks Detected |
|---|---|---|---|
| **Node.js** | `package.json` | `npm`, `pnpm`, `yarn`, `bun` | React, SvelteKit, Nuxt, Next.js, Remix, Astro, Fastify, Express |
| **Laravel** | `composer.json`, `artisan` | `composer` | Laravel |
| **PHP** | `composer.json` | `composer` | Generic Composer PHP |

### Explicit Technical Boundaries & Non-Goals
- **No Automatic Code Transpilation**: Krewire does not transpile JavaScript or PHP into Go.
- **Zero Host File Mutation**: `kiw` commands will never modify, overwrite, or delete `package.json`, `package-lock.json`, `pnpm-lock.yaml`, `composer.json`, or host dependency directories (`node_modules/`, `vendor/`).
- **Detection Only**: Detection provides configuration awareness, port conflict prevention, and build suggestions (`build.depends_on`). It never injects unrequested dependencies.
- **Independent Database & Service Adoption**: Adopting a Krewire service or worker does not require migrating existing PostgreSQL, MySQL, Redis, or external microservices. Krewire connects to existing databases via standard connection strings (`${env:DB_DSN}`).

---

## 3. Toolchain & Build Pipeline Orchestration

Teams can invoke and orchestrate existing frontend build tools, package managers, and asset pipelines directly through `krewire.yaml` without writing undocumented custom glue scripts:

### Declarative Task Orchestration (`scripts:`)
```yaml
project:
  name: hybrid-app
  kind: app

server:
  port: 8080

# Declarative tasks executed via 'kiw run <task>'
scripts:
  build:fe: "npm run build"
  build:all: "npm run build && kiw build"
  dev:fe: "vite --port 5173"
  dev:all: "concurrently 'npm run dev' 'kiw dev'"
  test:all: "npm test && kiw test"
```

Developers run tasks with unified ergonomics:
```bash
# Build the frontend asset bundle via existing tools
kiw run build:fe

# Build the complete hybrid stack for production
kiw run build:all
```

---

## 4. Port Conflict Detection & Prevention

When host dev servers (e.g. Vite on `:5173`, Next.js on `:3000`) and Krewire services (`:8080`) run side-by-side in the same project directory:
- `kiw info` and `kiw dev` inspect declared ports in both host manifests (`package.json` scripts) and `krewire.yaml`.
- If conflicting ports are found, `kiw` emits warning `FRK-CONF-030` with suggestions to remap port bindings via `server.port` or CLI `--addr`.

---

## 5. Cross-Runtime Inter-Process Communication (IPC) & Tracing

For high-performance communication between foreign runtimes and compiled Go services:
- **Standard Input/Output JSON Streaming**: Launch a compiled Go worker or service as a child process via `kiw run <workload>` or direct binary execution.
- **Distributed Tracing**: Krewire automatically extracts and forwards standard W3C `traceparent` headers (`00-<trace_id>-<span_id>-<flags>`), ensuring end-to-end tracing across language boundaries.
- **Structured Log Tagging**: Emits standardized `source="go"` and `[go]` tags for unified log aggregation alongside host logs (`[js]`, `[laravel]`).

---

## 6. Usability Comparison & Evidence Metrics

Adopting Krewire incrementally alongside an existing JavaScript/TypeScript repository demonstrably reduces coordination friction, tool fragmentation, and configuration bloat:

```text
┌────────────────────────────────────────────────────────────────────────┐
│             Tooling Usability Comparison: Hybrid Adoption              │
├──────────────────────────┬──────────────────┬──────────────────────────┤
│ Metric                   │ Traditional Glue │ Krewire Hybrid Adoption  │
├──────────────────────────┼──────────────────┼──────────────────────────┤
│ Coordination Configs     │ 5 files          │ 3 files (40% reduction)  │
│                          │ (package.json,   │ (package.json,           │
│                          │  vite.config.js, │  krewire.yaml,           │
│                          │  Makefile,       │  go.mod)                 │
│                          │  docker-compose, │                          │
│                          │  dev.sh)         │                          │
├──────────────────────────┼──────────────────┼──────────────────────────┤
│ Build & Run Steps        │ 6 manual steps   │ 2 steps (67% reduction)  │
│                          │ (install node,   │ (kiw run build:all,      │
│                          │  install go,     │  kiw dev)                │
│                          │  config proxy,   │                          │
│                          │  run fe,         │                          │
│                          │  run be,         │                          │
│                          │  build glue)     │                          │
├──────────────────────────┼──────────────────┼──────────────────────────┤
│ Silent File Overwrites   │ Risk of drift    │ 0 files (Strict SSOT)    │
└──────────────────────────┴──────────────────┴──────────────────────────┘
```

*Evidence verified by `TestHybridCoexistence_UsabilityComparison` in `tools/kiw/internal/bench/bench_test.go`.*

---

## 7. Unsupported Combinations & Clear Limitations

To ensure predictability and prevent technical debt, the following patterns are explicitly unsupported:

1. **In-Memory Shared State Across Process Boundaries**: Node.js and Go cannot share in-process heap memory; communication must occur via standard HTTP/JSON, Unix domain sockets, or IPC stdin/stdout streaming.
2. **Direct Transpilation or AST Emulation**: Krewire does not transpile TypeScript, JSX, or PHP into Go bytecode.
3. **Foreign Package Manager Emulation**: `kiw` will never replace `npm`, `pnpm`, `yarn`, `bun`, or `composer`. Developers use their chosen package manager for foreign dependencies.
4. **Mutating Host Configuration**: Krewire will never inject fields into `package.json` or `composer.json` without explicit user invocation.
