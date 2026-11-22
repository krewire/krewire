# Progressive Workload Transitions: Evolving Architectures Without Rewrites

| Field | Value |
|---|---|
| ID | KWF-EVO-001 |
| Title | Progressive Workload Transitions: Evolving Architectures Without Rewrites |
| Status | Approved (Canonical Architecture Guidance) |
| Date | 2026-11-21 |
| Owners | Krewire Core Architecture & Cloud Maintainers |
| Precedence | Precedence 1 (Core Technical Baseline) |

---

## 1. Executive Summary

A common failure mode in software engineering is the **Premature Rewrite Dilemma**: a project starts as a simple website or application, grows in traffic or functional scope, and is prematurely rewritten into a complex microservice mesh—or conversely, remains a fragile monolithic bottleneck because migration appears too risky.

Krewire resolves this tension through the **7-Stage Progressive Pipeline** (P0 through P6). The ecosystem provides tested, incremental transition paths that allow developers to add background processing, extract headless services, introduce API gateways, and manage multi-module workspaces without discarding existing code or duplicating configuration.

---

## 2. The 7-Stage Progressive Pipeline

Every Krewire project evolves along an additive continuum. Complexity is equipped only when justified by concrete operational needs:

```text
[P0: Static Content]      Marketing site, documentation manual (site, book)
        │
        ▼ (add interactivity)
[P1: Islands & WASM]      Server-rendered pages with WebAssembly widgets (pkg/runtime)
        │
        ▼ (add persistence & sessions)
[P2: Web Monolith]        Fullstack application, relational database (app)
        │
        ▼ (structure domain boundaries)
[P3: Modular Monolith]    Single binary, decoupled packages (pkg/model scope levels)
        │
        ▼ (offload heavy jobs)
[P4: Monolith + Worker]   Background job queues, async task execution (worker)
        │
        ▼ (extract high-throughput domains)
[P5: Headless Split]      Independent REST/RPC services, unified gateway (service, gateway)
        │
        ▼ (multi-region scale)
[P6: Distributed Mesh]    Declarative infrastructure, multi-tenant PaaS (infra, cloud)
```

---

## 3. When to Keep a Modular Monolith vs. When to Transition

Premature microservice decomposition introduces severe distributed systems overhead: network latency, distributed transactions, partial failures, and complex deployment pipelines.

### Keep a Modular Monolith (P2 / P3) When:
- A single team ($\le 10$ engineers) owns the codebase.
- Business transactions require strict ACID guarantees across domain boundaries.
- The entire workload runs comfortably with 10–30 MB RAM on a single $5/month VPS.
- Deployment velocity benefits from a single static binary (`CGO_ENABLED=0`).

### Transition to Specialized Workloads When:
- **Add a `worker`**: Long-running operations (PDF generation, bulk email dispatch, image resizing) block HTTP request handlers or exceed client timeout thresholds.
- **Extract a `service`**: A high-throughput domain (e.g. payment processing, telemetry ingestion) requires distinct CPU/memory profiles, independent release cycles, or tighter security sandboxing.
- **Add a `gateway`**: Multiple backend services need unified public ingress, centralized SSL/TLS termination, rate limiting, and CORS enforcement.
- **Adopt `go.work` (`ws`)**: Multiple independent git repositories or autonomous teams share common domain libraries under a unified local development environment.

---

## 4. The 5 Supported Workload Transition Paths

### 4.1 Transition Path 1: Static Site/Docs $\rightarrow$ Fullstack App (`site/book` $\rightarrow$ `app`)
- **Use Case**: A documentation book or marketing site needs user login, newsletter subscriptions, or dynamic dashboards.
- **Migration Steps**:
  1. Update `krewire.yaml` to configure server port.
  2. Add `server:` block specifying HTTP port.
  3. Create standard Go entrypoint (`main.go`) importing `pkg/web` and `pkg/app`.
  4. Move existing `.kiw` static pages into `web/pages/`.
- **Breaking Changes**: Zero. Existing `.kiw` component syntax and Pop-Brutalist styles remain 100% compatible.

---

