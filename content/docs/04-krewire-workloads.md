---
title: "Krewire Workloads"
description: "In-depth guide to Krewire's 8 official workload kinds: app, cli, site, book, worker, service, infra, and runtime."
date: "2026-09-29"
---

# Krewire Workloads

Krewire unifies eight distinct architectural workload kinds under one command-line interface (**`kiw`**) and one declarative devtool configuration (**`krewire.yaml`**).

The workload is declared in your project's `krewire.yaml` via the `project.kind` field. `kiw` uses this setting to drive the correct compiler, dev server, and deployment pipeline:

```yaml
project:
  name: "my-service"
  kind: "service" # app, cli, site, book, worker, service, infra, runtime
  version: "0.1.0"
```

---

## The 8 Workloads Matrix

| Workload | Kind Name | Primary Use Case | CLI Execution | Key Packages |
| :--- | :--- | :--- | :--- | :--- |
| **Fullstack Monolith** | `app` | Web applications, SSR, JSON APIs, sessions, database access | `kiw run`, `kiw dev` | `framework/web`, `framework/app`, `framework/ui` |
| **Terminal CLI & TUI** | `cli` | Command-line utilities, interactive developer TUIs | `kiw run`, `kiw build --cli` | `framework/tui`, `libs/term` |
| **Static Site (SSG)** | `site` | Landing pages, marketing websites, blogs, portfolios | `kiw build --site`, `kiw serve` | `framework/web/ssg`, `framework/dsl` |
| **Technical Book** | `book` | Multi-chapter documentation, books, software manuals | `kiw build --book`, `kiw serve` | `mdbind/book` |
| **Background Worker** | `worker` | Asynchronous job queues, cron schedules, retries, DLQ | `kiw run --worker` | `framework/worker` |
| **Microservice** | `service` | Distributed RPC/HTTP APIs, service registry, gateways | `kiw run --service` | `framework/service` |
| **Cloud Infra (IaC)** | `infra` | Provider-agnostic cloud resource provisioning (AWS/K8s) | `kiw infra apply` | `framework/infra` |
| **WASM Frontend** | `runtime` | Go-compiled client WebAssembly with reactive VDOM | `kiw build --wasm` | `framework/runtime` |

---

## 1. Fullstack Monolith (`app`)

The `app` workload produces a single, self-contained static Go binary that embeds static assets, templates, database migrations, and HTTP routing:

### Characteristics

- **Zero External Dependencies:** Built on Go's `net/http` and `embed.FS`. No external web servers (e.g. Apache, Nginx) required for local development or basic deployment.
- **Integrated Template Engine:** Server-Side Rendering (SSR) via `html/template` or `.kiw` component compilation.
- **REST & JSON APIs:** Expressive routing with groups, parameter binding, and RFC 7807 Problem Details.
- **Modular Monolith Ready (`KWF-5ZHQV`):** Internal domain boundaries allow future extraction of services without code rewrites.

### Scaffolding & Running

```bash
kiw new my-app --app
cd my-app
kiw run
```

---

## 2. Terminal CLI & TUI (`cli`)

The `cli` workload is tailored for building command-line utilities, developer tools, and rich interactive terminal user interfaces:

### Characteristics

- **Terminal Model:** Powered by `framework/tui` with an Elm/Bubble Tea-inspired Model-Update-View architecture.
- **Flag Parsing & Commands:** POSIX-compliant flag parser with automatic `--help` generation and exit-code discipline (`libs/core`).
- **Structured Logging:** Integrated with Go's `log/slog` for structured, colorized terminal output.

### Scaffolding & Running

```bash
kiw new my-cli --cli
cd my-cli
kiw run -- --verbose
```

---

## 3. Static Site Generator (`site`)

The `site` workload provides an ultra-fast static site generator powered by the `.kiw` component DSL:

### Characteristics

