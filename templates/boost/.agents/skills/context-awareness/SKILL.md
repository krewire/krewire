---
name: context-awareness
description: Situational awareness for Krewire — project kind, vision matrix, spec history, environment, and stale-context detection. Use before any task to build a correct, minimal mental model. Triggers: "context", "awareness", "what kind", "where am I", "project context".
---

# Context Awareness — Situational Intelligence

Build a correct, minimal, and fresh mental model before acting. Wrong context causes wrong code; this skill prevents it.

## 1. Four Context Layers

Load in order, stop when sufficient:

### Layer 1 — Project Context (where am I?)
- **Kind detection** — read `krewire.yaml` (`project.kind` among 8: `app`/`cli`/`site`/`book`/`worker`/`service`/`infra`/`kernel`), fallback markers (`ssg:` → `site`, `manuscript/` → `book`, `infra/` → `infra`, root `main.go` → `app`). Cross-check `krewire info`.
- **Layout** — `main.go` + `internal/` (`app`/`cli`/`worker`/`service`), `web/` (`app`), `manuscript/` (`book`), `ssg:` (`site`), `infra/` (`infra`).
- **Toolchain** — use `scout` subagent (read-only) to map without side effects.

### Layer 2 — Vision Context (what ecosystem?)
- **Workload matrix** — 9 workloads, 8 kinds, opt-in batteries (monolith zero-cost). Know which slice is `✅ Shipped` vs `🔜 Planned` via `docs/specs/index.md`.
- **Phase awareness** — Phase 1 WASM (`KWF-T4X9P`), Phase 2 infra (`KWF-B7N3D`), Phase 3 service/worker (`KWF-L5H2F`). Planned specs are spec-first.

### Layer 3 — Historical Context (what changed?)
- **Git history** — `git log --oneline -10` or `git log -1 -- <path>` for docs; revision lives in git, not in-file `Version`.
- **Spec history** — `docs/specs/` is  in `krewire/internal`; original `<project>/docs/specs/` are redirects (`MOVED.md`).
- **Stale detection** — if `AGENTS.md` mentions 5 kinds or `ssg.yaml`, context is stale → reload vision.

### Layer 4 — Environment Context (what can I run?)
- **Go toolchain** — `go version`, `GOWORK` disabled (no `go.work`), temp `replace` in `go.mod` for cross-repo testing.
- **Krewire binary** — `bin/krewire` (workspace-built) vs installed `krewire`; rebuild after `framework`/`guild` changes via `go build -o ../bin/krewire ./cmd/krewire`.
- **Branches** — `docs/` repo: `main` holds `manuscript/`+`krewire.yaml`, `gh-pages` holds `site/` output.

## 2. Progressive Disclosure

```
Scout (kind + workflow) → Vision compact (5 lines) → Spec index (which specs) → Single KWF-* spec (slice detail) → Code
```

Never load all specs per task. Lazy-load the one `KWF-*` you touch after checking `index.md`.

## 3. Caching & Freshness

- **Cache:** After first `project-vision.md` read, summarize 9-row matrix in 5 lines and reuse for subagents.
- **Invalidate:** On `git checkout`/`pull`, re-run `scout`; on new `KWF-*` spec added, re-read `index.md`.
- **Share:** Orchestrator passes compact summary to parallel subagents — don't let each re-read vision.

## 4. Anti-Patterns

- Assuming `app` when `worker:`/`service:`/`infra:` exists.
- Using per-repo `docs/specs/` instead of  `docs/specs/`.
- Invoking `krewire build` without `krewire info` kind check.
- Treating `gh-pages` branch as source for `manuscript/` edits.

## 5. Quick Checklist

- [ ] `krewire.yaml` + `krewire info` read; kind identified among 8
- [ ] `docs/specs/index.md` checked for Spec vs Impl status
- [ ] Git log checked for recent changes to touched area
- [ ] Environment (`go version`, `bin/krewire`) verified if running gates

## Rules

Read `rules/key-rules.md` before acting.
