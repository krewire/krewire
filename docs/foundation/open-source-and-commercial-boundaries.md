# Open-Source Core and Commercial Boundaries

| Field | Value |
|---|---|
| ID | GOV-001 |
| Title | Krewire Open-Source Core and Commercial Boundaries Policy |
| Status | Approved (Canonical Governance Policy) |
| Date | 2026-11-21 |
| Owners | Krewire Governance & Open-Source Maintainers |
| Precedence | Precedence 1 (Core Governance Baseline) |

---

## 1. Executive Summary

This document establishes the official governance policy for **Krewire**, defining the transparent boundary between our public open-source foundation, self-hostable developer tools, private enterprise platforms, and proprietary products.

Krewire is an open-source-first digital SDLC ecosystem. We believe that developer trust requires absolute honesty regarding what is open-source, what is self-hostable, and what constitutes commercial or enterprise software.

---

## 2. Repository Visibility and Licensing Matrix

The 9 repositories comprising the Krewire ecosystem operate under distinct licensing and visibility models:

| Repository | Visibility | License | Tier / Role | Self-Hosting Status |
|---|---|---|---|---|
| **`krewire`** | **Public** | **MIT** | Open Core Foundation (`pkg/`, `krew`) | 100% Free & Self-Hostable |
| **`ui`** | **Public** | **MIT** | Design System & `.kiw` Components | 100% Free & Self-Hostable |
| **`krewire.github.io`** | **Public** | **MIT** | Community Portal & Distribution Hub | Publicly Hosted & Distributable |
| **`krewire.com`** | **Public Workload** | **Proprietary** | Official Production Portal & Documentation | Hosted Official Workload |
| **`internal`** | **Private** | **Internal** | Architecture SSOT, ADRs, Roadmaps | Private Maintainer Hub |
| **`cloud`** | **Private** | **Commercial** | Enterprise Hybrid PaaS Control Plane | Enterprise Self-Hosted / Managed |
| **`auth`** | **Private** | **Commercial** | Enterprise IAM, SSO (OIDC/SAML2), RBAC | Enterprise Self-Hosted / Managed |
| **`dash`** | **Private** | **Commercial** | AI-Native Virtual Workspace & Sandboxes | Enterprise Workspace / SaaS |
| **`krew`** | **Private** | **Proprietary** | Autonomous ReAct AI Agent Engine | Commercial Proprietary Tool |

---

## 3. The Open-Core Covenant

Krewire adheres to an uncompromising, permanent Open-Core Covenant:

1. **Permanent MIT Open Core**:
   - Architectural Layers 0 through 3—including the core kernel (`pkg/model`, `pkg/lifecycle`, `pkg/errs`), all 37 modular domain packages (`pkg/*`), the universal developer CLI (`krew`), starter templates (`templates/`), and the UI design system (`ui`)—are **100% MIT-licensed forever**.
2. **Zero Artificial Crippleware**:
   - No feature in the open-core free tier will ever be crippled, seat-limited, time-bombed, or telemetry-gated to force upgrades to commercial tiers.
   - An independent builder or team can build, compile, deploy, and operate any of the eight workload kinds (`app`, `cli`, `site`, `book`, `worker`, `service`, `gateway`, `infra`) indefinitely without purchasing any commercial license.
3. **Additive Commercial Model**:
   - Commercial offerings (`cloud`, `auth`, `dash`) represent additive enterprise capabilities—such as managed multi-tenant PaaS orchestration, centralized enterprise SSO/SAML federation, compliance auditing, and hosted cloud execution sandboxes.
   - Commercial offerings build upon the open-core interfaces (`pkg/web/gateway`, `pkg/auth`, `pkg/cloud`) via standard APIs and tokens, without altering the open-core code.

---

## 4. Self-Hosting Guarantees and Prerequisites

### 4.1 What Is 100% Self-Hostable?
Any software built using Krewire's open-core toolchain (`krew`) is completely sovereign:
- **No License Keys**: Binaries require zero activation keys or account registrations.
- **Zero Phone-Home Telemetry**: Compiled binaries make zero network requests to Krewire servers.
- **Single Static Executable**: Applications compile to standalone binaries with static assets embedded via Go `embed.FS` (`CGO_ENABLED=0`, `-ldflags="-s -w"`).

### 4.2 Minimum Self-Hosting Prerequisites
- **Hardware**: 1 vCPU, 512 MB–1 GB RAM (easily accommodated on a $5/month VPS).
- **Operating System**: Linux (amd64, arm64), macOS, or Windows.
- **Database**: Embedded SQLite (zero-config) or external PostgreSQL/MySQL.
- **Reverse Proxy**: Host Nginx, Caddy, or built-in Krewire gateway (`pkg/web/gateway`).

---

## 5. Contribution and Redistribution Policy

- **Open Source Contributions**:
  - Contributions to `krewire`, `ui`, and `krewire.github.io` are licensed under the MIT License.
  - Contributors retain copyright of their submissions while granting Krewire the irrevocable right to distribute the code under the MIT license.
- **Redistribution**:
  - Anyone may fork, modify, vendor, package, or redistribute Krewire open-source code under the terms of the MIT License.
- **Commercial Platforms**:
  - Code in private repositories (`cloud`, `auth`, `dash`, `krew`) is proprietary and subject to commercial licensing agreements. Redistribution without written consent is prohibited.

---

## 6. Architectural Invariants (The Three Pillars)

Governance policies across the ecosystem must enforce Krewire's Three Pillars:

1. **Secure**:
   - Security is not a paid add-on. Baseline defenses (`SafePath`, CSRF, CORS, secure HTTP headers, PBKDF2 hashing) are built directly into the free MIT open-core foundation.
2. **Sustain**:
   - Near-zero cost engineering guarantees that developers can afford to run their software a decade from now without being subjected to surprise licensing changes or cloud vendor lock-in.
3. **Scalable**:
   - Applications scale smoothly from free single-node modular monoliths to enterprise multi-tenant architectures without requiring code rewrites.

> [!IMPORTANT]
> The three pillars must remain explicitly named **Secure**, **Sustain**, and **Scalable**. They define Krewire's enduring technical and moral contract with the developer community.
