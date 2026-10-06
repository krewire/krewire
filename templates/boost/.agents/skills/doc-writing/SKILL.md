---
name: doc-writing
description: Write and sync Krewire documentation — README, architecture.md, philosophy.md, manuscript, and specs — consistent with the unified vision. Triggers: "docs", "README", "architecture.md", "philosophy.md", "manuscript", "write docs".
---

# Doc Writing — Unified Vision Documentation

Produce documentation that is accurate, concise, and in sync with the unified 9-workload framework.

## 1. Source of Truth Hierarchy

Changes flow top-down. Never update upstream to match downstream.

1. `docs/project-vision.md` — workload matrix, roadmap
2. `docs/specs/index.md` — implementation matrix
3. `libs/core/workload.go` — `core.Kind`/`Workloads` (code)
4. Per-repo `docs/architecture.md` / `docs/philosophy.md` / `docs/index.md`
5. `README.md` per repo
6. `docs/manuscript/` (narrative)

## 2. When to Use

- Public behavior changes (new package, new `project.kind`, new `krewire` command)
- New `framework/*` package (`runtime`/`worker`/`service`/`infra`) or `libs/core`/`kern` change
- `arch-guard` or `sync-docs` reports drift

## 3. Workflow

1. **Read upstream first** — `project-vision.md` + `specs/index.md` + `core/workload.go` to get canonical names (`framework/tui` not `framework/cli`).
2. **Use the template** — `rules/doc-template.md` (global standard, IEEE 830 + Diátaxis) for the doc type you write (README / architecture / philosophy / manuscript / spec).
3. **Be concise** — say what it is, why it exists, how to use it. No fluff. Keep workload matrix to 9 rows, architecture tree to one page.
5. **Verify** — run `sync-docs` subagent; it must report `In-sync` before done.

## 4. Doc Types & Templates

| Doc | Template Section in `rules/doc-template.md` | Key Content |
|-----|---------------------------------------------|-------------|
| `README.md` | 2. README | Workload matrix, ecosystem layout, quick start, roadmap |
| `docs/architecture.md` | 3. Architecture | Module structure tree, design decisions, dependency graph |
| `docs/philosophy.md` | 4. Philosophy | Principles (opt-in, stdlib-first, single config) + contribution guidance |
| `docs/index.md` | 5. Index | Hub linking architecture/philosophy/specs |
| `manuscript/*.md` | 6. Manuscript | Narrative, tutorials, ordered Markdown |
| Spec | `spec-writing` skill `rules/spec-template.md` | Spec-specific sections |

## Rules

Read `rules/key-rules.md` and `rules/doc-template.md` before acting.
