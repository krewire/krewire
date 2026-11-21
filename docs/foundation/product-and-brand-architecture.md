# Product and Brand Architecture

| Field | Value |
|---|---|
| ID | STR-004 |
| Title | Krewire Product and Brand Architecture Specification |
| Status | Approved (Canonical Strategic Foundation) |
| Date | 2026-11-21 |
| Owners | Krewire Architecture & Strategy Maintainers |
| Precedence | Precedence 1 (Core Ecosystem Foundation) |

---

## 1. Executive Summary

This document defines the canonical product and brand architecture for the **Krewire** ecosystem. It establishes the organizational boundaries, naming standards, distribution channels, licensing postures, and component relationships across all first-party products, libraries, and developer tools.

Krewire is an umbrella ecosystem. It must not be conflated with any single repository or tool.

---

## 2. Brand Architecture Diagram

The Krewire product federation is structured as an architectural battery governed by the **6-Layer Pyramid Model** (ADR-E8P9D) and the **Three Pillars (Secure, Sustain, and Scalable)**:

```text
┌────────────────────────────────────────────────────────────────────────┐
│               LAYER 5: APEX PORTALS & UNIFIED WORKSPACES               │
│  • Krewire Dash (dash)     - AI-native virtual workspace & sandboxes   │
│  • Krewire Portal (krewire.com) - Official production site & docs      │
│  • Community Hub (krewire.github.io) - Community & sponsorship portal  │
│  • Internal SSOT (internal) - Architecture governance & ADRs           │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ consumes via Edge Gateways / APIs
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│             LAYER 4: APPLICATION ENGINES & PLATFORM SERVICES           │
│  • Krewire Cloud (cloud)   - Hybrid PaaS & multi-tenant control plane  │
│  • Krewire Auth (auth)     - Enterprise IAM, OIDC/SAML2 SSO, MFA, RBAC │
│  • Krewire Krew (krew)     - Autonomous ReAct AI agent engine (Proprietary)│
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ orchestrated by
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│              LAYER 3: UNIVERSAL DEVELOPER DRIVER & CLI                 │
│  • kiw devtool (tools/kiw) - Single CLI driving 8 workload kinds       │
│  • krewire.yaml            - Unified declarative configuration SSOT    │
│  • Krewire Boost (templates/boost) - Embedded AI agent constitution    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ imports & compiles
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   LAYER 2: MODULAR DOMAIN LIBRARIES                    │
│  • Pustaka Domain (pkg/*)  - web, orm, cache, sec, ai, runtime, tui   │
│  • Krewire UI (ui)         - Pop-Brutalist .kiw component catalog      │
│  • .kiw Engine (pkg/kiw)   - Pure-Go contextual template compiler      │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ depends on
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   LAYER 1: CORE FOUNDATION & KERNEL                    │
│  • pkg/model               - Declarative domain schemas & context SSOT │
│  • pkg/lifecycle           - Workload supervisor & process executor    │
│  • pkg/errs                - POSIX-compliant error & exit codes        │
│  • pkg/version             - Build watermark signatures & semver       │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ builds upon
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│             LAYER 0: GO STANDARD LIBRARY & OS PRIMITIVES               │
│  net/http, crypto, os, context, sync (CGO_ENABLED=0, Pure-Go Invariant)│
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Component Taxonomy & Catalog

Every first-party component in Krewire belongs to one of six architectural categories:

### 3.1 Umbrella Ecosystem (`Ecosystem`)
- **Brand Name**: **Krewire** (never *Krewire Framework* or *Krewire Project* when referring to the whole).
- **Scope**: The entire 9-repository collective, contributor community, design philosophy, and specifications.

---

### 3.2 Core Foundation & Developer Driver
- **Krewire Core** (`krewire`):
  - **Category**: Monorepo Foundation & Modular Libraries.
  - **Repository**: `github.com/krewire/krewire` (Public, MIT License).
  - **Scope**: Foundational kernel (`pkg/model`, `pkg/lifecycle`, `pkg/errs`), 37 domain libraries (`pkg/*`), and starter blueprints (`templates/`).
  - **Maturity**: Production-Ready.
- **`kiw` Devtool** (`kiw`):
  - **Category**: Universal Developer CLI Toolchain.
  - **Path**: `krewire/tools/kiw` (Public, MIT License).
  - **Scope**: Unified build runner, code generator, formatter, benchmarking suite, release qualifier, and context inspector (`kiw context`).
  - **Maturity**: Production-Ready.
- **Krewire Boost** (`Boost`):
  - **Category**: AI Agent Template & Workflow Constitution.
  - **Path**: `krewire/templates/boost` and `krewire/pkg/boost` (Public, MIT License).
  - **Scope**: Embedded multi-agent coordination system, 21 specialized engineering personas, and safety constitutions.
  - **Maturity**: Production-Ready.

---

### 3.3 Design System & UI Components
- **Krewire UI** (`ui`):
  - **Category**: Component Design System.
  - **Repository**: `github.com/krewire/ui` (Public, MIT License).
  - **Scope**: 66 Pop-Brutalist `.kiw` UI components, CSS styling tokens (`styles/krewire.css`), and pure Go `embed.FS` distribution (Model C package model). Zero npm runtime.
  - **Maturity**: Production-Ready.

---

### 3.4 Platform Engines (Enterprise & Proprietary)
- **Krewire Krew** (`krew`):
  - **Category**: Autonomous AI Agent Engine.
  - **Repository**: `github.com/krewire/krew` (Private, Proprietary Closed Source).
  - **Scope**: ReAct AI agent loop, multi-agent communication swarms, tool calling, Model Context Protocol (MCP), and IDE sidecars.
  - **Maturity**: Active Development.
- **Krewire Cloud** (`cloud`):
  - **Category**: Hybrid PaaS & Control Plane.
  - **Repository**: `github.com/krewire/cloud` (Private, Commercial).
  - **Scope**: Multi-tenant process supervisor, zero-downtime rolling deploys, edge API reverse proxy, and hermetic cloud runners.
  - **Maturity**: Staged / Enterprise Preview.
- **Krewire Auth** (`auth`):
  - **Category**: Identity & Access Management (IAM).
  - **Repository**: `github.com/krewire/auth` (Private, Commercial).
  - **Scope**: Multi-tenant OIDC/SAML 2.0 Single Sign-On (SSO), MFA (TOTP), password hashing (PBKDF2), and hierarchical RBAC.
  - **Maturity**: Staged / Enterprise Preview.
- **Krewire Dash** (`dash`):
  - **Category**: AI-Native Virtual Workspace & Sandboxes.
  - **Repository**: `github.com/krewire/dash` (Private, Commercial).
  - **Scope**: Isolated container execution sandboxes, web PTY terminals, server operations management (VPS, systemd, database, firewall), and collaboration console.
  - **Maturity**: Staged / Enterprise Preview.

---

### 3.5 Documentation & Web Portals
- **Krewire Portal** (`krewire.com`):
  - **Category**: Production Web Portal.
  - **Repository**: `github.com/krewire/krewire.com` (Proprietary Production Workload).
  - **Scope**: Official public website, interactive UI component playground, and documentation hub.
  - **Maturity**: Production-Ready.
- **Community Portal** (`krewire.github.io`):
  - **Category**: Open Source Community Portal.
  - **Repository**: `github.com/krewire/krewire.github.io` (Public, MIT License).
  - **Scope**: Community contributor guides, sponsorship portal, and public release distribution.
  - **Maturity**: Production-Ready.
- **Krewire Internal** (`internal`):
  - **Category**: Governance & Architecture SSOT.
  - **Repository**: `github.com/krewire/internal` (Private, Governance).
  - **Scope**: Architecture Decision Records (ADRs), roadmaps, contributor playbooks, and strategic research notes.
  - **Maturity**: Active Governance.

---

## 4. Naming & Brand Conventions

To maintain consistency across code, documentation, and public dialogue:

1. **Brand Capitalization**:
   - Always capitalize **Krewire** with a capital `K`.
   - The CLI tool is strictly lowercase: `kiw`.
   - The autonomous AI agent is capitalized as **Krew** (or **Krewire Krew**).
   - Component files use `.kiw` extension (lowercase).
2. **First-Party Product Naming**:
   - Official titles use the `Krewire <Product>` form: *Krewire Cloud*, *Krewire Auth*, *Krewire Dash*, *Krewire Krew*, *Krewire UI*.
   - Never use colloquial variations like *KrewCloud* or *KrewAuth*.
3. **Retired Components**:
   - *Admin Panel* has been merged into *Krewire Dash*.
4. **Cross-Linking Invariant**:
   - When referencing repository source code in documentation, always use standard GitHub links or local markdown relative links without hardcoding private secrets.

---

## 5. Architectural Invariants (3S Alignment)

The product architecture enforces the Three Pillars:

1. **Secure**:
   - Zero Peer Coupling: Lateral platform engines (`cloud`, `auth`, `krew`, `dash`) never import each other's private source trees directly. Interaction occurs exclusively through public API gateways or OIDC discovery tokens.
   - Downward-Only Layering: Code at Layer N may import only Layer < N.
2. **Sustain**:
   - Permanent MIT Open Core: Core layers (0 through 3) are forever MIT-licensed. No feature in the free tier is artificially crippled.
   - Single Binary Embed: Applications package static assets and templates via Go `embed.FS`.
3. **Scalable**:
   - Additive Progressive Pipeline: Workloads scale from static sites to distributed cloud mesh without rewrites.
