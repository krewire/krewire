# Workload Maturity & Implementation Readiness Baseline

| Field | Value |
|---|---|
| ID | KWF-MAT-001 |
| Title | Workload Maturity and Capability Readiness Baseline |
| Status | Approved (Canonical Implementation Baseline) |
| Date | 2026-11-21 |
| Owners | `@krewire/team-runtime`, `@krewire/team-cloud` |
| Precedence | Precedence 1 (Core Technical Baseline) |

---

## 1. Objective Maturity Criteria

To ensure developers and AI agents can assess implementation readiness accurately without marketing bias, Krewire classifies workloads and capabilities into five objective maturity levels:

- **`Stable` (Production-Ready)**:
  - All workload conformance gates pass end to end in CI under `-strict` ([conformance evidence](./conformance.md)); no coverage percentage is claimed until CI enforces a threshold.
  - Reproducible benchmark baselines verified against performance regressions (`krew bench`).
  - Active, verified production use (e.g., [`krewire.com`](https://krewire.com) or `krew` CLI binary).
  - Backward compatibility guaranteed under Semantic Versioning (SemVer).
  - Pure-Go invariant (`CGO_ENABLED=0`) verified without external runtime dependencies.
- **`Usable` (Functional / Single-Node)**:
  - Core execution loop is functional and thoroughly covered by automated test suites.
  - Suitable for single-node deployments, self-hosted VPS environments, and early production use.
  - Known limitations exist in multi-region clustering or distributed failover.
  - Documentation and starter blueprints are published.
- **`Experimental` (Preview / Alpha)**:
  - Architecture interfaces and prototypes implemented; broader production verification underway.
  - Public API contracts may evolve between minor releases.
  - Requires explicit opt-in flags or developer configuration.
- **`Planned` (Roadmap)**:
  - Formally specified in Architecture Decision Records (ADRs) or roadmaps; implementation not yet shipped.
- **`Unsupported` (Deprecated / Out of Scope)**:
  - Explicitly excluded from framework scope or decommissioned (e.g. Node.js runtime dependencies).

---

## 2. Canonical 8-Workload Maturity Matrix

Declared statuses below are checked against the executable conformance evidence in [§6](#6-conformance-evidence-krewirekrewire141). Where the two disagree, the evidence takes precedence.

| Workload | Maturity | Primary Package | Verified Implementation & Production Evidence | Known Limitations & Gaps | Owner |
|---|---|---|---|---|---|
| **`site`** | **Stable** | `pkg/web/ssg` | Powers [`krewire.com`](https://krewire.com) in production. SSG compilation, assets pipeline, client HMR, and multi-locale i18n. | Dynamic server-side API routes require pairing with an `app` entrypoint or gateway. | `@krewire/team-runtime` |
| **`book`** | **Stable** | `pkg/book` | Powers official ecosystem documentation in English & Indonesian with automated search indexing and TOC. | Manuscript compilation currently focuses on Markdown; AsciiDoc support is planned. | `@krewire/team-runtime` |
| **`app`** | **Stable** | `pkg/app`, `pkg/web`, `pkg/runtime` | Fullstack SSR + WebAssembly islands compilation. Standard library HTTP router. 18 `.kiw` conformance tests pass. | SSR state hydration for deeply nested reactive loops is under ongoing optimization. | `@krewire/team-runtime` |
| **`cli`** | **Stable** | `krew`, `tui` (`github.com/krewire/tui`) | The `krew` developer CLI itself is built as a Krewire CLI workload. Bitwise reproducible pure-Go binary compilation verified. | Terminal graphics protocols (Sixel, Kitty) are unbundled and optional. | `@krewire/team-runtime` |
| **`gateway`** | **Stable** | `pkg/web/gateway` | High-throughput reverse proxy ingress routing, token bucket rate limiting, header rewriting, and health check pass-through. | Dynamic service discovery beyond static route tables requires external DNS or consul adapters. | `@krewire/team-runtime` |
| **`worker`** | **Usable** | `pkg/cloud/worker`, `pkg/lifecycle` | Priority queue execution, bulkhead concurrency management, exponential backoff retries, and graceful OS signal shutdown. | Redis and PostgreSQL queue backends supported; distributed Kafka/RabbitMQ adapters planned. | `@krewire/team-cloud` |
| **`service`** | **Usable** | `pkg/cloud/service`, `pkg/resilience` | Headless REST microservice runtime with correlation request tracing, circuit breakers, and bulkhead limits. | Direct HTTP/REST and SSE supported; gRPC protobuf generation integration is in roadmap. | `@krewire/team-cloud` |
| **`infra`** | **Experimental** | `pkg/cloud/infra`, `pkg/cloud/deploy` | Versioned provider contract and conformance suite; deterministic digest-bound plans; locked, checksummed state; journaled apply with recovery; drift inspection and explicit reconciliation; `localfile` reference provider passes lifecycle and failure-injection tests. | Only the `localfile` reference provider exists; Docker/systemd/cloud providers, `krew infra` command wiring, and a distributed lock backend are pending. See [promotion gates](./infra.md#9-promotion-gates-experimental--usable). | `@krewire/team-cloud` |

---

## 3. Sub-Capabilities & Provider Integration Matrix

A workload's top-level maturity does not imply that every pluggable provider shares the same readiness:

### 3.1 Database & ORM Drivers (`pkg/orm`)
- **SQLite**: **Stable** (Pure-Go driver, in-memory & file-based, bounded connection pools, zero CGO).
- **PostgreSQL**: **Stable** (Pure-Go `pgx` engine, SSL/TLS, schema migrations, pool tuning).
- **MySQL / MariaDB**: **Usable** (Pure-Go driver, standard SQL schema migrations).
- **MongoDB**: **Usable** (NoSQL document repository driver, BSON serialization).
- **Elasticsearch**: **Usable** (Search index driver and bulk indexing).

### 3.2 Cache Drivers (`pkg/cache`)
- **In-Memory**: **Stable** (Thread-safe LRU cache with TTL expiration and zero allocations).
- **Redis**: **Stable** (Pure-Go RESP protocol client, connection pooling, clustered fallback).
- **File Cache**: **Stable** (Atomic filesystem cache with SafePath protection).
- **Memcached**: **Usable** (ASCII/binary protocol driver for legacy infrastructures).

### 3.3 Identity & Security Primitives (`pkg/sec`, `pkg/auth`)
- **Password Hashing (PBKDF2)**: **Stable** (OWASP-recommended iterations, constant-time verification).
- **Path Traversal Guard (`SafePath`)**: **Stable** (Boundary-confined directory operations).
- **Security Headers & Middleware**: **Stable** (Automated CSP, HSTS, X-Frame-Options, CSRF, CORS).
- **JWT / Bearer Token Verification**: **Stable** (HMAC-SHA256, Ed25519 token validation).
- **OIDC / PKCE Federation**: **Usable** (Federated login with public discovery endpoints).
- **SAML 2.0 Identity**: **Usable** (XML signature verification and assertion parsing).

### 3.4 Background Queue Drivers (`pkg/cloud/worker`)
- **In-Memory Channel Queue**: **Stable** (Local concurrent goroutine pool with worker limits).
- **Redis Queue Backend**: **Usable** (Reliable queue via Redis lists and atomic popping).
- **PostgreSQL Queue Backend**: **Usable** (`SKIP LOCKED` transactional queue driver).
- **Kafka / RabbitMQ**: **Planned** (Distributed event streaming integration).

---

## 4. Maintenance & Review Lifecycle

1. **Ownership & Governance**:
   - Runtime workloads (`site`, `book`, `app`, `cli`, `gateway`) are maintained by `@krewire/team-runtime`.
   - Cloud workloads (`worker`, `service`, `infra`) are maintained by `@krewire/team-cloud`.
2. **Promotion Gate**:
   - A capability moves from `Experimental` to `Usable` only after automated integration tests and documented starter blueprints exist.
   - A capability moves from `Usable` to `Stable` only when verified in live production workloads and qualified by `krew release --qualify`.
3. **Automated Verification**:
   - The release qualification engine (`krew/internal/release/qualify.go`) verifies cross-repository boundary conformance on every release tag.

---

## 5. Architectural Invariants (The Three Pillars)

Every maturity designation is bound by Krewire's immutable Three Pillars:

1. **Secure**: Safety by default, stdlib-first, SafePath sanitization, zero hardcoded secrets.
2. **Sustain**: Near-zero operational cost, 10–30 MB lean RAM footprint, single static binary embed, permanent MIT open core.
3. **Scalable**: Additive progressive pipeline, reversible provider swapping, decoupled lateral gateways.

> [!IMPORTANT]
> The three pillars must remain explicitly named **Secure**, **Sustain**, and **Scalable**. They serve as the definitive qualification criteria for promoting any capability to `Stable`.

---

## 6. Conformance Evidence (krewire/krewire#141)

The end-to-end harness ([`internal/conformance`](../../internal/conformance), [method and criteria](./conformance.md)) runs each workload's fixture from a clean temp dir and records machine-readable evidence (`go run ./internal/conformance/cmd/qualify`). Status as of the baseline in [`internal/conformance/baseline.json`](../../internal/conformance/baseline.json):

| Workload | Declared | Evidenced | Conformance | Fixture | Open gaps |
|---|---|---|---|---|---|
| `site` | Stable | Usable | **passed** (12/12) | [`testdata/site`](../../internal/conformance/testdata/site) | Stable gates not automated. (`probe.notfound` was fixed: `ssg.Site.Handler` now answers unknown paths with 404 and the site's own 404 page.) |
| `book` | Stable | Usable | **passed** (12/12) | [`testdata/book`](../../internal/conformance/testdata/book) | Stable gates (compatibility, benchmark, production evidence) not automated. |
| `app` | Stable | Experimental | **partial** (12/12) | [`testdata/app`](../../internal/conformance/testdata/app) | WASM island compilation excluded here (covered by `platform_test.go` and `pkg/runtime/conformance`). |
| `cli` | Stable | Usable | **passed** (10/10, reproducible build) | [`testdata/cli`](../../internal/conformance/testdata/cli) | `tui` integration is external to core. |
| `worker` | Usable | Experimental | **partial** (10/10) | [`testdata/worker`](../../internal/conformance/testdata/worker) | Redis/PostgreSQL backends need external services; only the in-memory queue is exercised. |
| `service` | Usable | Usable | **passed** (12/12) | [`testdata/service`](../../internal/conformance/testdata/service) | None. |
| `gateway` | Stable | Usable | **passed** (12/12) | [`testdata/gateway`](../../internal/conformance/testdata/gateway) | Stable gates not automated. |
| `infra` | Experimental | Experimental | **partial** (11/11) | [`testdata/infra`](../../internal/conformance/testdata/infra) | Plan, approval-gated apply and drift/reconcile run against the `localfile` reference provider (#138); Docker, systemd and cloud providers do not exist in core and are excluded. |

The integrated AI-native workflow qualification is reported separately and is currently `excluded`: its contracts (#131/#140) are not on `main`. Release readiness therefore stays false.
