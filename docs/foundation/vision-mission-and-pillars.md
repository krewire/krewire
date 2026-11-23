# Vision, Mission, and The Three Pillars

| Field | Value |
|---|---|
| ID | STR-002 |
| Title | Krewire Vision, Mission, The Three Pillars, and Strategic Hierarchy |
| Status | Approved (Canonical Strategic Foundation) |
| Date | 2026-11-21 |
| Owners | Krewire Architecture & Strategy Maintainers |
| Precedence | Precedence 1 (Core Ecosystem Foundation) |

---

## 1. Executive Summary

This document formalizes the strategic hierarchy of **Krewire**, articulating the relationship between:
1. **Vision** — the long-term future Krewire seeks to enable.
2. **Mission** — the enduring contribution Krewire makes toward that future.
3. **The Three Pillars** — the non-negotiable principles guiding every architectural decision.
4. **Engineering Approach** — the technologies and practices used to realize the mission.

---

## 2. Strategic Hierarchy

```text
┌────────────────────────────────────────────────────────────────────────┐
│                                VISION                                  │
│  A world where builders create, own, and evolve enduring software     │
│  without unnecessary financial, operational, or cognitive dependency.  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ guides
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                                MISSION                                 │
│  Make software engineering accessible, sovereign, and sustainable by   │
│  eliminating toolchain fragmentation and engineering near-zero cost.   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ governed by
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        THE THREE PILLARS (3S)                          │
│        [ S1: Secure ]      │    [ S2: Sustain ]    │   [ S3: Scalable ]│
│       Safety by default    │   Near-zero cost      │  Additive growth  │
│       Stdlib-first         │   10–30 MB lean RAM   │  7-stage pipeline │
│       Zero stored secrets  │   Static binary embed │  Provider swaps   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ implemented through
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          ENGINEERING APPROACH                          │
│  • Go-First & Stdlib-First (pure Go, CGO_ENABLED=0, single static bin) │
│  • Modular Libraries (pay only for what you import; zero god-modules)  │
│  • Permanent Open Core (Layer 0–3 100% MIT-licensed forever)           │
│  • AI-Native SDLC (deterministic context SSOT via kiw context & MCP)   │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Vision

### English
> **A world where builders can create, own, and evolve enduring software without unnecessary financial, operational, or cognitive dependency.**

### Indonesian (Bahasa Indonesia)
> **Dunia di mana para pembangun perangkat lunak dapat menciptakan, memiliki, dan mengembangkan software yang berdaya tahan lama tanpa ketergantungan finansial, operasional, atau kognitif yang membebani.**

### Rationale
Software creation has become artificially complex and expensive. Teams are forced to stitch together fragmented runtimes, pay escalating cloud subscription bills, manage sprawling multi-gigabyte dependency trees, and suffer vendor lock-in. Krewire envisions a sovereign future where independent builders, startups, and enterprises can own software that remains maintainable, affordable, and fully operational for decades.

---

## 4. Mission

### English
> **To make software engineering accessible, sovereign, and sustainable by eliminating toolchain fragmentation and engineering near-zero cost software ownership from initial specification to production.**

### Indonesian (Bahasa Indonesia)
> **Menjadikan rekayasa perangkat lunak mudah diakses, berdaulat, dan berkelanjutan dengan mengeliminasi fragmentasi perkakas serta merekayasa kepemilikan perangkat lunak berbiaya mendekati nol mulai dari spesifikasi awal hingga produksi.**

### Core Objectives
1. **Eliminate Toolchain Fragmentation**: Replace incompatible multi-language stacks with one cohesive, compiled Go toolchain (`krew`) spanning sites, books, apps, workers, microservices, gateways, and infrastructure.
2. **Engineer Near-Zero Cost Software Ownership**: Guarantee that production workloads run reliably with 10–30 MB RAM footprints on a minimal $5/month VPS, making software affordable for anyone in the world.
3. **Bridge Upstream Specification to Downstream Production**: Provide an unbroken path from the first written requirement to automated testing, deployment, and live monitoring without fragile handoffs.

---

## 5. The Three Pillars (3S)

The Three Pillars are Krewire's non-negotiable architectural invariants. They sit at the same conceptual level and must never be renamed, merged, reordered, or replaced with action words (such as *Protect*, *Preserve*, or *Evolve*).

### 5.1 Secure
> *"Safety is the default, never an option or a paid tier: a small attack surface, zero secrets in the tree, and behavior anyone can verify."*

- **Stdlib-First Foundation**: Prefer the vetted Go standard library (`net/http`, `crypto`, `os`) over unvetted third-party packages to minimize supply chain risk.
- **SafePath Directory Traversal Defense**: All user-provided and file-system operations are bounded and sanitized by strict path traversal guards.
- **Zero Committed Secrets**: Secrets, passwords, and tokens are dynamically resolved via environment variables (`${env:NAME}`) and never committed to source trees.
- **Contextual Auto-Escaping**: Web rendering auto-escapes HTML/JS contextually to eradicate Cross-Site Scripting (XSS).
- **Hardened HTTP Defaults**: Integrated OWASP-aligned security headers (CSP, HSTS, X-Frame-Options, nosniff), CSRF validation, and SSRF-hardened HTTP clients.

### 5.2 Sustain
> *"Software a small team—or a single builder—can still read, run, maintain, and afford a decade from now. Near-zero cost is engineered, never discounted."*

- **Near-Zero Cost Footprint**: Maintain a 10–30 MB lean RAM footprint and sub-millisecond cold starts, enabling production workloads to run indefinitely on minimal hardware.
- **Single Static Binaries**: Package applications as standalone, CGO-free binaries (`CGO_ENABLED=0`, `-ldflags="-s -w"`) with static assets and templates embedded via Go `embed.FS`.
- **Pure-Go Runtime Discipline**: UI components and server templates compile natively via Go (`.kiw` engine), avoiding fragile `node_modules` ecosystems.
- **Independent Modular Libraries**: Code under `pkg/*` is strictly decoupled; importing `pkg/web` does not pull in ORM or cloud dependencies. Projects pay only for what they import.
- **Permanent MIT Open Core**: Core layers (Layers 0 through 3) are 100% MIT-licensed forever. No feature on the free tier is ever crippled, seat-limited, or telemetry-gated.

### 5.3 Scalable
> *"Growth is additive, not a rewrite. A product pays only for the complexity it already needs, and can grow to distributed scale when it must."*

- **The 7-Stage Progressive Pipeline**:
  `[P0] Static (site/book) → [P1] Islands (runtime) → [P2] Monolith (app) → [P3] Modular Monolith (domains) → [P4] Headless Split → [P5] Services + Workers + Infra → [P6] Distributed Mesh`.
  Projects begin as modular monoliths and scale additively without rewrites.
- **Reversible Provider Swapping**: Database, cache, and storage layers are decoupled behind clean Go interfaces (`orm.DB`, `cache.Driver`, `storage.Provider`). Switching from SQLite to PostgreSQL or In-Memory to Redis requires configuration adjustments, not business logic rewrites.
- **Lateral Peer Decoupling**: Sibling platform services never import peer source trees directly; communication occurs exclusively via standard API gateways, OIDC identity federation, or asynchronous message buses.
- **Anti-God-Module Discipline**: Every unit and package has exactly one reason to change, adhering strictly to the Single Responsibility Principle.

---

## 6. Engineering Approach

To execute the Mission while honoring the Three Pillars, Krewire enforces four core engineering disciplines:

1. **Go-First & Pure-Go Compilation**:
   - Go is chosen for its deterministic garbage collector, instantaneous cold starts, single-binary distribution, and robust standard library.
   - All code compiles cleanly with `CGO_ENABLED=0`.
2. **Specification-Driven Context (SSOT)**:
   - Projects expose their authoritative context through `kiw context`, providing humans and AI agents with the same machine-readable truth (`model.ProjectContext`).
3. **Strict Downward-Only Layering (ADR-E8P9D)**:
   - Architecture follows the 6-layer pyramid: `Layer N` may import only `Layer < N`. Circular dependencies are physically prohibited.
4. **Autonomous AI-Native Ergonomics**:
   - Native integration with AI coding agents (`krew`, `boost`, IDE agents) via Model Context Protocol (MCP) and deterministic quality gates (`gofmt`, `go vet`, `go test`, `krew release --qualify`).