- **File-Based Routing:** `pages/index.kiw` serves `/`; `pages/about.kiw` serves `/about`.
- **Zero Client-Side JS:** Emits pure semantic HTML and scoped CSS by default.
- **Tailwind CSS Integration:** Automatically detects `tailwind.config.js` and compiles minified utility classes.
- **Incremental Builds:** Fast rebuilds during development (`kiw dev`) with millisecond turnaround.

### Scaffolding & Running

```bash
kiw new my-site --site
cd my-site
kiw dev # Starts dev server at localhost:8080 with live reload
```

---

## 4. Technical Book (`book`)

The `book` workload transforms Markdown manuscripts into responsive, reader-friendly documentation portals via the integrated `mdbind` engine:

### Characteristics

- **Automated Chapter Numbering:** Numbered directory and file prefixes (`01-overview.md`) map to `1. Overview`, `1.1 Subchapter`, etc.
- **Responsive Navigation:** Interactive collapsible sidebar, chapter filter search input, and mobile drawer toggler.
- **Dark/Light Theming:** Persistent theme switcher syncing with OS preferences via `localStorage`.
- **Hybrid Site Coexistence:** A book can be mounted cleanly under `/docs/` while an SSG landing page owns `/` in the same output directory.

### Scaffolding & Running

```bash
kiw new my-docs --book
cd my-docs
kiw dev
```

---

## 5. Background Worker (`worker`)

The `worker` workload runs background asynchronous job processors, distributed task queues, and scheduled cron jobs:

### Characteristics

- **Robust Retries & Backoff:** Configurable exponential backoff, jitter, and maximum retry attempts.
- **Dead-Letter Queues (DLQ):** Failed jobs are safely quarantined for inspection and replay.
- **Graceful Shutdown:** Respects OS signals (`SIGINT`, `SIGTERM`), draining active tasks before exit.

### Running

```bash
kiw run --worker
```

---

## 6. Microservice (`service`)

The `service` workload equips high-throughput Go microservices with cloud-native distributed service patterns:

### Characteristics

- **Service Registry & Discovery:** First-class integrations with Consul, etcd, NATS, and Kubernetes DNS.
- **API Gateway Routing:** Path rewriting, weighted canary traffic shifting, and token bucket rate limiting.
- **Resilience Primitives:** Circuit breakers, request timeouts, and concurrency bulkheads.
- **Distributed Tracing:** Native OpenTelemetry (OTel) instrumentation with W3C `traceparent` propagation.

### Running

```bash
kiw run --service
```

---

## 7. Cloud Infrastructure as Code (`infra`)

The `infra` workload allows developers to declare cloud architecture directly in Go:

### Characteristics

- **Declarative Go Structs:** Define VPCs, databases, object storage, and compute clusters using typed Go declarations instead of HCL or YAML.
- **Deterministic Planning:** `kiw deploy --plan` generates an execution graph with zero side-effects before applying.
- **Provider Support:** Multi-cloud drivers covering AWS (ECS, Lambda, RDS, S3, CloudFront) and Kubernetes (Deployments, Services, Ingress).

### Running

```bash
kiw deploy --plan
kiw infra apply
```

---

## 8. WebAssembly Runtime (`runtime`)

The `runtime` workload compiles Go directly to WebAssembly (`GOOS=js GOARCH=wasm`) to power interactive browser experiences:

### Characteristics

- **Virtual DOM Diffing:** High-performance Go VDOM tree with partial DOM patching.
- **Interactive Islands:** Render complete static HTML via SSR, then selectively hydrate dynamic widgets (`data-kiw-island`, `hydrate="visible|idle|load"`).
- **Standard Toolchain:** Compiles with standard `go build`—no experimental compilers or JS bundlers required.

### Running

```bash
kiw build --wasm
```

---

## Summary

By selecting a workload kind in `krewire.yaml`, developers access tailored ergonomics while retaining the universal benefits of the Go runtime and standard library.

Proceed to [**1.5 Upgrade Guide →**](/docs/upgrade-guide) to understand versioning, deprecations, and configuration migrations.
