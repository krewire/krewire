# Specification — Core Errors & Exit Codes

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWL-W0J2X                                   |
| Title       | Core Errors & Exit Codes                    |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Libraries / SDK — Core                      |

## 1. Context

The `core` package defines the shared error model for the Krewire ecosystem: a
common `Error` type that pairs a message with a process exit code, plus the
canonical exit-code taxonomy.

This specification formalizes that taxonomy and the rules governing its use.

## 2. Problem Statement

Exit-code conventions differ across Go CLIs — ad-hoc integer codes, ambiguous
meanings, and no shared error type carrying a process status. Automation cannot
trust process termination status, and expected misuse is not distinguished from
runtime failure.

## 3. Goals

- G1 — Define a compact, industry-aligned exit-code taxonomy (0/1/2).
- G2 — Provide an error model that carries a stable exit code end-to-end.
- G3 — Ensure consistent semantics across the framework, libraries, and generated applications.

## 4. Non-Goals

The following are explicitly out of scope for the initial phase:

- NG1 — Extending the taxonomy beyond `0`/`1`/`2` (deferred until a concrete need exists).
- NG2 — A logging subsystem; diagnostics formatting lives in the framework.
- NG3 — Error wrapping/context chains beyond the basic message + code model.

## 5. Requirements

### 5.1 Exit-Code Taxonomy

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| KWL-ER-001 | Define `0` as success.                                            | Must     |
| KWL-ER-002 | Define `1` as a generic runtime failure.                          | Must     |
| KWL-ER-003 | Define `2` as usage error (bad or missing arguments/configuration). | Must   |
| KWL-ER-004 | Provide a mapping from a raw process code to the canonical value.  | Must     |

### 5.2 Error Model

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| KWL-ER-005 | Provide an error type holding a message and an exit code.         | Must     |
| KWL-ER-006 | The error type must implement the `error` interface.              | Must     |
| KWL-ER-007 | Provide constructors for usage and failure errors.                | Must     |
| KWL-ER-008 | Expose the associated exit code for process-level mapping.        | Must     |

### 5.3 Usage Rules

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| KWL-ER-009 | Exit code `2` is reserved for invalid user input, never for runtime failures | Must |
| KWL-ER-010 | Exit code `1` must not be used to signal expected CLI mis-usage.  | Must     |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Friction-free.** The error model must be trivially constructible in handlers.
- NFR3 — **Portability.** Exit-code values are numeric and OS-agnostic.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — The taxonomy mapping and error constructors are covered by tests.
- S2 — All framework CLI commands return only codes from the taxonomy.
- S3 — Documentation comments cover every exported identifier.

## 8. References

- [KWL-M1ZKS](./KWL-CORE-M1ZKS-krewire-libraries.md) — Krewire Libraries initial specification.
- [KWL-R934Y](./KWL-TERM-R934Y-terminal-io-rendering.md) — Terminal I/O & Rendering.
- [KWF-KAKQL](https://github.com/krewire/framework/blob/main/docs/specs/KWF-CLI-KAKQL-cli-errors-diagnostics.md) — CLI Errors & Diagnostics.
- [KWF-5XJFC](https://github.com/krewire/framework/blob/main/docs/specs/KWF-CLI-5XJFC-cli-application-model.md) — CLI Application Model.