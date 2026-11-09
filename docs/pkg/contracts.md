# Reusable Library Contracts & Capability Boundaries

This specification documents the public contracts, capability boundaries, and stability expectations of Krewire's foundational and domain library packages (`pkg/*`).

---

## 1. Package Inventory & Responsibility Boundaries

The libraries under `pkg/` provide reusable primitives without embedding application orchestration or commercial policies:

| Package | Layer | Responsibilities & Boundary | Stability |
| :--- | :--- | :--- | :--- |
| `pkg/errs` | Layer 1 | Structured error categories, exit codes, and typed failures (`UsageError`, `ExitCode`). Zero external dependencies. | Stable |
| `pkg/model` | Layer 1 | Core ecosystem domain entities, workload shapes, and validation models. | Stable |
| `pkg/version` | Layer 1 | Semantic versioning, build signatures, and compatibility checking (`IsCompatible`). | Stable |
| `pkg/lifecycle`| Layer 1 | Process lifecycle states, graceful shutdown, and context signal traps. | Stable |
| `pkg/env` | Layer 1 | Environment resolution, `.env` file loading, and namespaced secret handling. | Stable |
| `pkg/log` | Layer 1 | Structured logging with source tags (`log.WithSource`). | Stable |
| `pkg/web` | Layer 2 | HTTP routing, middleware, security headers (CSP, HSTS), and SSG rendering. | Stable |
| `pkg/kiw` | Layer 2 | `.kiw` template compiling and component parsing. | Stable |
| `pkg/orm` | Layer 2 | Database abstraction and lightweight SQL builder with reversible drivers. | Evolving |
| `pkg/cache` | Layer 2 | In-memory and distributed cache interfaces (`cache.Driver`). | Evolving |
| `pkg/resilience`| Layer 2| Circuit breaking, retry with jitter, rate limiting, and bulkheading. | Stable |
| `pkg/sec` | Layer 2 | Cryptographic helpers, PBKDF2 hashing, and path-traversal guards (`SafePath`). | Stable |
| `pkg/storage` | Layer 2 | Object and blob storage abstractions (`storage.Provider`). | Evolving |
| `pkg/ai` | Layer 2 | Provider-agnostic LLM client abstractions (streaming, embeddings). | Evolving |

---

## 2. Library Design Discipline

1. **Stdlib-First Boundary**:
   - Prefer standard library packages (`net/http`, `crypto`, `context`, `os`) over third-party packages.
2. **Anti-God-Module Rule**:
   - Each package has a single reason to change. Business workflows belonging to specific applications (`panel`, `dash`) are never added to `pkg/*`.
3. **Downward-Only Imports**:
   - `pkg/*` never imports `tools/kiw` or downstream product applications.
4. **Contract Verification**:
   - All critical shared contracts have automated unit and invariant tests (`go test ./pkg/...`).
