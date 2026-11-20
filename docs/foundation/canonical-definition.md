# Krewire Canonical Definition, Taxonomy, and Identity

| Field | Value |
|---|---|
| ID | STR-001 |
| Title | Krewire Canonical Definition, Taxonomy, and Brand Identity |
| Status | Approved (Canonical Single Source of Truth) |
| Date | 2026-11-21 |
| Owners | Krewire Strategy & Architecture Maintainers |
| Precedence | Precedence 1 (Core Ecosystem Identity) |

---

## 1. Executive Summary

This document establishes the canonical, approved identity, definition, taxonomy, and communication standards for **Krewire**.

Historically, Krewire has been described using overlapping formulations—such as a "Go-first software engineering ecosystem", an "end-to-end digital SDLC ecosystem", an "application ecosystem", or a "collection of modular Go libraries". While these terms capture related capabilities, this document resolves ambiguity by establishing one authoritative Single Source of Truth (SSOT).

Every repository, documentation portal, CLI command, presentation, and marketing surface across the Krewire organization must align with the definitions specified herein.

---

## 2. Canonical Definitions Across Communication Tiers

### 2.1 The Canonical One-Sentence Definition

#### English
> **Krewire is an end-to-end, Go-first digital SDLC ecosystem engineered to build, run, and scale secure, sustainable, and scalable software products from specification to production.**

#### Indonesian (Bahasa Indonesia)
> **Krewire adalah ekosistem SDLC digital berbasis Go yang dirancang menyeluruh (end-to-end) untuk membangun, menjalankan, dan mengembangkan produk perangkat lunak yang aman (Secure), berkelanjutan (Sustain), dan terukur (Scalable) mulai dari spesifikasi hingga produksi.**

---

### 2.2 Short Variant (~120–160 characters)
*Intended for GitHub repository descriptions, package manager tags, social media bios, and search engine metadata.*

- **English**:
  "The Go-first digital SDLC ecosystem for building secure, sustainable, and scalable software."
- **Indonesian**:
  "Ekosistem SDLC digital berbasis Go untuk membangun perangkat lunak yang aman, berkelanjutan, dan terukur."

---

### 2.3 Standard Variant (~50 words)
*Intended for repository README headers, documentation overviews, landing page hero summaries, and press introductions.*

- **English**:
  "Krewire is a Go-first digital SDLC ecosystem that unifies developer tooling, modular domain libraries, and application runtimes under one cohesive architecture. Built around three immutable pillars—Secure, Sustain, and Scalable—it eliminates toolchain fragmentation and delivers near-zero cost software ownership from initial specification to cloud production."
- **Indonesian**:
  "Krewire adalah ekosistem SDLC digital berbasis Go yang menyatukan perkakas pengembang, pustaka domain modular, dan runtime aplikasi dalam satu arsitektur yang padu. Dibangun di atas tiga pilar utama—Secure, Sustain, dan Scalable—Krewire mengeliminasi fragmentasi toolchain dan menghadirkan kepemilikan perangkat lunak berbiaya mendekati nol mulai dari spesifikasi awal hingga produksi cloud."

---

### 2.4 Extended Variant (~150–250 words)
*Intended for executive summaries, whitepapers, keynote addresses, contributor guides, and institutional partnerships.*

- **English**:
  "Krewire is an end-to-end digital SDLC ecosystem engineered in Go to bridge the divide between software consumption and sovereign software creation. Modern engineering is plagued by deep toolchain fragmentation, bloated runtime dependencies, fragile build systems, and prohibitive operational costs. Krewire replaces this complexity with a unified, standard-library-aligned battery driven by a single developer CLI (`kiw`).
  
  The ecosystem spans the entire software development lifecycle—from early specification and autonomous AI agents, to typed domain libraries, pure-Go UI rendering, enterprise identity (SSO/RBAC), hybrid PaaS, and self-hosted server orchestration.
  
  Everything in Krewire is governed by three non-negotiable architectural pillars: **Secure** (safety by default, stdlib-first, zero hardcoded secrets), **Sustain** (near-zero cost engineering, 10–30 MB lean RAM footprints, single static binary embed, permanent MIT open core), and **Scalable** (additive growth across seven pipeline stages without rewrites). Through this cohesive architecture, Krewire ensures that a single developer or small engineering team can build and afford sovereign, world-class digital products a decade into the future."

- **Indonesian**:
  "Krewire adalah ekosistem SDLC digital menyeluruh berbasis Go yang dirancang untuk menjembatani jurang antara konsumsi perangkat lunak dan kedaulatan penciptaan perangkat lunak. Rekayasa perangkat lunak modern kerap terbebani oleh fragmentasi perkakas, ketergantungan runtime yang membengkak, sistem build yang rapuh, dan biaya operasional yang mahal. Krewire menggantikan kompleksitas tersebut dengan rangkaian pustaka terpadu selaras pustaka standar Go yang digerakkan oleh satu CLI pengembang (`kiw`).
  
  Ekosistem ini mencakup seluruh siklus hidup pengembangan perangkat lunak—mulai dari perumusan spesifikasi awal dan agen AI otonom, hingga pustaka domain bertipe, rendering UI murni Go, identitas enterprise (SSO/RBAC), hybrid PaaS, dan orkestrasi server mandiri.
  
  Seluruh komponen Krewire diatur oleh tiga pilar arsitektur yang tidak dapat diubah: **Secure** (keamanan bawaan, berlandaskan stdlib, tanpa rahasia hardcoded), **Sustain** (rekayasa biaya mendekati nol, konsumsi memori 10–30 MB, biner statis mandiri, open core MIT permanen), dan **Scalable** (pertumbuhan aditif di tujuh tahap tanpa tulis ulang kode). Melalui arsitektur terpadu ini, Krewire memastikan pengembang tunggal maupun tim kecil mampu membangun dan membiayai produk digital kelas dunia yang berdaulat hingga satu dekade ke depan."

