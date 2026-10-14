# Krewire Packages — Technical Reference

Comprehensive catalog and dependency guide for domain libraries in `github.com/krewire/krewire`.

---

## Architecture & Layering

All domain packages strictly conform to the downward-only dependency hierarchy described in [Architecture & Layering](../foundation/architecture.md):
```text
tools/kiw ──► packages/* ──► packages/kern ──► Go Standard Library
```

---

## Package Domains

### 1. [Core & Control Plane](./core/kern.md)
The bedrock runtime executor, domain models, and error management:
- [Kernel Supervision (`kern`)](./core/kern.md)
- [Declarative Models & Workload Matrix (`kern/model`)](./core/model.md)
- [Errors, Exit Codes & Logging (`kern/errs`, `kern/env`)](./core/errors-and-env.md)

### 2. [Web & Client Runtime](./web-runtime/web.md)
Fullstack web routing, static site generation, and interactive client WebAssembly:
- [HTTP Router, SSG, SSR & Gateway (`web`)](./web-runtime/web.md)
- [WebAssembly VDOM Engine (`runtime`)](./web-runtime/runtime.md)
- [Web Components & Terminal UI (`ui`, `tui`)](./web-runtime/ui-tui.md)

### 3. [System, Security & Storage](./system/security-and-auth.md)
Defensive security, configuration parsing, and persistence abstractions:
- [Security Defaults & Authentication (`sec`, `auth`)](./system/security-and-auth.md)
- [Declarative Config & Struct Validation (`config`, `validation`)](./system/config-and-validation.md)
- [Object Storage & Testable Filesystem (`storage`, `fs`, `file`)](./system/storage-and-fs.md)

### 4. [Platform & Support Utilities](./platform/cloud.md)
Cloud runtime, testing infrastructure, and modular utilities:
- [Cloud Orchestration & Infra Blueprints (`cloud`)](./platform/cloud.md)
- [Test Harness & Browser Automation (`testing`)](./platform/testing.md)
- [Hub, Resilience, Markdown, Functional, i18n (`utilities`)](./platform/utilities.md)

---

## Granular Module Reference
For individual package APIs and implementation notes, see the [Module Index](./modules/index.md).
