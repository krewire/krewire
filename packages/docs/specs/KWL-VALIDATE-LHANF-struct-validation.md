# Specification — Struct Validation

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWL-LHANF                                   |
| Title       | Struct Validation                          |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Libraries — Validation                      |

## 1. Context

Krewire requests and configs need validating structured inputs (forms, JSON
bodies, CLI flags, settings). Validation is a generic concern: it inspects a
typed struct, applies declared rules, and reports failures — none of which
depends on `net/http` or the framework. It belongs in Krewire Libraries as a
framework-agnostic dependency so the `web` pipeline (KWF-230KF) and CLI layers
can share one rule model.

Rules are declared inline on struct fields with a `validate` tag; the package
is stdlib-only and never imports `framework`.

## 2. Problem Statement

Today inputs are validated ad hoc: servers hand-check each field, CLI flags
free-text parse, and there is no shared way to describe "this value is
required", "that number is between 1 and 100", or "here is an email". The
result is scattered validation, inconsistent messages, and duplicated rules
between request DTOs and configuration structs. One struct-tag validator
removes that duplication while keeping validation explicit and type-safe.

## 3. Goals

- G1 — Validate a struct using rules declared in `validate` struct tags.
- G2 — Support the common rule set: `required`, `min`, `max`, `len`, `email`, `pattern`, `oneof`, `omitempty`.
- G3 — Return a structured error aggregating all field failures.
- G4 — Stay framework-agnostic and stdlib-only (no `framework` import).
- G5 — Stay explicit: no reflection-driven control flow beyond reading tags.
- G6 — Integrate cleanly with `config` (KWL-2X1QZ) for validated settings.

## 4. Non-Goals

- NG1 — Cross-field or custom (registration-based) validation callbacks in this phase.
- NG2 — Automatic struct branching, `union` types, or compile-time rule checks.
- NG3 — HTTP response formatting (the web layer, KWF-230KF, owns that).
- NG4 — Validation of untyped `map[string]any` shapes.
- NG5 — A separate rule language beyond the struct tag syntax.

## 5. Requirements

### 5.1 Package & Entry Points

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| VAL-AP-001  | Provide package `validation` in `github.com/krewire/libs/validation`.  | Must     |
| VAL-AP-002  | Provide `validation.Struct(v any) error` returning nil when all rules pass and a `*ValidationError` when any rule fails. | Must |
| VAL-AP-003  | Provide `validation.Field(value any, tag string) error` validating a single value, used for scalar checks. | Must |
| VAL-AP-004  | Ignore unexported fields; nil pointers marked `omitempty` are skipped. | Must |

### 5.2 Rules

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| VAL-RL-001  | `required` fails for zero values (`""`, `0`, `false`, nil, empty slice/map). | Must |
| VAL-RL-002  | `min=<n>` / `max=<n>` apply to numbers (inclusive) and, for strings, to byte length. | Must |
| VAL-RL-003  | `len=<n>` constrains slice/map length and string byte length exactly. | Must |
| VAL-RL-004  | `email` matches a basic RFC-5322-shaped address using stdlib `net/mail`. | Must |
| VAL-RL-005  | `pattern=<regexp>` compiles the pattern once per call and matches the full value. | Must |
| VAL-RL-006  | `oneof=a b c` accepts only a listed literal (strings and comparable numbers). | Must |
| VAL-RL-007  | `omitempty` as the last element disables all other rules for zero values. | Must |
| VAL-RL-008  | Multiple rules compose as `validate:"required,min=3"`; the first failing rule reports. | Must |

### 5.3 Errors

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| VAL-ER-001  | `ValidationError` aggregates `FieldError` entries with `Error() string` listing each failure. | Must |
| VAL-ER-002  | `FieldError` carries `Field` (struct path), `Value`, and `Rule`.   | Must |
| VAL-ER-003  | Nested structs validate recursively; field paths join with `.` (e.g. `User.Email`). | Must |
| VAL-ER-004  | A nil pointer to a struct without `omitempty` is a `required` failure. | Must |
| VAL-ER-005  | Unknown or malformed `validate` tag rules return a wrapped error (config/type error), never a panic. | Must |

### 5.4 Org & Testing

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| VAL-OR-001  | Every rule has a table-driven test covering pass, fail, and edge values. | Must |
| VAL-OR-002  | `validate` imports only the Go standard library.                 | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Zero external deps.** Go stdlib only.
- NFR3 — **Performance.** Rule compilation must avoid re-compiling static regexps on hot paths (compile once per call; no global cache in this phase).
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.

## 7. Success Criteria

- S1 — A request DTO using `required`, `min`, `email` rejects bad input with a `ValidationError` listing every field.
- S2 — `validate.Field("a@b.c", "email")` passes while `"nope"` fails.
- S3 — A config struct loaded via KWL-2X1QZ validates in one call before use.
- S4 — `validate` compiles and passes tests with `gopkg.in/yaml.v3` absent from its own dependency graph.

## 8. Related Specifications

| SpecID    | Title                                           |
| --------- | ----------------------------------------------- |
| [KWL-M1ZKS](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-M1ZKS-krewire-libraries.md) | Krewire Libraries — Initial Specification |
| [KWL-2X1QZ](./KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading |
| [KWF-230KF](https://github.com/krewire/framework/blob/main/docs/specs/KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline (consumer) |

## 9. References

- [KWF-M07QS](https://github.com/krewire/framework/blob/main/docs/specs/KWF-WEB-M07QS-krewire-web-framework.md) — Krewire Web Framework.
- Stdlib `reflect`, `regexp`, `net/mail`, `strconv`.