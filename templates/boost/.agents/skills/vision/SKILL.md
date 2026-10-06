---
name: vision
description: Shared vision context for all Krewire agents. Load this first to align with the unified framework — workload matrix, roadmap, and spec index. Use before any workload-specific work.
---

# Vision — Shared Context Loader

Load the unified Krewire vision efficiently. Every agent should start here instead of re-reading scattered docs.

## 1. Load the compact vision

Read in order, stop when you have enough:

1. `docs/project-vision.md` — 96-line workload matrix, architecture, roadmap (phases 0-5), principles.
2. `docs/specs/index.md` — implementation matrix (49 specs, Spec vs Impl Status).
3. Only the specific `KWF-*` spec you need (e.g., `KWF-T4X9P` for runtime, `KWF-B7N3D` for infra).

Do NOT re-read `AGENTS.md` + `README.md` + all specs per task — the vision file is the compressed source.

## 2. Extract what you need

- **Workload:** which `project.kind` (8 kinds: `app`/`cli`/`site`/`book`/`worker`/`service`/`infra`/`kernel`) and which phase (1 WASM, 2 infra, 3 service/worker).
- **Status:** is it `✅ Shipped` (implement directly) or `🔜 Planned` (spec-first, then scaffold)?
- **Dependencies:** read the `Depends On` column to order work.

## 3. Route to the right slice

| Workload | Subagent | Skill |
|----------|----------|-------|
| Frontend / WASM / hydration | `runtime` | `wasm-runtime` |
| Cloud infra / deploy | `infra` | `infra-provision` |
| Microservice / gateway / resilience | `service` | `service-mesh` |
| Background jobs / queues | `worker` | `worker-queue` |
| Cross-cutting / integration | `build` (orchestrator) | `vision` + relevant skills |

Use `scout` first if the project kind is unknown; then load `vision` to pick the slice.

## 4. Optimize context

- **Cache:** after reading `project-vision.md`, summarize the 9-row matrix in 5 lines for downstream subagents — don't paste the full file.
- **Lazy load:** only load the detailed `KWF-*` spec when the task touches that slice.
- **Parallelize:** independent slices (e.g., `runtime` build + `infra` plan) → launch subagents in parallel with the shared 5-line vision summary, not full doc re-reads.

## Rules

Read `rules/key-rules.md` before acting.
