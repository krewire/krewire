# Specification Template — Global Industry Standard (Krewire Adaptation)

> Copy this template to `docs/specs/` as `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
> SpecID is a unique random 5-character alphanumeric code (e.g., `K1N2Q`). Never reuse or sequence.
> See `docs/specs/index.md` for the implementation matrix and `docs/project-vision.md` for workload context.

File name: `{ProjectId}-{Scope}-{SpecID}-{slug}.md`

| Field    | Value |
|----------|-------|
| SpecID   | `KWL-K1N2Q` — `{ProjectId}-{Scope}-{Code}` (e.g., `KWL-CORE-K1N2Q`) |
| Title    | Concise, outcome-focused title (imperative, searchable) |
| Status   | `Draft` → `Review` → `Approved` → `Implemented` → `Deprecated` |
| Date     | `YYYY-MM-DD` (creation date) |
| Author   | `Name <email>` or `Krewire Contributors` |
| Domain   | `<Repo> — <Area>` (e.g., `Libraries — Core`, `Framework — Runtime`) |
| Reviewers| `Name` (required for global standard — at least 1 peer) |
| Deciders | `Name` (required for ADR-style approval — who decides) |
| Stakeholders | `User` / `Operator` / `AI Agent` (who is impacted) |

> **Krewire convention:** No `Version`, `Last updated`, or `Changes` fields. Revision history lives in `git log -- <path>` (`git log -1 -- <path>` for latest). For global standard, `Reviewers`/`Deciders`/`Stakeholders` are required for reviewability.

> **RFC 2119:** Use **MUST** / **MUST NOT** / **SHOULD** / **SHOULD NOT** / **MAY** for normative requirements.

---

## 1. Summary

One-paragraph executive summary: what this spec proposes, why it matters now, and the intended outcome. A reviewer should understand the decision without reading further. Reference the unified vision (`KWF-M8K2Q`) and workload matrix where relevant.

## 2. Background & Context

What exists today and why change is needed. Ground in reality:

- Current architecture, files, and constraints (cite `path/file.go:line` or commit)
- Past decisions and their consequences
- Link to `docs/project-vision.md` and related specs (`KWL-*`/`KWF-*`)

## 3. Problem Statement

Concrete, verifiable problem this spec solves. Use measurable language.

- Who is affected (user, developer, operator)?
- What is the pain (repro steps, file:line, error, or metric)?
- What happens if not solved?

## 4. Goals & Non-Goals

### Goals

- G1 — Specific, testable outcome (use **MUST** language per RFC 2119)
- G2 — ...

### Non-Goals

- NG1 — Explicitly out of scope and why (prevents scope creep)

> One spec per initiative — scope creep is a sign to split.

### 4.5 Assumptions & Constraints

| ID | Assumption / Constraint | Type | Validation |
|----|-------------------------|------|------------|
| A1 | ... (e.g., Go 1.22+, no `go.work`, `krewire.yaml`-only) | Assumption | ... |
| C1 | ... (e.g., `libs/core` stdlib-only, no `framework` import) | Constraint | `gofmt` + `go vet` |

### 4.6 Glossary

| Term | Definition | Source |
|------|------------|--------|
| Workload | One cell of the 9-workload matrix (cli, worker, infra, etc.) | `docs/project-vision.md` |
| SpecID | Random 5-char code in `{ProjectId}-{Scope}-{SpecID}-{slug}.md` | `docs/specs/index.md` |
| Control Plane | `libs/core` (declarative) + `libs/kern` (imperative) | `KWL-K1N2Q` / `KWL-KERN-X8P3L` |

## 5. Requirements

### 5.1 Functional Requirements

| ID          | Requirement | Priority | RFC 2119 |
|-------------|-------------|----------|----------|
| `KWL-CORE-001` | The system **MUST** validate `Kind` via `core.ParseKind` and return `core.ExitCodeUsage` on unknown | Must | MUST |
| `KWL-CORE-002` | The system **SHOULD** ... | Should | SHOULD |

- Each row traces `spec → implementation → test` (`FRK-*`/`KWL-*`/`KWM-*` etc.)
- Priority: `Must` (blocking) / `Should` (important) / `May` (optional)
- Use RFC 2119 keywords: **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, **MAY**

### 5.2 Non-Functional Requirements

| ID   | Category | Requirement |
|------|----------|-------------|
| NFR1 | Performance | ... (e.g., `libs/core` stays stdlib-only; no `framework` import) |
| NFR2 | Quality Gates | `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` passes |
| NFR3 | Compatibility | Backward compatible with `KWL-W0J2X` `ExitCode`/`Error` API |

## 6. Detailed Design / Proposal

### 6.1 Architecture

- Module structure (tree or diagram), dependency direction, and control-plane placement (`libs/core` declarative + `libs/kern` imperative)
- Data model and API contracts (idiomatic Go: `(value, error)`, functional options, zero-value usable, `go doc` comments)

### 6.2 API Design

```go
// Example — keep signatures minimal and verifiable
func ParseKind(s string) (Kind, error)
type Workload struct { Kind Kind; Package string; SpecID SpecID }
```

### 6.3 Alternatives Considered (with Trade-off Matrix)

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| A — ... | ... | ... | ... |
| B — ... | ... | ... | ... |

Decision uses impact-to-effort ordering (see `agent-workflow` skill → `rules/impact-to-effort.md`).

### 6.4 System Context & Diagrams

```
[Diagram: module dependency, workload matrix, or sequence. Use ASCII or Mermaid.]
libs/core (Kind/Workload) ← libs/kern (Kernel/Module) ← framework/* ← krewire
```

### 6.5 Cost & Performance

| Aspect | Estimate | Notes |
|--------|----------|-------|
| Dev cost | ... (XS/S/M/L) | ... |
| Runtime cost | ... (binary size, memory) | e.g., `libs/core` zero-cost when unused |
| Performance | ... (latency, throughput) | e.g., `core.ParseKind` O(1) |

### 6.6 Security, Privacy & Compliance

- Secrets: `secret.Ref` not literal, `core.ValidateKrewireYamlPath` enforces `krewire.yaml` only
- Auth: ... | Privacy: ... | Compliance: IEEE 830, semver per `core.Version`

### 6.7 Accessibility & Internationalization

- Accessibility: ... (e.g., `framework/ui` a11y, `tui` output)
- i18n: Language in specs is English per `AGENTS.md`; human may write Indonesian but agent replies English

### 6.8 Observability

- Logs: `log/slog` structured, `core` business rules emit `ExitCodeUsage` on violation
- Metrics: ... | Traces: OTel `service/tracing` if `service` workload

## 7. Dependencies & Impact

- **Depends On:** Specs that must land first (see `docs/specs/index.md` Depends On column)
- **Impacts:** Repos, packages, and docs that this spec touches (e.g., `libs/core`, `libs/kern`, `AGENTS.md`, `docs/project-vision.md`)
- **Migration:** Steps to adopt, backward-compatibility, or `MOVED.md` redirects if moving files

## 8. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Circular dependency if `core` imports `framework` | High | `core` stays stdlib-only; `kern` depends only on `core` |
| SpecID collision | Medium | Random 5-char, check `docs/specs/index.md` |

## 9. Testing & Verification Plan

- **Unit:** `go test ./...` in owning repo (e.g., `libs/core` — `TestParseKind`, `TestSpecID`)
- **Integration:** Cross-repo `replace` in `go.mod` for `framework`/`krewire` consuming `libs`
- **Per-kind gates:** `go build .` for `app`/`cli`/`worker`/`service`, `krewire build` for `site`/`book`, `krewire build --plan` for `infra`
- **Spec traceability:** Each `Must` row has a test file:line reference

## 10. Rollout & Operations

- **Phase:** Which roadmap phase (see `docs/project-vision.md` Phases 0-5) — sorted by impact-to-effort then dependency
- **Rollout steps:** Spec → implementation → `gofmt`/`go vet`/`go test` → `push`+`tag` → propagate `go.mod` downstream → rebuild `bin/krewire` → `docs/specs/index.md` Impl Status `Planned` → `Shipped`
- **Timeline:** `Week 1: spec review` → `Week 2: implementation` → `Week 3: gates + docs sync` (or `XS`/`S`/`M` estimate)
- **Monitoring:** Logs, metrics, or health checks if runtime behavior changes
- **Rollback:** How to revert (`git revert`, feature flag)
- **Decision Log:** Record key decisions with date and decider (ADR-style) — e.g., `2026-08-21: chose libs/core for Kind registry (decider: @author)`

## 11. Open Questions

| # | Question | Owner | Resolution |
|---|----------|-------|------------|
| 1 | ... | ... | ... |

## 12. Success Criteria

- S1 — Verifiable outcome (e.g., `go doc github.com/krewire/libs/core` lists `Kind` with examples)
- S2 — Downstream `framework`/`krewire` can import new API without `go vet` failures
- S3 — `docs/specs/index.md` updated with correct `Impl Status`

## 13. Related Specifications

| SpecID | Title | Relationship |
|--------|-------|--------------|
| `KWL-W0J2X` | Errors & Exit Codes | Extends |
| `KWF-M8K2Q` | Unified Framework Vision | Source |

Cross-repo links use `../<project>/` or full GitHub URL to `krewire/internal` main branch.

## 14. References

- `docs/project-vision.md` — unified workload matrix
- `docs/specs/index.md` — implementation matrix (Spec vs Impl Status)
- Code: `libs/core/core.go`, `libs/kern/kern.go`
- Version: `libs/core/version.go` (`core.Version`, `core.CheckEcosystemCompatibility`)
- External: RFC 2119 (MUST/SHOULD), IEEE 830 / IEEE 1016, ISO/IEC 25010, ADR template, Google Design Doc format, semver.org

## 15. AI Agent Instructions

> For the AI agent that will implement this spec. A human reviewer can skip this section.

```yaml
context:
  - load: docs/specs/index.md       # check Depends On, Spec vs Impl
  - load: specific KWF-*/KWL-* spec for this workload only (lazy)
routing:
  - workload == "core" → skill: arch-guard, agent: reviewer
  - workload == "kern" → skill: agent-workflow, agent: service
  - workload == "tui" → skill: wasm-runtime (if frontend) or tui
  - multi-workload → agent: orchestrator → build
handoff:
  Task: "<Title> — <SpecID>"
  Kind: "<app|cli|site|book|worker|service|infra|kernel>"
  Spec: "<SpecID>"
  Files: "<touched paths>"
  Gates: "gofmt -l ., go vet ./..., go test ./..., per-kind gate, arch-guard, sync-docs"
gates: ["gofmt -l .", "go vet ./...", "go test ./...", "arch-guard Pass", "sync-docs In-sync"]
```

---

### Checklist Before Submitting

- [ ] SpecID is random 5-char, unique, not sequenced; file name follows `{ProjectId}-{Scope}-{SpecID}-{slug}.md`
- [ ] Metadata table complete, no `Version` field; `Date` is creation date
- [ ] Every `Must` requirement has an ID and will trace to implementation + test
- [ ] `docs/specs/index.md` and per-project `index.md` updated
- [ ] Alternatives and Risks sections filled (not left empty)
- [ ] Verification plan lists real commands (`gofmt`, `go vet`, `go test`, per-kind gate)