---

## 3. Taxonomy & Structural Distinctions

To prevent organizational confusion, maintainers and AI agents must strictly observe the following architectural distinctions:

### 3.1 Ecosystem (`Ekosistem`)
- **Definition**: The entire multi-repository federation, collective community, design system, specifications, developer tools, and platform engines.
- **Rule**: Never refer to the single core monorepo (`krewire/krewire`) as "the ecosystem". The core monorepo is merely the foundational component *of* the ecosystem.

### 3.2 Core Monorepo (`Inti / Core`)
- **Definition**: The repository [`krewire/krewire`](https://github.com/krewire/krewire).
- **Contents**: Foundational kernel packages (`pkg/model`, `pkg/lifecycle`, `pkg/errs`), modular domain packages (`pkg/*`), the developer CLI driver (`tools/kiw`), and official starter blueprints (`templates/`).
- **Licensing**: 100% MIT Open Source forever.

### 3.3 Framework & Modular Libraries (`Pustaka Modular`)
- **Definition**: The independent Go packages housed under `krewire/pkg/*` (`web`, `orm`, `cache`, `sec`, `ui`, `ai`, `resilience`, etc.) and the component compiler (`pkg/kiw`).
- **Rule**: Krewire is distributed as modular, composable libraries, not an inflexible mega-framework. Projects import only what they need; unreferenced packages add zero bytes and zero runtime cost.

### 3.4 Toolchain (`Toolchain / Perkakas Pengembang`)
- **Definition**: The unified developer CLI `kiw` (`tools/kiw`), declarative configuration engine (`krewire.yaml`), code generators, formatters, and release qualification runners.
- **Rule**: `kiw` drives all eight workload kinds (`app`, `cli`, `site`, `book`, `worker`, `service`, `gateway`, `infra`).

### 3.5 Platform Products (`Produk Platform`)
- **Definition**: Distinct higher-tier applications, services, and portals built on top of the Krewire core:
  - **Krewire Krew** (`krew`): Autonomous ReAct AI agent engine (Proprietary / Closed Source).
  - **Krewire Cloud** (`cloud`): Hybrid PaaS control plane, process supervisor, and edge gateway (Private Enterprise).
  - **Krewire Auth** (`auth`): Multi-tenant SSO (OIDC/SAML2), MFA (TOTP), and RBAC platform (Private Enterprise).
  - **Krewire Dash** (`dash`): AI-native virtual workspace, execution sandboxes, and collaboration console (Private Enterprise).
  - **Krewire Portal** (`krewire.com`): Official website and documentation hub.
  - **Community Portal** (`krewire.github.io`): Community sponsorship and public distribution portal.

---

## 4. Licensing Posture & Maturity Matrix

| Repository | Scope | License | Posture / Tier | Maturity Level |
|---|---|---|---|---|
| `krewire` | Core monorepo (`pkg/`, `tools/kiw`) | MIT | Open Core Foundation | Production-Ready |
| `ui` | Component catalog & design tokens | MIT | Open Core Foundation | Production-Ready |
| `krewire.github.io` | Community and sponsorship hub | MIT | Open Source Community | Production-Ready |
| `krewire.com` | Official website & documentation portal | Proprietary | Official Production Workload | Production-Ready |
| `internal` | Private governance & architecture ADRs | Private | Internal Maintainer Hub | Active Governance |
| `cloud` | Hybrid PaaS & control plane | Commercial | Enterprise PaaS Platform | Staged / Preview |
| `auth` | Enterprise IAM, SSO & RBAC | Commercial | Enterprise Security Platform | Staged / Preview |
| `dash` | Virtual workspace & dev sandboxes | Commercial | Enterprise Collaboration | Staged / Preview |
| `krew` | Autonomous ReAct AI agent engine | Proprietary | Commercial AI Platform | Active Development |

---

## 5. The Three Pillars (3S) Guardrail

The three architectural pillars of Krewire are non-negotiable and strictly invariant:

1. **Secure**: Safety by default, standard-library-first dependencies, path traversal guards (`SafePath`), zero hardcoded secrets, contextual escaping.
2. **Sustain**: Near-zero operational cost, 10–30 MB lean RAM footprint, single static binary embed (`CGO_ENABLED=0`), zero npm/node runtime, permanent MIT open core.
3. **Scalable**: Additive growth across the 7-stage progressive pipeline without rewrites, reversible database/cache provider swapping, decoupled lateral gateways.

> [!IMPORTANT]
> The three pillars must remain exactly **Secure**, **Sustain**, and **Scalable**. Keep all three names explicit, immutable, and at the same architectural tier. Never rename, merge, reorder, or replace them with action verbs or synonyms (such as *Protect*, *Preserve*, or *Evolve*).

---

## 6. Documentation & Website Synchronization Checklist

When updating public documentation, repository descriptions, or marketing materials:

- [ ] Confirm repository description matches the **Short Variant**.
- [ ] Confirm README introduction uses the **Canonical One-Sentence** or **Standard Variant**.
- [ ] Ensure distinction between the open-source **Core Monorepo** and higher-tier **Platform Products** is preserved.
- [ ] Ensure open-source scope is accurate (do not claim closed-source or enterprise platforms are MIT-licensed).
- [ ] Ensure maturity claims accurately reflect the **Maturity Level** matrix.
- [ ] Verify the Three Pillars are explicitly written as **Secure**, **Sustain**, and **Scalable**.
