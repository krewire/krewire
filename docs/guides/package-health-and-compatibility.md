# Package Health, Compatibility & Upgrade Guide

This guide establishes the package health assessment framework, API stability guarantees, breaking-change policies, and upgrade workflows across the Krewire digital SDLC ecosystem per the **Three Pillars (3S)**:
- **Secure (S1)**: Cryptographic dependency verification (`go mod verify`), AST vulnerability scanning (`govulncheck`), and zero unvetted external libraries.
- **Sustain (S2)**: 100% MIT-licensed core libraries, zero Node.js/npm runtime requirements, and pure standard-library foundation.
- **Scalable (S3)**: Predictable additive evolution without breaking changes across minor and patch releases.

---

## 1. Package Stability Tiers

Every first-party Go package under `github.com/krewire/krewire/pkg/*` and every ecosystem component exposes an explicit stability tier (`model.StabilityTier`):

| Stability Tier | Production Readiness | Compatibility Guarantee | Deprecation Window |
|---|---|---|---|
| **STABLE (✓)** | Production Ready | Strict backwards compatibility. No breaking changes permitted within minor (`vX.Y.0`) or patch (`vX.Y.Z`) releases. | Minimum 1 minor release cycle with compiler deprecation notices before removal. |
| **BETA (⚠)** | Evaluation / Pre-1.0 | Feature-complete. APIs are undergoing stabilization and may receive additive adjustments. | Breaking changes accompanied by migration guidance in release notes. |
| **EXPERIMENTAL (ℹ)** | Active Exploration | Prototype or experimental capability. APIs may change freely based on real-world feedback. | None. Not recommended for mission-critical production workloads. |
| **DEPRECATED (✗)** | Legacy / EOL | Obsolete package or symbol. Scheduled for removal in the next major release. | Active deprecation period with mandatory replacement recommendation. |
| **UNSUPPORTED (—)** | Out-of-Scope | Unmaintained or third-party component outside official support boundaries. | No maintenance or security SLA provided. |

---

## 2. Canonical First-Party Package Catalog

Metadata for foundational (Layer 1) and domain (Layer 2) libraries in `krewire`:

| Package | Layer | Stability | License | Min Go | Third-Party Dependencies | Breaking Change Policy |
|---|---|---|---|---|---|---|
| `pkg/model` | Layer 1 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: depends solely on Go stdlib and `pkg/errs`. |
| `pkg/errs` | Layer 1 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: immutable exit codes and diagnostic error constructors. |
| `pkg/lifecycle` | Layer 1 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: `Startable` interface and supervisor contracts are permanent. |
| `pkg/version` | Layer 1 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: canonical SemVer comparison and watermark rules. |
| `pkg/sec` | Layer 2 | `STABLE` | MIT | 1.22+ | 1 (`golang.org/x/net/html`) | Strict SemVer: fail-closed security defaults remain permanent. |
| `pkg/auth` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: stdlib-only crypto and context identity propagation. |
| `pkg/web` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: standard `net/http` handler interface compatibility. |
| `pkg/log` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: standard `log/slog` handler extension and correlation keys. |
| `pkg/config` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: backward-compatible declarative schema field additions. |
| `pkg/orm` | Layer 2 | `STABLE` | MIT | 1.22+ | 1 (`modernc.org/sqlite` opt-in) | Strict SemVer: query builder expressions remain strictly additive. |
| `pkg/cache` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: `cache.Driver` interface compatibility across implementations. |
| `pkg/resilience`| Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: retry policies and circuit breaker interfaces remain stable. |
| `pkg/storage` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: `storage.Provider` interface compatibility. |
| `pkg/kiw` | Layer 2 | `STABLE` | MIT | 1.22+ | 0 (Stdlib only) | Strict SemVer: compiler produces deterministic Go template bytecode. |
| `pkg/runtime` | Layer 2 | `BETA` | MIT | 1.22+ | 0 (Stdlib only) | Beta: WebAssembly VDOM diffing API stabilizing towards v1.0. |
| `pkg/ai` | Layer 2 | `BETA` | MIT | 1.22+ | 0 (Stdlib only) | Beta: LLM provider adapters and streaming abstractions add capabilities additively. |

