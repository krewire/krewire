# GitHub Issue Template — Comprehensive (Krewire Global Standard)

> Use this template for every GitHub issue. It is optimized for both human developers and AI agents: a human can triage in 30 seconds, an AI can implement without follow-up questions.
> File location for reference: `.agents/skills/issue-writing/rules/issue-template.md` (this file). Copy the markdown below when creating an issue via `gh issue create`.

---

## Frontmatter (for GitHub Issue Forms — optional YAML)

```yaml
name: Krewire Task
about: One initiative — feature, bug, or architecture guard — for the unified 9-workload framework
title: "<type>(<scope>): <imperative summary>"
labels: ["kind/<workload>", "area/<repo>", "priority/P0-P3"]
assignees: []
```

---

## 1. Title

```
<type>(<scope>): <imperative summary>
```

- `type`: `feat` | `fix` | `docs` | `refactor` | `spec` | `chore` | `guard` (arch-guard)
- `scope`: `core` | `kern` | `tui` | `web` | `runtime` | `worker` | `service` | `infra` | `libs` | `framework` | `krewire` | `guild` | `internal`
- Example: `feat(core): add Kind.Workload registry per KWL-K1N2Q`

Title is searchable and follows Conventional Commits. No period at end.

## 2. Metadata

| Field | Value |
|-------|-------|
| **Type** | `Feature` | `Bug` | `Task` | `Spec` | `Architecture Guard` |
| **Workload & Kind** | `framework/tui` (`cli` kind) / `framework/runtime` (`site`/`app`) / `framework/worker` / `framework/service` / `framework/infra` / `libs/core` / `libs/kern` / `krewire` / `guild` / `docs` / `internal` |
| **Priority** | `P0` (blocker) / `P1` (high) / `P2` (medium) / `P3` (low) — sort by impact-to-effort first, then dependency chain |
| **Spec** | `docs/specs/` link or `SpecID` (e.g., `KWL-K1N2Q`) — `N/A` if bug without spec (but spec-first is preferred) |
| **Size** | `XS` (<1h) / `S` (half-day) / `M` (1-2d) / `L` (week) / `XL` (split) |
| **Reporter** | `@username` + AI agent used (e.g., `arch-guard`, `sync-docs`) |

## 3. Context

Why this matters now. One paragraph + links.

- Link `docs/project-vision.md` workload matrix and roadmap phase (0-5)
- Link relevant spec `file:line` (e.g., `docs/specs/libs/KWL-K1N2Q-core-business-rules.md:42`)
- For `arch-guard` / `sync-docs` findings: cite the guard check that failed

## 4. Problem Statement

Concrete, verifiable, file:line-grounded.

### For Features / Specs
- Who is affected (CLI user, `app` developer, `infra` operator, AI agent)?
- What is blocked or missing (cite `core.Workloads` or `docs/specs/index.md` Impl Status `🔜 Planned`)?
- What happens if not solved (debt, drift, blocked workload)?

### For Bugs
- **Expected:** ...
- **Actual:** ...
- **Repro:** `go test ./... -run TestX` or `krewire build --plan` steps, with file:line
- **Impact:** `core.ExitCode` or user-visible symptom
- **Environment:** `go version`, `krewire info`, branch

## 5. Proposal / Solution

Ordered, small, verifiable steps. Each step is a file or command.

```markdown
1. Write spec `docs/specs/KWL-XXX-*.md` with requirement rows `KWL-XXX-001` (spec-first)
2. Implement `libs/core/workload.go` — add `KindX` + `WorkloadFor` (tdd: `TestKindX`)
3. Update `framework/docs/architecture.md` tree and `README.md` workload matrix
```

- List files to create / change / delete
- State dependency order upfront (foundations before dependents)
- Note if `arch-guard` / `sync-docs` must pass after

### Alternatives Considered (for `spec` / `guard` types)

| Alternative | Why rejected |
|-------------|--------------|
| A — ... | ... |

## 6. Acceptance Criteria (Binary, Verifiable)

Every criterion is a checkbox that is either pass or fail. Must include gates.

