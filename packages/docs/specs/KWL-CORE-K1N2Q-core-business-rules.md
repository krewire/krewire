# Specification — Core Business Rules & Workload Registry

| Field  | Value                          |
|--------|--------------------------------|
| SpecID | KWL-K1N2Q                      |
| Title  | Core Business Rules & Workload Registry |
| Status | Draft                          |
| Date   | 2026-08-21                     |
| Author | Krewire Contributors            |
| Domain | Libraries — Core               |

## 1. Context

`libs/core` currently exposes only process primitives (`ExitCode`, `Error` — `KWL-W0J2X`). Business rules that define the ecosystem — the 8 `project.kind` values, the 9-workload matrix, `SpecID`/`RequirementID` formats, and invariants (*one `krewire.yaml` only*, *spec-first*, *opt-in batteries*, *no `go.work`*) — are scattered across `AGENTS.md`, `krewire/internal/shape`, and `KWF-M8K2Q`. Without a single source, validation is duplicated and drift is possible. The unified vision needs `libs/core` to be the declarative control plane: the domain layer every other package imports for types and validation.

## 2. Problem Statement

- Kind/workload definitions live in multiple places; adding `worker`/`service`/`infra` required touching 10+ files.
- SpecID and requirement ID formats are documented but not enforced as types.
- Invariants are enforced ad-hoc in `krewire` and `framework`, not as reusable business rules.
- `libs/config` and `libs/validation` cannot delegate workload-aware validation to a shared authority.

## 3. Goals

- G1 — `libs/core` becomes the single authority for `Kind`, `Workload`, `SpecID`, `RequirementID`, `Project`, and domain events.
- G2 — 100% backward compatible: existing `ExitCode`/`Error` API unchanged; new types are additive.
- G3 — Pure domain layer: `core` depends only on the Go standard library and sibling `github.com/krewire/libs/*` packages; it never imports `framework` or `kiw` (KWL-CORE-050).
- G4 — All business validation is reusable via `core` so `libs/config`, `libs/validation`, `framework`, and `krewire` converge.

## 4. Non-Goals

- NG1 — Not an executor or lifecycle manager; that is `libs/kern` (`KWL-KERN-X8P3L`).
- NG2 — Not re-implementing `libs/config` (YAML loading) or `libs/validation` (struct validation); `core` provides types and business predicates those packages call.
- NG3 — No I/O, no filesystem, no `os.Getenv` inside `core`; side effects stay in `kern`/`krewire`.

## 5. Requirements

### 5.1 Kind & Workload Registry

The 8 kinds from the unified vision (`internal/docs/project-vision.md`, `KWF-M8K2Q`):

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-001 | `type Kind string` with constants `KindApp`, `KindCLI`, `KindSite`, `KindBook`, `KindWorker`, `KindService`, `KindInfra`, `KindKernel`; `func (Kind) IsValid() bool`; `func ParseKind(string) (Kind, error)` returning `UsageError` on unknown. | Must |
| KWL-CORE-002 | `type Workload struct { Kind Kind; Package string; Title string; SpecID string; Status Status }` (JSON-tagged) and `var Workloads []Workload` covering 9 workloads — the **canonical registry**; the readable table in `internal/docs/project-vision.md` derives from it — plus `func WorkloadFor(Kind) (Workload, bool)`. | Must |
| KWL-CORE-003 | `type Status string` with `StatusShipped`, `StatusPlanned`; workload statuses sourced from `KWF-M8K2Q` and kept in sync. | Must |

### 5.2 Spec & Requirement IDs

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-010 | `type SpecID string` with format `{ProjectId}-{Scope}-{5-char}` (slug suffix optional); `func ParseSpecID(string) (SpecID, error)` validates `ProjectId` in the documented prefix set `{KWF,KWN,KWL,KWM,KWG,KRW}` (framework, kiw, libs, mdbind, boost, sites), `Scope` as `[A-Z0-9]+`, and a 5-char alphanumeric code. | Must |
| KWL-CORE-011 | `type RequirementID string` with format `FRK-*`/`KWL-*`/`KWM-*` etc. + `func ParseRequirementID(string) error`. | Should |
| KWL-CORE-012 | Helpers `SpecID.Project()`, `.Scope()`, `.Code()` for indexing. | Should |

### 5.3 Project & Invariants

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-020 | `type Project struct { Name, ModulePath string; Kind Kind; ConfigPath string }` with `func (Project) Validate() error` enforcing: name kebab-case (`^[a-z][a-z0-9-]*$`), module path is import path, kind is valid. | Must |
| KWL-CORE-021 | `func ValidateKrewireYamlPath(path string) error` ensures config path is `krewire.yaml` (no `ssg.yaml`) — encodes invariant *one config file only*. | Must |
| KWL-CORE-022 | Predicate `HasOptInViolation(kind Kind, imported []string) bool` detects opt-in cost violations: a `KindApp` monolith importing the `service` or `infra` battery (matched on a path-segment boundary). `framework/worker` is deliberately allowed in-process, and a kind's own battery is never a violation. The blocked battery import paths are derived from `Workloads` so the list has one owner. | Should |