---

## 3. Public API Compatibility & Breaking-Change Policy

Krewire enforces semantic versioning per [SemVer 2.0.0](https://semver.org/):

1. **Patch Releases (`vX.Y.Z → vX.Y.Z+1`)**:
   - Backward-compatible bug fixes, security patches, and performance optimizations only.
   - Zero modifications to exported public function signatures, struct fields, or interfaces.
2. **Minor Releases (`vX.Y.0 → vX.Y+1.0`)**:
   - Additive new features, new packages, and optional configuration options.
   - Zero breaking changes to existing exported APIs.
3. **Major Releases (`vX.0.0 → vX+1.0.0`)**:
   - May introduce breaking architectural changes only after following the formal deprecation lifecycle.
4. **Deprecation Protocol**:
   - Any symbol designated for obsolescence is annotated with standard Go doc comments:
     ```go
     // Deprecated: use auth.Identity instead.
     type Identity = auth.Identity
     ```
   - Deprecated symbols remain functional and tested for at least one minor release cycle.
   - Migration guides must provide immediate copy-paste replacement examples.

---

## 4. Multi-Repository Ecosystem Compatibility Matrix

The 9 platform repositories in the Krewire digital SDLC ecosystem are qualified against a synchronized compatibility matrix verified by `krew release --qualify`:

| Repository | Role | Owning Team | Compatible Krewire Core | Ecosystem Dependencies |
|---|---|---|---|---|
| `krewire` | Core Monorepo (`pkg/*`, `krew`) | `@krewire/team-runtime` | `v0.2.0` (Self) | None (Foundational) |
| `ui` | Pop-Brutalist Design System | `@krewire/team-runtime` | `^0.2.0` | `krewire` |
| `krew` | Autonomous AI Agent Engine | `@krewire/team-agent` | `^0.2.0` | `krewire` |
| `cloud` | Hybrid PaaS & Control Plane | `@krewire/team-cloud` | `^0.2.0` | `krewire`, `auth` |
| `auth` | Enterprise IAM & SSO Platform | `@krewire/team-iam` | `^0.2.0` | `krewire` |
| `dash` | AI-Native Virtual Workspace | `@krewire/team-workspace` | `^0.2.0` | `krewire`, `auth`, `krew` |
| `internal` | Architectural Documentation Hub | `@krewire/team-governance`| `^0.2.0` | `krewire` |
| `krewire.com` | Official Portal & Website | `@krewire/team-portal` | `^0.2.0` | `krewire`, `ui` |
| `krewire.github.io` | Sponsorship & Community Portal | `@krewire/team-portal` | `^0.2.0` | `krewire`, `ui` |

---

## 5. Upgrade Guidance & Migration Workflow

Developers upgrading dependencies within the Krewire ecosystem should follow a structured three-step verification workflow:

### Step 1: Inspect Package Health & Status
```bash
# Assess the health and compatibility contract of a specific package
kiw pkg health sec

# View the full first-party package health catalog
kiw pkg health
```

### Step 2: Update Module Dependencies
```bash
# Update ecosystem dependencies to declared compatibility constraints
krew update
```

### Step 3: Run Automated Verification Quality Gates
```bash
# 1. Run static posture and security assurance audit
krew vet --sec

# 2. Run dependency checksum and vulnerability audit
krew vet --vuln

# 3. Verify cross-repository qualification contracts
krew release --qualify
```

If `krew vet --sec` or `krew vet --vuln` detects a deprecated pattern, unpinned checksum, or security advisory, execution fails closed, blocking broken artifacts before commit or release.
