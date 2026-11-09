# Ecosystem Dependency Model & Directed Acyclic Graph (DAG)

This specification defines the canonical architectural DAG, package ownership boundaries, and dependency invariants across the Krewire ecosystem.

---

## 1. The Ecosystem Capability & Dependency DAG

The Krewire ecosystem follows a strict downward dependency hierarchy:

```text
Layer 5: Applications & Portals (krewire.com, krewire.github.io)
   ↓
Layer 4: Platform Engines & Standalone Products
   ├── Parallel Products: Dash (Virtual Workspace) || Forge (Panel & Form Builder)
   └── Horizontal Platform Capabilities: Cloud (PaaS) || Auth (IAM/SSO)
   ↓
Layer 3: Developer Driver & CLI (tools/kiw)
   ↓
Layer 2: UI Foundation (kui / .kiw components) & Domain Libraries (pkg/*)
   ↓
Layer 1: Core Foundation & Kernel (pkg/model, pkg/errs, pkg/version, pkg/lifecycle)
   ↓
Layer 0: Go Standard Library & OS Primitives
```

---

## 2. Compile-Time vs. Runtime Integration Boundaries

A critical architectural distinction is maintained between **compile-time dependencies** and **runtime integrations**:

### Compile-Time Dependencies (Go Module Imports)
- **Strict Downward-Only**: A higher layer may import a lower layer (`Layer N → Layer < N`). Lower layers must **never** import higher layers.
- **Zero Peer Coupling (Lateral Independence)**: Parallel Layer 4 products (`forge` and `dash`) are lateral peers. They **never** import each other's Go packages or private source trees.
- **Tooling Isolation Guard**: Pustaka domain (`pkg/*`) must never import the CLI tool (`tools/kiw`).

### Runtime / Service Integrations (Decoupled Protocols)
- **Horizontal Services**: `cloud` and `auth` are horizontal platform capabilities. Products (`dash`, `forge`, or third-party applications) integrate with them over standard decoupled interfaces:
  - OpenID Connect (OIDC) / OAuth 2.0 / JWKS public token verification.
  - HTTP REST APIs via edge reverse proxy / gateway (`https://api.krewire.com/auth`, `https://api.krewire.com/cloud`).
  - Asynchronous event bus / SSE event streams.
- **No Mandatory Coupling**: Hosted `cloud` and `auth` are never mandatory prerequisites for standalone product operation. Both `forge` and `dash` can operate completely self-hosted and standalone without hosted cloud subscriptions.

---

## 3. Product Ownership & Parallelism

- **Forge**: Declarative web, form, resource, and admin panel builder for Go and `.kiw`.
- **Dash**: Virtual developer workspace, execution sandboxes, and collaboration console.
- Neither `forge` nor `dash` depends on the other at compile time or deployment time.

---

## 4. Enforcement & Automated Quality Gates

Layering invariants are enforced automatically in CI and local testing:
- `layering_test.go`: Enforces downward-only imports, foundation rankings, and tooling isolation.
- `gofmt` & `go vet`: Validates structural hygiene and absence of cyclic dependencies.
- `CGO_ENABLED=0 go build ./...`: Enforces pure-Go portability without native C runtime dependencies.
