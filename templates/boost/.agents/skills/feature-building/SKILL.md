---
name: feature-building
description: Spec-driven feature delivery from requirement to verified implementation. Triggers: "feature", "build feature", "spec-driven", "new capability", "implement spec".
---

# Feature Building — Spec-Driven Delivery

Deliver features from requirement to verified code with full traceability.

## 1. Lifecycle (Spec-Driven)

```
Requirement → Spec (Draft→Approved) → Plan (impact-to-effort) → Implement (slices) → Verify (gates) → Summarize
```

**At each step, the artifact is the spec, not the code.**

## 2. Step-by-Step

### 2.1 Requirement → Spec (Spec-First)
- Run `requirement-gathering` skill to clarify user story, acceptance criteria, workload kind.
- Write spec using `spec-writing` skill with `rules/spec-template.md` (global standard).
- One spec per initiative: `{ProjectId}-{Scope}-{SpecID}-{slug}.md` in `docs/specs/`.
- Requirement rows (`FRK-*`/`KWL-*`) use RFC 2119 `MUST`/`SHOULD` and trace to implementation.
- Update `docs/specs/index.md` and `docs/specs/index.md` (Spec vs Impl Status).

### 2.2 Spec → Plan (Impact-to-Effort)
- Run `plan` agent with `agent-workflow` + `vision` skills.
- Order by `impact-to-effort` (P0→P3) then dependency chain (foundations first) — see `agent-workflow` skill `rules/impact-to-effort.md`.
- State order upfront for human correction (per `AGENTS.md` Sorting rule).

### 2.3 Plan → Implement (Slice Parallelism)
- Decompose into independent slices: `runtime` (WASM), `infra` (provider), `service` (gateway/resilience), `worker` (queues), `tui` (CLI).
- Control plane: business rules in `libs/core` (`core.Kind`/`Workload`), execution in `libs/kern` (`Kernel`/`Supervisor`).

### 2.4 Implement → Verify (Gates + Traceability)
- **Per-kind gates:** `go build .` (`app`/`cli`/`worker`/`service`), `krewire build` (`site`/`book`), `krewire build --plan` (`infra`), WASM hydration check (`runtime`).
- **Traceability:** Every `Must` row (`FRK-*`/`KWL-*`) has a test `file:line` and code `file:line` link in spec.
- `tester` + `reviewer` + `arch-guard` + `sync-docs` all must pass.

### 2.5 Summarize
Report: SpecID, requirement rows (`FRK-*`/`KWL-*`), files changed, gates passed, traceability matrix.

## Rules

Read `rules/key-rules.md` before acting.