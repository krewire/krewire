---
name: conflict-resolution
description: Resolve design, priority, and technical disagreements through structured decision-making (RACI + ADR). Triggers: "conflict", "design decision", "trade-off", "disagreement", "arch-guard".
---

# Conflict Resolution — Structured Design Decisions

Turn disagreements into documented decisions with traceability.

## When to Use

- Two+ valid designs for one requirement (API shape, module boundary, CLI UX)
- `arch-guard` finds competing valid patterns
- Priority conflict between workloads
- Team/AI agents disagree on `arch-guard` interpretation

## Process (RACI + ADR)

### 1. Frame (RACI)
| Role | Who |
|------|-----|
| Responsible | Who analyzes options |
| Accountable | Who decides (decider) |
| Consulted | Who gives input (arch-guard, domain experts) |
| Informed | All stakeholders |

### 2. Options (Trade-off Matrix)
Use `spec-template.md` §6.3 Trade-off Matrix: list all viable alternatives with Pros/Cons/Impact/Effort/Risk. Score by impact-to-effort + dependency chain.

### 3. Evaluate
Criteria (weighted by vision principles):
- Alignment with `project-vision.md` (opt-in, stdlib-first, zero-cost monolith)
- Impact-to-effort (P0→P3) + dependency chain
- Risk (circular deps, opt-in violation, breaking change)

### 4. Decide & Record (ADR)
Decider writes `docs/adr/XXXX-decision.md` (from `spec-template.md` §6.3 or `internal/docs/adr/0000-template.md`):

```markdown
# ADR XXXX — <Title>

| Field | Value |
|-------|-------|
| ID | ADR-XXXX |
| Status | Accepted / Superseded |
| Date | YYYY-MM-DD |
| Deciders | @name |
| Consulted | @name |
| Informed | @all |

## Context
Why, constraints (vision, core.IsOptIn, KWF-M8K2Q phase).

## Decision
Chosen option (one paragraph).

## Consequences
- Positive: ...
- Negative: ...
- Risks: ...

## Alternatives
| Option | Why rejected |
|--------|--------------|
| A — ... | ... |

## Verification
- arch-guard Pass / sync-docs In-sync
```

### 5. Communicate & Trace
- Record decision in spec `FRK-CONFLICT-XXX` row or new `FRK-*` row
- Update `arch-guard` if boundary rules change
- Inform all Consulted/Informed

## Rules

Read `rules/key-rules.md` before acting.