### 4.2 Transition Path 2: Monolith $\rightarrow$ Monolith + Background Worker (`app` $\rightarrow$ `app + worker`)
- **Use Case**: Offloading asynchronous tasks from web request handlers without splitting the repository.
- **Migration Steps**:
  1. In the existing project, create a `worker` entrypoint under `cmd/worker/main.go` or sub-package `worker/`.
  2. Equip worker boilerplate: `krew init --worker` in the worker directory.
  3. Share common domain entities and database models (`internal/domain/`).
  4. Web handlers push job payloads to Redis or PostgreSQL queue (`pkg/cloud/worker`); the worker daemon consumes them asynchronously.
- **Rollback Strategy**: If worker deployment fails, background tasks can temporarily run in-process using Go goroutines via `worker.NewMemoryQueue()`.

---

### 4.3 Transition Path 3: Monolith $\rightarrow$ Headless Microservice Split (`app` $\rightarrow$ `app + service`)
- **Use Case**: Extracting a specialized subsystem (e.g. billing engine) into an independent service.
- **Migration Steps**:
  1. Define clean request/response schemas and repository interfaces in `pkg/model`.
  2. Scaffold a new headless service: `krew new billing-service --service`.
  3. Move domain logic into the new service.
  4. Replace internal monolithic function calls in `app` with resilient HTTP/JSON RPC calls using `pkg/resilience` (circuit breaker and exponential backoff).
- **Rollback Strategy**: Maintain domain interface compatibility so the service can be re-embedded as a monolithic Go package if network overhead exceeds performance gains.

---

### 4.4 Transition Path 4: Multi-Service $\rightarrow$ Unified Edge Gateway (`services` $\rightarrow$ `gateway + services`)
- **Use Case**: Centralizing public routing, authentication verification, and rate limiting across multiple services.
- **Migration Steps**:
  1. Scaffold gateway: `krew new api-gateway --gateway`.
  2. Define declarative route table in `gateway/krewire.yaml`:
     ```yaml
     gateway:
       port: 8080
       routes:
         - path: "/api/v1/auth/*"
           upstream: "http://127.0.0.1:8081"
         - path: "/api/v1/billing/*"
           upstream: "http://127.0.0.1:8082"
     ```
  3. Enable rate limiting and security headers in `pkg/web/gateway`.
- **Breaking Changes**: External clients point exclusively to port 8080; internal service ports are bound to loopback (`127.0.0.1`) and shielded by firewall.

---

### 4.5 Transition Path 5: Single Repo $\rightarrow$ Multi-Module Workspace (`go.work` / `kiw ws`)
- **Use Case**: Coordinating multiple Krewire services and libraries locally without publishing intermediate git tags.
- **Migration Steps**:
  1. In root directory, initialize workspace: `kiw ws init` or `go work init`.
  2. Add member modules: `go work use ./app ./worker ./service ./gateway`.
  3. Edits in shared libraries (`pkg/*`) take effect instantly across all sibling services.

---

## 5. Unsupported Transitions & Explicit Anti-Patterns

To protect developers from architectural traps, the following transitions are explicitly unsupported:

1. **In-Place Overwrite of Populated Projects**:
   - `krew init --app` will refuse to overwrite a populated directory containing custom `main.go` code without explicit confirmation (`--force`).
2. **Direct Conversion of CLI to Static Site**:
   - `cli` and `site` represent fundamentally different execution models. Rather than morphing a CLI binary project in-place, create a companion `site` project in the workspace (`docs/` or `site/`) to host the documentation.
3. **Premature Distributed Mesh (Jumping P0 $\rightarrow$ P6)**:
   - Jumping directly from static marketing content to a distributed microservice cluster is an anti-pattern. Follow the progressive pipeline.

---

## 6. Preserving the Three Pillars (3S) Invariant

All architectural transitions must satisfy the Three Pillars:

1. **Secure**:
   - Zero Peer Coupling: Extracted services never share private source trees directly; boundaries are mediated by public gateways or token federation.
2. **Sustain**:
   - Transitions pay only for what they equip. Adding a `worker` does not transitively bloat the `app` binary. Both run smoothly within lean 10–30 MB RAM footprints.
3. **Scalable**:
   - Every transition is additive and reversible via provider swapping (`orm.DB`, `cache.Driver`).

> [!IMPORTANT]
> The three pillars must always appear as **Secure**, **Sustain**, and **Scalable**. They ensure that architectural evolution never degrades system safety, cost sustainability, or modular scalability.
