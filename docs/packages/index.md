# Krewire Packages — Technical Reference

Comprehensive catalog and dependency guide for domain libraries in `github.com/krewire/krewire`.

---

## Architecture & Layering

All domain packages strictly conform to the downward-only dependency hierarchy described in [Architecture & Layering](../foundation/architecture.md):
```text
tools/kiw ──► packages/* ──► packages/model, packages/errs ──► Go Standard Library
```

---

## Package Domains

### 1. [Core & Control Plane](./core/kern.md)
The bedrock runtime executor, domain models, application assembly, and error management:
- [Execution & Lifecycle Supervision (`lifecycle`)](./modules/lifecycle.md)
- [Declarative Models & Workload Matrix (`model`)](./modules/model.md)
- [Errors & Diagnostic Exit Codes (`errs`)](./modules/errs.md)
- [Environment Resolution & Structured Logging (`env`, `log`)](./modules/log.md)
- [Ecosystem Versioning & Signature (`version`)](./modules/version.md)
- [Application Assembly & Container (`app`)](./modules/app.md)

### 2. [Web & Client Runtime](./web-runtime/web.md)
Fullstack web routing, component templates, static site generation, and interactive client WebAssembly:
- [HTTP Router, SSG, SSR & Gateway (`web`)](./web-runtime/web.md)
- [The `.kiw` DSL Engine & Templates (`kiw`)](./modules/kiw.md)
- [WebAssembly VDOM Engine (`runtime`)](./web-runtime/runtime.md)
- [Web Components & Design Tokens (`ui`)](./web-runtime/ui-tui.md)
- [Terminal UI Toolkit (`tui`)](./web-runtime/ui-tui.md)
- [Asset Management & Bundling Pipeline (`assets`)](./modules/assets.md)

### 3. [System, Security & Storage](./system/security-and-auth.md)
Defensive security, configuration parsing, and persistence abstractions:
- [Security Defaults & Authentication (`sec`, `auth`)](./system/security-and-auth.md)
- [Declarative Config & Struct Validation (`config`, `validation`)](./system/config-and-validation.md)
- [Object Storage & Testable Filesystem (`storage`, `fs`, `file`)](./system/storage-and-fs.md)

### 4. [Platform, AI & Utilities](./platform/cloud.md)
Cloud orchestration, AI capabilities, testing infrastructure, and modular utilities:
- [Native AI SDK (`ai`)](./modules/ai.md)
- [Cloud Orchestration & Infra Blueprints (`cloud`)](./platform/cloud.md)
- [Test Harness & Browser Automation (`testing`)](./platform/testing.md)
- [Hub, Resilience, Markdown, Functional, i18n, Term, Boost (`utilities`)](./platform/utilities.md)

---

## Granular Module Reference
For individual package APIs and implementation notes, see the [Module Index](./modules/index.md).