### 5.4 Domain Events & Exit Integration

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-030 | `type DomainEvent struct { Type string; Payload any; At time.Time }` for cross-module communication (e.g., `worker.job.enqueued`). | Should |
| KWL-CORE-031 | `Error` and `ExitCode` remain the canonical process primitives; new types use them (e.g., `ParseKind` returns `*Error` with `ExitCodeUsage`). | Must |

### 5.5 Version & Compatibility

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-040 | `type Version struct { Major, Minor, Patch int; PreRelease, Build string }`; `func ParseVersion(string) (Version, error)` accepts `v?MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]`; `func MustParseVersion(string) Version` panics on invalid input (for package-level constants only); `func (Version) String() string` emits the canonical form without a leading `v`. | Must |
| KWL-CORE-041 | `func (Version) Compare(Version) int` implements semver.org §11 precedence: major/minor/patch compared numerically; a version with a prerelease has **lower** precedence than the same version without one; prerelease identifiers are compared **identifier by identifier** — numeric identifiers numerically and always lower than alphanumeric identifiers, otherwise ASCII order — and a longer identifier list wins when all shared identifiers are equal; build metadata is ignored. | Must |
| KWL-CORE-042 | `func (Version) IsCompatible(required Version) bool` applies caret semantics for the ecosystem: for `0.y.z` the minor must match and the version must be `>=` required; for `>=1.0.0` the major must match and the version must be `>=` required. | Must |
| KWL-CORE-043 | `type ModuleName string` with `var EcosystemVersions map[ModuleName]Version` (the blessed release matrix) and `func CheckEcosystemCompatibility(required, actual map[ModuleName]Version) error` (fail-fast) plus `func CheckCompatibility(actual map[ModuleName]Version, reqs map[ModuleName]map[ModuleName]Version) []error` (aggregates every violation). | Should |
| KWL-CORE-044 | Every package-level version literal (`CurrentVersion` and every `EcosystemVersions` entry) is guaranteed parseable by a test, so the init-time `MustParseVersion` panic can never fire in a released build. | Should |

### 5.6 Dependency Invariant

| ID          | Requirement | Priority |
|-------------|-------------|----------|
| KWL-CORE-050 | `libs/core` may import **only** the Go standard library and `github.com/krewire/libs/*`; it must never import `github.com/krewire/framework/*`, `github.com/krewire/kiw/*`, or any third-party module. Enforced by a test over the package's own source imports. | Must |
| KWL-CORE-051 | The deprecated compatibility shims (`core.go`, `diag.go`, `env.go`, `stack.go`) re-export `github.com/krewire/libs/vein` for backward compatibility; new code imports `vein` directly. Removing them is a breaking, ecosystem-wide change tracked by an ADR — never a silent edit. | Should |

`// N/A: KWL-CORE-051 is documentation-only — the shims themselves stay covered by KWL-P8W2N (diag_test.go, stack_test.go) and KWL-K4T7W (env_test.go).`

## 6. Non-Functional Requirements

- NFR1 — Zero breaking changes; `go vet` and `go test ./...` pass in `libs` and downstream `framework`/`krewire` unchanged.
- NFR2 — `core` stays standard-library + intra-module (`libs/*`) only; it never imports `framework`, `kiw`, or third-party modules (KWL-CORE-050).
- NFR3 — 100% `gofmt` clean, idiomatic Go, `go doc` comments on exported types.

## 7. Success Criteria

- S1 — `go doc github.com/krewire/libs/core` lists `Kind`, `Workload`, `SpecID`, `Project` with examples; `libs/core` tests cover `ParseKind`, `ParseSpecID`, `Project.Validate` including error cases returning `ExitCodeUsage`.
- S2 — `libs/config` can delegate `Kind` validation to `core.ParseKind` (no duplicated logic).
- S3 — `krewire/internal/shape` can be refactored to use `core.Kind` without behavior change (follow-up, not in this spec).

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| [KWL-W0J2X](./KWL-CORE-W0J2X-errors-exit-codes.md) | Errors & Exit Codes (extends) |
| [KWL-2X1QZ](./KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading (consumer) |
| [KWL-LHANF](./KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation (consumer) |
| [KWF-M8K2Q](https://github.com/krewire/framework/blob/main/docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified Vision (source of workload matrix) |
| [KWL-KERN-X8P3L](./KWL-KERN-X8P3L-kernel-executor.md) | Kernel Executor (imperative counterpart) |

## 9. References

- `internal/docs/project-vision.md` — 9-workload matrix
- `AGENTS.md` — 8-kind detection table
