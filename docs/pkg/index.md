# Krewire Packages — Technical Reference

Comprehensive catalog and dependency guide for the 37 domain libraries in `github.com/krewire/krewire`.

---

## Architecture & Layering

All domain packages strictly conform to the downward-only dependency hierarchy described in [Architecture & Layering](../foundation/architecture.md):
```text
tools/kiw ──► pkg/* ──► pkg/model, pkg/errs ──► Go Standard Library
```

---

## Package Domains

### 1. [Core & Control Plane](./core/model.md)
The bedrock runtime executor, domain models, application assembly, and error management:
- [Execution & Lifecycle Supervision (`lifecycle`)](./mod/lifecycle.md)
- [Declarative Models & Workload Matrix (`model`)](./mod/model.md)
- [Errors & Diagnostic Exit Codes (`errs`)](./mod/errs.md)
- [Environment Resolution & Structured Logging (`env`, `log`)](./mod/log.md)
- [Ecosystem Versioning & Signature (`version`)](./mod/version.md)
- [Application Assembly & Container (`app`)](./mod/app.md)

### 2. [Web & Client Runtime](./web-runtime/web.md)
Fullstack web routing, component templates, static site generation, and interactive client WebAssembly:
- [HTTP Router, SSG, SSR & Gateway (`web`)](./web-runtime/web.md)
- [The `.kiw` DSL Engine & Templates (`kiw`)](./mod/kiw.md)
- [WebAssembly VDOM Engine (`runtime`)](./web-runtime/runtime.md)
- [Web Components & Design Tokens (`ui`)](./web-runtime/ui-tui.md)
- [Terminal UI Toolkit (`tui`)](./web-runtime/ui-tui.md)
- [Asset Management & Bundling Pipeline (`assets`)](./mod/assets.md)
- [Markdown Content & Book Compiler (`book`)](./mod/book.md)

### 3. [System, Security & Storage](./system/security-and-auth.md)
Defensive security, configuration parsing, and persistence abstractions:
- [Security Defaults & Authentication (`sec`, `auth`)](./system/security-and-auth.md)
- [Declarative Config & Struct Validation (`config`, `validation`)](./system/config-and-validation.md)
- [Object Storage & Testable Filesystem (`storage`, `fs`, `file`)](./system/storage-and-fs.md)

### 4. [Platform, AI & Utilities](./platform/cloud.md)
Cloud orchestration, AI capabilities, testing infrastructure, and modular utilities:
- [Native AI SDK (`ai`)](./mod/ai.md)
- [Cloud Orchestration & Infra Blueprints (`cloud`)](./platform/cloud.md)
- [Test Harness & Browser Automation (`testing`)](./platform/testing.md)
- [Caching, Redis & ORM (`cache`, `redis`, `orm`)](./platform/utilities.md)
- [Mail Delivery & Commerce (`mail`, `commerce`)](./platform/utilities.md)
- [Team Collaboration & Policies (`collab`)](./mod/collab.md)
- [Hub, Resilience, Markdown, Functional, i18n, Term, Boost (`utilities`)](./platform/utilities.md)

---

## Granular Module Reference
For individual package APIs and implementation notes, see the [Module Index](./mod/index.md).
