---
name: issue-writing
description: Write precise, actionable GitHub issues from specs, bugs, and arch-guard/sync-docs findings. Triggers: "issue", "github issue", "triage", "bug report", "feature request".
---

# Issue Writing — GitHub Issues That Unblock

Turn vague requests into issues that can be picked up async without follow-up questions.

## 1. When to create an issue

- New feature needs a spec (`KWL-*`/`KWF-*`) before code
- `arch-guard` finds a blocker (boundary violation, missing spec)
- `sync-docs` finds drift (README vs vision)
- Bug with file:line repro
- `spec-writing` proposes a new initiative

One issue per initiative — same rule as one spec per initiative.

## 2. Template (global standard)

```markdown
## Title
<type>(<scope>): <imperative summary>

## Type
Feature | Bug | Task | Spec | Architecture Guard

## Workload & Kind
framework/tui | framework/web | framework/tui | framework/runtime | framework/worker | framework/service | framework/infra | libs/core | libs/kern | krewire | guild | docs | internal

## Context
Why now. Link `docs/project-vision.md` and spec file:line (e.g., `docs/specs/libs/KWL-K1N2Q-core-business-rules.md:42`).

## Problem Statement
Concrete problem with file:line or repro. For bugs: expected vs actual, `core.ExitCode` if relevant. For arch-guard: violation + impact.

## Proposal
Ordered small verifiable steps. For features: list files to create/change. For specs: name the new `SpecID` and `index.md` placement.

## Acceptance Criteria
- [ ] Requirement row added in spec (`FRK-*`/`KWL-*`) and traced to implementation + test
- [ ] `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` passes + per-kind gate
- [ ] Docs updated (`README.md`, `docs/architecture.md`, `docs/philosophy.md` if public behavior changes)
- [ ] No `go.work`, no committed `replace` directives
- [ ] `sync-docs` reports In-sync, `arch-guard` reports Pass

## References
- Spec: `docs/specs/...`
- Vision: `docs/project-vision.md`
- Related issue/PR: #...

## Labels
`kind/<workload>`, `area/<repo>`, `priority/P0-P3`, `spec/<SpecID>` if applicable
```

## 3. Writing rules

- Title uses Conventional Commits `type(scope): summary` and is searchable; use imperative mood.
- Context cites vision and spec, not invention; link `AGENTS.md` or `specs/index.md`.
- Problem is file:line-grounded, not vague.
- Proposal steps are ordered by impact-to-effort then dependency chain; state order upfront.
- Acceptance criteria are binary and include exact gates to run.
- For `arch-guard` findings: Type `Architecture Guard`, include violation `file:line` and `core.IsOptIn` result.
- For `sync-docs` drift: Type `Task`, checklist per repo.

## 4. Suggested `gh` command

```bash
gh issue create --repo krewire/<repo> \
  --title "feat(tui): ..." \
  --label "kind/cli,area/framework,priority/P1" \
  --body-file - <<'EOF'
<markdown above>
EOF
```

## Rules

Read `rules/key-rules.md` before acting.
