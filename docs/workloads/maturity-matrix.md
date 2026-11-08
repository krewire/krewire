# Workload Readiness & Implementation Maturity Baseline

This document publishes the evidence-based implementation maturity assessment for all eight Krewire workloads, fulfilling the core excellence baseline for developer adoption.

---

## 1. Maturity Tiers & Criteria

Each workload is categorized into one of three explicit maturity states based on concrete repository evidence:

- **Stable (Production-Ready)**:
  - Tested across automated unit and integration suites (`go test ./...`).
  - Running in production workloads (e.g., `krewire.com`, `api.krewire.com`, or `kiw` CLI binary).
  - Fully supports pure Go (`CGO_ENABLED=0`) and the 3S Pillars (Secure, Sustainable, Scalable).
  - Complete documentation, tutorials, and starter scaffolding.
- **Usable (Beta)**:
  - Core pipelines and drivers are functional and covered by unit tests.
  - Suitable for single-node deployments, self-hosted environments, and early production workloads.
  - Known limitations exist in multi-region or distributed cluster scaling.
- **Experimental (Alpha)**:
  - Core interfaces and prototypes declared; broader automated and production verification underway.

---

## 2. Workload Maturity Matrix

| Workload | Maturity Status | Primary Module / Package | Verified Production Evidence | Known Limitations & Gaps |
|---|---|---|---|---|
| **`site`** | **Stable** | `pkg/web/ssg`, `tools/kiw` | Powers [`krewire.com`](https://krewire.com) in production. SSG generation, assets bundling, client HMR, and multi-locale i18n. | Dynamic server-side API routes require pairing with an `app` entrypoint or gateway. |
| **`book`** | **Stable** | `pkg/book`, `pkg/markdown` | Powers ecosystem documentation manuals in English and Indonesian with automated search indexing and TOC. | Manuscript asset pipeline currently focuses on Markdown; AsciiDoc support planned. |
| **`app`** | **Stable** | `pkg/web`, `pkg/app`, `pkg/runtime` | Fullstack SSR + WebAssembly (`client.wasm`) side-by-side compilation with standard library HTTP routers. | SSR state hydration for deeply nested reactive loops is under ongoing refinement. |
| **`cli`** | **Stable** | `tools/kiw`, `pkg/tui`, `pkg/term` | `kiw` developer CLI itself is written as a Krewire CLI workload. Bitwise reproducible builds and cross-compilation verified. | Terminal graphics protocol support (e.g. Sixel, Kitty) is optional and unbundled. |
| **`gateway`** | **Stable** | `pkg/web/gateway`, `cloud/pkg/gateway` | Powers live reverse proxy ingress on [`api.krewire.com`](https://api.krewire.com) routing `/auth`, `/cloud`, and `/dash`. | Dynamic service discovery beyond static route tables requires DNS or consul adapters. |
| **`worker`** | **Usable** | `pkg/cloud/worker`, `pkg/lifecycle` | Priority queue processing, concurrency management, and graceful signal shutdown tested in `tools/kiw`. | Redis and PostgreSQL queue backends implemented; distributed Kafka/RabbitMQ adapters planned. |
| **`service`** | **Usable** | `pkg/cloud/service`, `pkg/resilience` | Headless REST microservice runtime with correlation tracing, circuit breakers, and rate limiting. | gRPC proto generation integration is documented but requires external `protoc` toolchain. |
| **`infra`** | **Usable** | `pkg/cloud/infra`, `pkg/cloud/deploy` | Declarative resource topology planning (`kiw build --plan`) and state reconciliation. | Provider drivers currently target local Docker and Linux systemd; cloud PaaS provider in active verification. |

---

## 3. Prioritized Implementation Gaps

1. **Gap 1: Unified Scaffold Verification (P0)**:
   Ensure `kiw new <project>` and `kiw init` scaffold projects with 100% zero-configuration `kiw build` success across all 8 starters.
2. **Gap 2: Distributed Worker Coordination (P1)**:
   Add out-of-the-box Redis cluster locking and worker heartbeat telemetry for multi-instance deployments.
3. **Gap 3: Live Service Mesh Telemetry (P1)**:
   Standardize Prometheus exposition endpoints across headless `service` and `gateway` workloads.