- [ ] Requirement row(s) added in spec (`FRK-*`/`KWL-*`) and traced to implementation + test `file:line`
- [ ] `gofmt -l .` empty
- [ ] `go vet ./...` clean in owning repo
- [ ] `go test ./...` passes (or `krewire test`) — list specific test file:line
- [ ] Per-kind gate: `go build .` for `app`/`cli`/`worker`/`service`, `krewire build` for `site`/`book`, `krewire build --plan` for `infra`, hydration check for `runtime`
- [ ] Docs updated: `README.md`, `docs/architecture.md`, `docs/philosophy.md`, `docs/project-vision.md` if public behavior changes
- [ ] `arch-guard` reports `Pass` (no `libs/core` importing `framework`, no `ssg.yaml`, etc.)
- [ ] `sync-docs` reports `In-sync` (workload matrix, `framework/tui` not `framework/cli`)
- [ ] No `go.work`, no committed `replace` directives, no secrets committed

## 7. Testing Plan

- **Unit:** `go test ./... -run Test<Feature>` in `libs`/`framework`/`krewire`
- **Integration:** Cross-repo `replace` for `framework` → `libs` consuming new `core` API
- **Per-kind:** Which `krewire` command verifies it (`build`/`run`/`deploy --plan`)
- **Spec traceability:** Each `Must` row maps to a test `file:line`

## 8. Rollout & Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| ... | High/Med/Low | ... |

- **Rollout:** Spec → implementation → `gofmt`/`go vet`/`go test` → `push`+`tag` → propagate `go.mod` downstream → rebuild `bin/krewire`
- **Rollback:** `git revert` or feature flag

## 9. Dependencies & Impact

- **Depends On:** Specs that must land first (see `docs/specs/index.md` Depends On)
- **Impacts:** Repos, packages, docs touched
- **Migration:** Steps for adopters, `MOVED.md` redirects if moving files

## 10. References

- Spec: `docs/specs/...`
- Vision: `docs/project-vision.md`
- Code: `libs/core/core.go:line`, `libs/kern/kern.go:line`
- Related issue/PR: `#...`
- External: RFC 2119, semver, etc.

## 11. AI Agent Instructions

> This section is for the AI agent that will implement the issue. A human can skip it.

```yaml
context:
  - load: docs/specs/index.md       # check Spec vs Impl Status
  - load: specific KWF-*/KWL-* spec for this workload only (lazy)
routing:
  - workload == "runtime" → agent: runtime, skill: wasm-runtime
  - workload == "infra" → agent: infra, skill: infra-provision
  - workload == "service" → agent: service, skill: service-mesh
  - workload == "worker" → agent: worker, skill: worker-queue
  - multi-workload → agent: orchestrator → build
handoff:
  Task: "<title>"
  Kind: "<app|cli|site|book|worker|service|infra|kernel>"
  Spec: "<SpecID or N/A>"
  Files: "<touched paths>"
  Gates: "gofmt, go vet, go test, per-kind gate"
gates: ["gofmt -l .", "go vet ./...", "go test ./...", "per-kind gate"]
```

## 12. Labels (Suggested)

`kind/<workload>` (e.g., `kind/core`, `kind/runtime`), `area/<repo>` (e.g., `area/libs`, `area/framework`), `priority/P0-P3`, `spec/<SpecID>` if applicable, `guard/arch` or `guard/sync` for guard findings

---

### Checklist Before Submitting the Issue

- [ ] Title is `type(scope): imperative summary` and searchable
- [ ] Metadata table complete (Type, Workload, Priority, Spec, Size)
- [ ] Context links vision + spec `file:line`, not invented architecture
- [ ] Problem is file:line-grounded, not vague
- [ ] Proposal steps are ordered by impact-to-effort then dependency, and state order upfront
- [ ] Acceptance criteria include real gates (`gofmt`, `go vet`, `go test`, per-kind) and `arch-guard`/`sync-docs` pass
- [ ] AI Agent Instructions filled for autonomous implementation
- [ ] Suggested `gh` command ready

### Suggested `gh` Command

```bash
gh issue create --repo krewire/<repo> \
  --title "feat(core): add Kind.Workload registry per KWL-K1N2Q" \
  --label "kind/core,area/libs,priority/P1,spec/KWL-K1N2Q" \
  --body-file - <<'EOF'
<paste markdown above>
EOF
```
