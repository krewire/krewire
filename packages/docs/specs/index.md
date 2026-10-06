# Specifications Index — Krewire Libraries

This directory holds the formal specifications for the Krewire Libraries.

Ordered by **impact-to-effort** (high impact, low effort first) and **dependency chain** (foundations first).

| SpecID    | Title                                      | Status | Depends On |
| --------- | ------------------------------------------ | ------ | ---------- |
| [KWL-M1ZKS](./KWL-CORE-M1ZKS-krewire-libraries.md) | Krewire Libraries — Initial Specification | Draft | — |
| [KWL-K1N2Q](./KWL-CORE-K1N2Q-core-business-rules.md) | Core Business Rules & Workload Registry | Draft | KWL-M1ZKS, KWL-W0J2X |
| [KWL-ARCH-J2K9Q](./KWL-ARCH-J2K9Q-ecosystem-scope-levels.md) | Ecosystem Scope Levels — Workspace → Module → Domain → Service → Unit | Draft | KWL-K1N2Q |
| [KWL-TEST-P8M4L](./KWL-TEST-P8M4L-spec-driven-testing.md) | Spec-Driven Testing — Requirements → Tests Traceability | Draft | KWL-ARCH-J2K9Q, KWN-P0FWA |
| [KWL-KERN-X8P3L](./KWL-KERN-X8P3L-kernel-executor.md) | Kernel Executor & Supervisor | Draft | KWL-K1N2Q |
| [KWL-W0J2X](./KWL-CORE-W0J2X-errors-exit-codes.md) | Core Errors & Exit Codes | Draft | KWL-M1ZKS |
| [KWL-2X1QZ](./KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading (YAML + env overlay) | Draft | KWL-M1ZKS, KWL-K1N2Q |
| [KWL-LHANF](./KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation (required/min/max/len/email/pattern/oneof) | Draft | KWL-M1ZKS, KWL-K1N2Q |
| [KWL-R934Y](./KWL-TERM-R934Y-terminal-io-rendering.md) | Terminal I/O & Rendering | Draft | KWL-M1ZKS |
| [KWL-K4T7W](./KWL-ENV-K4T7W-environments-and-debug-mode.md) | Environments & Debug Mode | Draft | KWL-M1ZKS, KWL-W0J2X, KWL-2X1QZ |
| [KWL-P8W2N](./KWL-ERR-P8W2N-error-handling-stack-traces-and-logging.md) | Error Handling, Stack Traces & Structured Logging | Draft | KWL-M1ZKS, KWL-W0J2X, KWL-K4T7W |
| [KWL-Q3N8P](./KWL-MARKDOWN-Q3N8P-shared-markdown-renderer.md) | Shared Markdown Renderer (Goldmark) | Draft | KWL-M1ZKS, KWF-PT8OD, KWM-FX9H2 |
| [KWL-O10SC](./KWL-SEC-O10SC-owasp-top-10-security-controls.md) | OWASP Top 10 Security Controls | Draft | KWL-M1ZKS |

## Conventions

- Each specification is stored as a single Markdown file named `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
- SpecIDs are unique, random 5-character alphanumeric codes (e.g., `KWL-M1ZKS`).
- New specifications must be added to this index when created.
- Ordering: impact-to-effort (high impact, low effort first), then dependency chain (foundations first).