---
name: agent-workflow
description: Orchestrate Krewire AI agents — handoffs, state, parallel vs sequential, and lifecycle from kickoff to verified delivery. Use for multi-step tasks, multi-workload work, or when optimizing agent throughput. Triggers: "workflow", "orchestrate", "handoff", "parallel", "agent flow".
---

# Agent Workflow — Handoffs, State & Lifecycle

Systematically move work through the right agents, with the right state, at the right concurrency.

## 1. Lifecycle (5 states, per AGENTS.md)

```
Understand → Plan → Implement → Verify → Summarize
     ↑           ↓
     └──── Ask if ambiguous
```

- **Understand:** `context-awareness` + `scout` (kind/workflow/layout). Never assume.
- **Plan:** `plan` agent or `spec-writing` skill for non-trivial tasks; read `docs/project-vision.md` and relevant `KWF-*` spec.
- **Verify:** `tester` + `quality-gate` (per-kind gates) + `reviewer`/`security` before merge.
- **Summarize:** What changed, why, how verified (real commands, not claims).

## 2. Agent Roster (15 agents) & Routing

| Need | Agent | Mode |
|------|-------|------|
| Implement, debug, build | `build` | primary |
| Map unfamiliar project | `scout` | subagent (read-only) |
| Design before code | `plan` | primary (read-only unless asked) |
| WASM frontend | `runtime` | subagent |
| Cloud infra | `infra` | subagent |
| Microservice | `service` | subagent |
| Background jobs | `worker` | subagent |
| Route multi-workload tasks | `orchestrator` | subagent |
| Review before merge | `reviewer` | subagent (read-only) |
| Tests | `tester` | subagent |
| Root-cause failure | `debugger` | subagent |
| Large safe restructure | `refactor` | subagent |
| Docs/specs | `docs` | subagent |
| Security pass | `security` | subagent (read-only) |
| Ship | `deploy` | subagent |

Routing rule: `vision` skill picks slice → `orchestrator` decomposes → `build` delegates slices → `tester`/`reviewer` verify.

## 3. Handoff Protocol

For every handoff, pass a **compact state packet** (not full history):

```
Task: <one-line goal>
Kind: <app|cli|site|book|worker|service|infra|kernel>
Vision: <5-line matrix summary>
Spec: <KWF-* ID or "spec-first needed">
Files: <touched paths>
Gates: <which gates to run>
```

- **Do** include spec ID and requirement row (`FRK-*`) for traceability.

## 4. Parallel vs Sequential

**Parallelize when:**
- Slices touch disjoint files (e.g., `runtime` VDOM + `infra` provider)
- Independent quality gates (`gofmt` + `go vet` + `go test` in different repos)
- Independent `Read` calls (multiple files in one turn)

**Sequence when:**
- Dependency chain (`infra` state before `service` deploy; `spec` before code)
- Same file touched by multiple slices
- `Plan` before `Implement`

Batch: independent tool calls in one turn (one tool per message, parallel turns). Orchestrator states the chosen order up front per AGENTS.md sorting rule.

## 5. State Management

- **Per-project map:** After `scout` + `kickoff`, record kind/workflow/layout in `AGENTS.md` under "Project-Specific Customization" or `docs/PROJECT_MAP.md` to reuse.
- **Spec traceability:** Every new behavior links `spec → implementation → test` (`KWF-*` + `FRK-*`).
- **Stale detection:** If subagent reports 5 kinds or `ssg.yaml`, re-load vision — context is stale.

## 6. When to Use Which Skill with Workflow

| Skill | Paired Agent |
|-------|--------------|
| `vision` | `plan`/`orchestrator` start |
| `context-awareness` | `scout` + every primary task start |
| `wasm-runtime` | `runtime` |
| `infra-provision` | `infra` + `deploy` |
| `service-mesh` | `service` |
| `worker-queue` | `worker` |
| `agent-optimization` | `orchestrator` for perf tuning |
| `quality-gate` | `tester` + `reviewer` before done |

## 7. Anti-Patterns

- Running `plan` + `build` sequentially when independent — parallelize.
- Sequential `Read` per file — batch.
- Handoff without vision summary — causes subagent to re-read full docs.
- Claiming gates passed without running real commands.

## Rules

Read `rules/key-rules.md` before acting. Pair with `context-awareness` at session start.
