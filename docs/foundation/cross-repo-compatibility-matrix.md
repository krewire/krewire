# Cross-Repository Compatibility Matrix & Release Qualification

This specification establishes the official compatibility matrix, version constraints, and automated release qualification process across the Krewire ecosystem ([Issue #105](https://github.com/krewire/krewire/issues/105)).

---

## 1. Objectives & Qualification Invariants

Release qualification validates that all supported components across the 10 Krewire repositories integrate cleanly and honor their contracts prior to release:

1. **Contract-Driven Verification**: Cross-repository qualification exercises actual compile-time, CLI, and runtime contracts—not merely that isolated packages pass local unit tests.
2. **Explicit Readiness Tiers**: Components and integration boundaries are classified into four deterministic statuses:
   - **`Supported`**: Generally Available (GA), feature-complete, covered by automated contract tests, and blocking for releases.
   - **`Beta`**: Pre-release preview, feature-complete for evaluation, actively tested.
   - **`Experimental`**: Architectural canary/spike, explicitly excluded from release failure gates by default.
   - **`Planned`**: Forward-looking roadmap item, not yet declared shipped.
3. **Decoupled Peer Boundaries**: Sibling platforms at Layer 4 (`cloud`, `auth`, `krew`) and Layer 5 (`dash`) never import private code laterally. Integration occurs exclusively via decoupled contracts (public interfaces, standard CLI flags, OIDC/JWKS tokens, and edge gateways).
4. **Traceable Results**: Every qualification run records tested commit SHA, resolved module versions, execution timestamp, and per-boundary diagnostic results.

---

## 2. Canonical Compatibility Matrix

| Boundary ID | Producer (Source) | Consumer (Target) | Supported Versions | Status | Owner | Contract Scope |
|---|---|---|---|---|---|---|
| `krewire-ui` | `krewire` (Core/CLI) | `ui` (Design System) | `^0.2.0` | `Supported` | `@krewire/team-runtime` | `.kiw` component compilation, tokens, and hydration directives |
| `krewire-krew` | `krewire` (Core SDK) | `krew` (Agent Engine) | `^0.2.0` | `Supported` | `@krewire/team-agent` | Core kernel models, lifecycle supervisors, and errs |
| `krewire-cloud` | `krewire` (Core/Web) | `cloud` (PaaS Control) | `^0.2.0` | `Supported` | `@krewire/team-cloud` | `pkg/cloud/runner`, `pkg/cloud/service`, and `pkg/web/gateway` |
| `krewire-auth` | `krewire` (Core/Sec) | `auth` (IAM Platform) | `^0.2.0` | `Supported` | `@krewire/team-iam` | `pkg/auth`, `pkg/sec`, PBKDF2/JWT claims, and password hashing |
| `krewire-dash` | `krewire` (Core/TUI) | `dash` (Workspace) | `^0.2.0` | `Supported` | `@krewire/team-workspace` | `pkg/tui`, `pkg/term`, and PTY terminal process contracts |
| `auth-cloud` | `auth` (Identity) | `cloud` (PaaS Gateway) | `^0.2.0` | `Supported` | `@krewire/team-iam` | Token federation, public JWKS verification, and edge headers |
| `auth-dash` | `auth` (Identity) | `dash` (Workspace) | `^0.2.0` | `Supported` | `@krewire/team-iam` | Session claims, user identity context, and RBAC policies |
| `krew-dash` | `krew` (Agent Core) | `dash` (AI Workspace) | `^0.2.0` | `Supported` | `@krewire/team-agent` | AI inference, agent task orchestration, streaming events, and approval gates |
| `cloud-mesh` | `cloud` (PaaS) | Distributed Workers | `0.3.x-exp` | `Experimental` | `@krewire/team-cloud` | Multi-node worker clustering and overlay mesh |
| `dash-p2p` | `dash` (Workspace) | Edge Peering | `unreleased` | `Planned` | `@krewire/team-workspace` | Peer-to-peer workspace sync over WebRTC |

---

## 3. Integration Boundary Contracts

### Boundary 1: `krewire` ──► `ui`
- **Contract**: `ui/kiw.pkg.json` declares a valid package manifest adhering to `https://krewire.com/schemas/kiw.pkg.schema.json`.
- **Validation**: `.kiw` component syntax compiles through `pkg/kiw` parser without syntax regressions; CSS tokens in `styles/krewire.css` exist and map to Pop-Brutalist color and border standards.

### Boundary 2: `krewire` ──► `krew`
- **Contract**: `krew` consumes `github.com/krewire/krewire/pkg/model`, `pkg/lifecycle` supervisors, and `pkg/errs` diagnostic exit codes.
- **Validation**: Autonomous agent tool dispatch, session memories, and streaming callbacks execute against stable Go core primitives.

### Boundary 3: `krewire` ──► `cloud`
- **Contract**: `cloud` consumes `pkg/cloud/runner`, `pkg/cloud/service`, and `pkg/web/gateway`.
- **Validation**: Service supervisor boots child workloads, monitors health check endpoints (`/healthz`, `/readyz`), and reverse-proxies requests through the Edge Gateway.

### Boundary 4: `krewire` ──► `auth`
- **Contract**: `auth` implements token issuance and cryptographic verification matching `pkg/auth` claims and `pkg/sec` defensive guards.
- **Validation**: Cryptographic random entropy generation, constant-time password comparisons (`subtle.ConstantTimeCompare`), and JWT header validation pass without security regressions.

### Boundary 5: `krewire` ──► `dash`
- **Contract**: `dash` consumes `pkg/tui` widget layouts, `pkg/term` ANSI color rendering, and execution sandboxes.
- **Validation**: Terminal sessions allocate virtual PTYs and stream outputs safely without leaking host credentials.

### Boundary 6: `auth` ──► `cloud`
- **Contract**: Decoupled token federation. `cloud` verifies identity tokens using public JWKS discovery endpoints without importing private `auth` database models.
- **Validation**: Bearer token parsing and role-based route gating (`cloud/pkg/gateway`) succeed with valid cryptographic signatures.

### Boundary 7: `krew` + `auth` ──► `dash`
- **Contract**: Dash acts as the human control plane for Krew agents operating with Auth-authenticated user credentials.
- **Validation**: Task execution profiles, approval requests, and diff reviews execute without unauthorized escalation.

---

## 4. Automated Qualification Command & Artifacts

The qualification suite is executed via the `krew` CLI or automated test runners:

```bash
# Human-readable summary
krew release qualify

# Machine-readable JSON output for CI pipelines
krew release qualify --json --out dist/qualification-results.json
```

### JSON Report Schema
```json
{
  "timestamp": "2026-11-21T03:00:00Z",
  "git_commit": "8547f17",
  "workspace_root": "/home/krewire/workspace/krewire",
  "module_versions": {
    "krewire": "0.2.0",
    "krew": "0.2.0",
    "cloud": "0.2.0",
    "auth": "0.2.0",
    "dash": "0.2.0",
    "ui": "0.2.0"
  },
  "total_boundaries": 11,
  "passed_count": 9,
  "failed_count": 0,
  "skipped_count": 2,
  "overall_passed": true,
  "results": [
    {
      "boundary_id": "krewire-ui",
      "source_module": "krewire",
      "target_module": "ui",
      "status": "Supported",
      "passed": true,
      "skipped": false,
      "details": "Verified @krewire/ui kiw.pkg.json manifest and component contracts",
      "error": "",
      "duration_ms": 12
    }
  ]
}
```

---

## 5. Maintenance & Ownership Protocol

1. **Adding New Boundaries**: Any new cross-repository dependency must be registered in `krew/internal/release/qualify.go` with an assigned owner, status, and automated contract test.
2. **Promoting Experimental Boundaries**: Moving a boundary from `Experimental` to `Supported` requires:
   - Full implementation in the owning repositories.
   - Comprehensive contract test verifying error handling, timeouts, and edge cases.
   - Approval by both source and target owning teams.
3. **Breaking Change Escalation**: If a contract fails in CI, release pipelines are immediately blocked. Breaking changes must either bump minor/major versions synchronously across affected modules or provide backwards-compatible adapter shims.
