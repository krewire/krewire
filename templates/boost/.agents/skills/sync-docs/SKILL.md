---
name: sync-docs
description: Keep Krewire docs in sync — README, architecture.md, philosophy.md, specs, manuscript vs unified vision. Triggers: "sync docs", "docs drift", "README", "architecture.md", "manuscript".
---

# Sync Docs — Documentation Consistency

Keep every Krewire document consistent with the single source of truth.

## 1. Source of truth hierarchy

1. `docs/project-vision.md` — 9-workload matrix, roadmap, architecture diagram
2. `docs/specs/index.md` — 51 specs, Spec vs Impl Status
3. `libs/core/workload.go` — `core.Kind`/`Workloads` (canonical code)
4. Per-repo `docs/architecture.md` / `docs/philosophy.md` / `docs/index.md`
5. `README.md` per repo + `docs/manuscript/` (narrative)
6. `AGENTS.md` / `guild/template/AGENTS.md` — 8-kind table, command matrix

Changes flow top-down: vision → specs → `core` → `docs/architecture` → `README`/`manuscript` → `AGENTS.md`.

## 2. Sync checks

| Area | Files to Compare | Drift Example |
|------|------------------|---------------|
| Workload matrix | `README.md` vs `project-vision.md` vs `core.Workloads` | `README` still shows `framework/cli` not `framework/tui` |
| Architecture tree | `framework/docs/architecture.md` vs `libs/docs/architecture.md` vs `project-vision.md` | `framework` tree missing `kern/` or still shows `cli/` not `tui/` |
| Philosophy | `*/docs/philosophy.md` vs `project-vision.md` Principles | Missing `core` declarative + `kern` imperative control plane |
| Spec location | `<project>/docs/specs/` vs `docs/specs/` | Original contains specs not `MOVED.md` |
| Package name | `framework/tui` imports vs docs | Docs still say `framework/cli` or `cli.NewApp` not `tui.NewApp` |
| Manuscript | `docs/manuscript/01-introduction.md` vs vision | Still says meta-framework, not unified 9-workload |

## 3. Workflow

1. Run `sync-docs` subagent (read-only) to report drift with `file:line`.
2. For each drift, apply the hierarchy: update downstream docs to match upstream source (never the reverse).
3. After fixes, re-run `arch-guard` to ensure no boundary violation was introduced.

Example fix order:
```
```

## 4. Gates

- `sync-docs` subagent must report `In-sync` before `reviewer` approves docs PRs.
- `docs/specs/index.md` Impl Status must match `core.Workloads` Status (`Shipped` vs `Planned`).

## Rules

Read `rules/key-rules.md` before acting.
