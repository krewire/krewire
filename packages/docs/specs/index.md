# Specifications Index — Krewire Libraries

This directory holds the formal specifications for the Krewire Libraries.

Ordered by **impact-to-effort** (high impact, low effort first) and **dependency chain** (foundations first).

| SpecID    | Title                                      | Status | Depends On |
| --------- | ------------------------------------------ | ------ | ---------- |
| [KWL-M1ZKS](https://github.com/krewire/krewire/issues/7) | Krewire Libraries — Initial Specification | Draft | — |
| [KWL-K1N2Q](https://github.com/krewire/krewire/issues/6) | Core Business Rules & Workload Registry | Draft | KWL-M1ZKS, KWL-W0J2X |
| [KWL-ARCH-J2K9Q](https://github.com/krewire/krewire/issues/4) | Ecosystem Scope Levels — Workspace → Module → Domain → Service → Unit | Draft | KWL-K1N2Q |
| [KWL-TEST-P8M4L](https://github.com/krewire/krewire/issues/15) | Spec-Driven Testing — Requirements → Tests Traceability | Draft | KWL-ARCH-J2K9Q, KWN-P0FWA |
| [KWL-KERN-X8P3L](https://github.com/krewire/krewire/issues/11) | Kernel Executor & Supervisor | Draft | KWL-K1N2Q |
| [KWL-W0J2X](https://github.com/krewire/krewire/issues/8) | Core Errors & Exit Codes | Draft | KWL-M1ZKS |
| [KWL-2X1QZ](https://github.com/krewire/krewire/issues/5) | Configuration Loading (YAML + env overlay) | Draft | KWL-M1ZKS, KWL-K1N2Q |
| [KWL-LHANF](https://github.com/krewire/krewire/issues/16) | Struct Validation (required/min/max/len/email/pattern/oneof) | Draft | KWL-M1ZKS, KWL-K1N2Q |
| [KWL-R934Y](https://github.com/krewire/krewire/issues/14) | Terminal I/O & Rendering | Draft | KWL-M1ZKS |
| [KWL-K4T7W](https://github.com/krewire/krewire/issues/9) | Environments & Debug Mode | Draft | KWL-M1ZKS, KWL-W0J2X, KWL-2X1QZ |
| [KWL-P8W2N](https://github.com/krewire/krewire/issues/10) | Error Handling, Stack Traces & Structured Logging | Draft | KWL-M1ZKS, KWL-W0J2X, KWL-K4T7W |
| [KWL-Q3N8P](https://github.com/krewire/krewire/issues/12) | Shared Markdown Renderer (Goldmark) | Draft | KWL-M1ZKS, KWF-PT8OD, KWM-FX9H2 |
| [KWL-O10SC](https://github.com/krewire/krewire/issues/13) | OWASP Top 10 Security Controls | Implemented | KWL-M1ZKS |
| [KWL-N8K4V](https://github.com/krewire/krewire/issues/35) | Native Zero-Dependency Markdown Parser | Planned | KWL-M1ZKS |

## Conventions

- Each specification is stored as a single Markdown file named `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
- SpecIDs are unique, random 5-character alphanumeric codes (e.g., `KWL-M1ZKS`).
- New specifications must be added to this index when created.
- Ordering: impact-to-effort (high impact, low effort first), then dependency chain (foundations first).