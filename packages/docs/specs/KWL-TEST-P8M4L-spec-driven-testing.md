# Specification — Spec-Driven Testing

| Field  | Value                          |
| ------ | -------------------------------- |
| SpecID | KWL-TEST-P8M4L                   |
| Title  | Spec-Driven Testing — Requirements → Tests Traceability |
| Status | Draft                            |
| Date   | 2026-08-22                       |
| Author | Krewire Contributors              |
| Domain | Libraries — Testing / Guild      |

## 1. Context

Krewire is spec-driven: every initiative starts as a spec in `internal/docs/specs/<project>/` with requirement rows like `FRK-SSG-010` (AGENTS.md Spec-Driven Development). Today, however, the link from requirement to test is informal — tests are named `TestFoo` without a SpecID, CI only checks `go test ./...` pass, and reviewers cannot tell if a spec's `Must` requirements are covered. Failures drift from specs because there is no traceability gate.

This spec formalizes **spec-driven testing**: the discipline that every `Must`/`Should` requirement has one or more named tests, that test names and file locations encode spec and scope, and that `kiw test` (and CI) can report coverage per spec.

It consumes `KWL-ARCH-J2K9Q` scope levels (Workspace → Module → Domain → Service → Unit) to place tests at the correct level, and extends `KWN-P0FWA` (Project Validation) which currently only runs `go test ./...`.

## 2. Problem Statement

- Requirement IDs exist but tests do not reference them; impossible to audit whether a spec is implemented.
- Tests live at arbitrary scopes: a service-level requirement is tested with a package-level unit test or not at all.
- No naming convention: `TestNewCreatesKernel` vs `Test_KWF_XXX` — no grep can find all tests for a spec.
- CI is binary (pass/fail) not per-spec; a missing test for a new requirement is not caught.

## 3. Goals

- G1 — Every `Must` requirement gets at least one test; `Should` may have tests or explicit `// N/A: reason`.
- G2 — Test name and file encode `SpecID` + `RequirementID` + `Scope`, enabling `rg KWL-TEST` to audit.
- G3 — `kiw test` remains `go test ./...` but gains `--spec <SpecID>` filter and per-spec summary (future, not breaking).
- G4 — Zero friction for contributors: `gofmt`/`go vet`/`go test` remain the only required local gates; spec-driven naming is a convention checked by review/lint, not a hard build break on day one.
- G5 — Backward compatible: existing tests without spec tags remain valid; migration is incremental.

## 4. Non-Goals

- NG1 — Not a new test runner; still `go test` with stdlib `testing`.
- NG2 — Not coverage enforcement (e.g., 80%); we track requirement coverage, not line coverage.
- NG3 — Not BDD/Gherkin; Go table-driven tests remain the idiom.
- NG4 — Not rewriting existing test suites in one PR; migrate opportunistically when touching a spec's area.

## 5. Requirements

### 5.1 Requirement → Test Mapping

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-TST-001 | For each spec, create a test file `*_test.go` in the package that implements the spec's scope (e.g., `KWL-ARCH-J2K9Q` scope `Unit` → `libs/core/scope_test.go`). | Must |
| KWL-TST-002 | Each requirement `XXX-999` gets at least one test function whose name contains the requirement ID, e.g. `func TestKWL_SCP_001_ParseScope_Valid(t *testing.T)` or `func Test_SCP_001(t *testing.T)` with a leading comment `// Spec: KWL-ARCH-J2K9Q KWL-SCP-001 Scope: Unit`. Table-driven subtests may cover multiple requirements but top-level function must name the primary. | Must |
| KWL-TST-003 | Test file header must list the SpecID(s) it covers: `// Tests for KWL-ARCH-J2K9Q, KWL-TEST-P8M4L` (one line). `go vet` does not check; `spec-lint` may. | Should |
| KWL-TST-004 | If a `Should` requirement has no test, add `// N/A: <reason>` next to the requirement row in the spec or a `// TODO(KWL-TST):` in code, so audit can distinguish intentional gap from oversight. | Should |
| KWL-TST-005 | Workspace-scope requirements (e.g., `kiw new` workflow) are tested via `internal/commands` integration tests or `scaffold` tests that exercise the CLI, not via unit tests alone. | Must |

