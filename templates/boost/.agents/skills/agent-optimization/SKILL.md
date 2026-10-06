---
name: agent-optimization
description: Optimize AI agent capacity — context compaction, parallel orchestration, lazy loading, and vision-aware routing for the unified Krewire ecosystem. Use when agent performance is slow, context overflows, or multi-workload tasks need faster execution.
---

# Agent Optimization — Capacity Tuning

Systematically optimize how agents consume context and execute work for the 8-kind unified framework.

## 1. Context Compaction (save 60-80% tokens)

- **Summarize, don't paste:** When delegating to `runtime`/`infra`/`service`/`worker`, pass a 5-line vision summary + the single `KWF-*` spec ID, not full docs.
- **Lazy load specs:** Only `Read` the detailed `KWF-T4X9P`/`B7N3D`/`L5H2F` when the slice touches it. Use `docs/specs/index.md` to check `Planned` vs `Shipped` before loading.
- **Cache spec index:** `docs/specs/index.md` is the single source for dependency ordering — read once, reuse across subagents.

## 2. Parallel Orchestration (2-4x speedup)

```
Human request (multi-workload)
       │
       ▼
  scout (if kind unknown) ──→ vision compact
       │
       ▼
  orchestrator ──→ decompose into slices
       │
       ├─→ runtime subagent ─┐
       ├─→ infra subagent ───┤ parallel
       ├─→ service subagent ─┤ (independent slices)
       ├─→ worker subagent ──┘
       │
       ▼
  build (compose) + tester/reviewer (verify)
```

- Independent slices (no shared files) → launch as parallel tool calls in one turn.
- Dependent slices (e.g., `infra` state before `service` deploy) → order by `Depends On` column, state the order up front.
- Batch `Read` calls: multiple files in one turn, not sequential.

## 3. Skill Routing (vision-aware)

| Trigger | Skill | Subagent |
|---------|-------|----------|
| WASM, hydration, VDOM, island, widget, frontend | `wasm-runtime` | `runtime` |
| infra, AWS, Kubernetes, terraform, plan, state | `infra-provision` | `infra` |
| service, registry, gateway, circuit breaker, tracing | `service-mesh` | `service` |
| worker, queue, cron, DLQ, background job | `worker-queue` | `worker` |
| multi-workload, cross-cutting | `vision` + `agent-optimization` | `orchestrator` → `build` |

Use `scout` + `vision` first when kind is unknown; then route.

## 4. Performance Budgets

- **Context budget:** ≤ 5% of window for vision; ≥ 90% for task code.
- **Latency budget:** scout (< 15s) → vision (5s) → parallel slices (60s) → verify (30s).
- **Quality gates remain:** `gofmt -l .`, `go vet ./...`, `go test ./...` + per-kind gate — run independent gates in parallel, not sequential.

## 5. Measurement

- After optimization, report: tokens saved (compact vs full), parallel speedup (slices run in parallel vs sequential), and cache hits (spec index re-reads avoided).
- If context still overflows, split the task — scope creep in a spec is a sign to split, same for agent work.

## Rules

Read `rules/key-rules.md` before acting. Use this skill with `build`/`plan` for any multi-workload task.