### 5.2 Naming & Location Conventions

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-TST-010 | Test function naming: `Test<SpecCode>_<ReqID>_<Scenario>` where `SpecCode` is the last 5-char code (e.g., `J2K9Q`) or full `KWL_ARCH_J2K9Q`, `ReqID` is `SCP_001` etc., `Scenario` is `Valid`/`Invalid`/`Kernel` etc. Existing `TestNewCreatesKernel` is allowed but new tests for new specs must use the spec-tagged form. | Should |
| KWL-TST-011 | Scope-driven placement: Workspace → `kiw/internal/commands` or `guild` tests; Module → `go.mod` root / repo root; Domain → `internal/<domain>/` (e.g. `internal/catalog/`); Service → `service/` or `cmd/<service>/` tests; Unit → same package as code (`package.Func`). | Must |
| KWL-TST-012 | Shared helpers for spec-driven tests live in `*_test.go` in the same package; no `testutil` package unless cross-package reuse is proven. | Should |

### 5.3 Execution & Reporting

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-TST-020 | `kiw test` (KWN-P0FWA) continues to run `go test ./...` and stream output; it must not fail if a spec has no test yet. | Must |
| KWL-TST-021 | Future `kiw test --spec KWL-ARCH-J2K9Q` filters to tests whose name or file header contains that SpecID (via `go test -run`). Not required for this spec's acceptance but the naming in KWL-TST-002 must enable it. | Should |
| KWL-TST-022 | CI (`.github/workflows/ci.yml` in each repo) must run `go test ./...`; a future `spec-coverage` job may parse `rg "KWL-[A-Z]+-[A-Z0-9]+-[A-Z0-9]{5}"` vs `rg "TestKWL"` and report missing coverage as a warning, not a hard fail initially. | Should |
| KWL-TST-023 | Quality gates remain `gofmt -l .`, `go vet ./...`, `go test ./...` per AGENTS.md; spec-driven naming is a review checklist item, not a gate that blocks `go test`. | Must |

### 5.4 Migration & Tooling (Opt-In)

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-TST-030 | Provide `docs/guides/spec-driven-testing.md` (in `internal/docs/guides/`) with a 1-page cheat sheet: how to tag a new test, how to audit `rg KWL-TST`. | Should |
| KWL-TST-031 | `internal/docs/specs/index.md` implementation matrix may add a `Test` column linking SpecID → test file(s) (optional for legacy specs). | Should |
| KWL-TST-032 | Example: migrate one existing spec's tests to the new naming as a reference (e.g., `libs/core` `KWL-CORE-K1N2Q` → `scope_test.go`). | Should |

## 6. Non-Functional Requirements

- NFR1 — No new dependencies; stdlib `testing` only; `libs/core` scope type is the only shared code.
- NFR2 — Backward compatible: existing `TestFoo` tests pass unchanged; new convention is additive.
- NFR3 — `gofmt` clean, `go vet` clean, `go test` green in every repo before merge.
- NFR4 — Docs in English, Markdown, spec-driven.

## 7. Success Criteria

- S1 — A new spec after this one has at least one requirement with a test whose name contains the RequirementID and a file header comment `// Tests for <SpecID>`; `rg <RequirementID>` finds the test.
- S2 — `AGENTS.md` Testing section and `internal/docs/guides/spec-driven-testing.md` describe the naming convention; a contributor can follow the guide without asking.
- S3 — `kiw test` still passes on all repos; no CI workflow is broken by the new convention.
- S4 — One migrated example exists (e.g., `libs/core/scope_test.go` for `KWL-ARCH-J2K9Q` or `guild` test for `KWL-TEST-P8M4L` itself).

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| [KWL-ARCH-J2K9Q](./KWL-ARCH-J2K9Q-ecosystem-scope-levels.md) | Ecosystem Scope Levels (defines Scope used for test placement) |
| [KWL-K1N2Q](./KWL-CORE-K1N2Q-core-business-rules.md) | Core Business Rules (Kind/Project/SpecID types) |
| [KWN-P0FWA](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-TEST-P0FWA-project-validation.md) | Project Validation (`kiw test` → `go test ./...`) |
| [KWN-Z0VFC](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-DEVTOOL-Z0VFC-krewire-devtool.md) | Krewire Devtool (CLI entry) |
| [KWG-K2N7Q](https://github.com/krewire/guild/blob/main/docs/specs/KWG-ECO-K2N7Q-krewire-native-guild-template.md) | Guild Template (agent workflow for spec-driven dev) |

## 9. References

- Go testing: https://pkg.go.dev/testing, https://go.dev/doc/tutorial/add-a-test
- Table-driven tests: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests
- Existing test patterns: `libs/core/kind_test.go`, `krewire/internal/scaffold/scaffold_test.go`
- AGENTS.md Workflow: Understand → Plan → Implement → Verify → Summarize
