# Impact-to-Effort Prioritization — Global Standard (Krewire)

Sort **every** decision, backlog, task list, spec list, migration plan, release plan, or todo by **impact-to-effort** first, then **dependency chain**.

## Matrix

|               | Low Effort | High Effort |
|---------------|------------|-------------|
| **High Impact** | **P0 — Do first** (quick wins) | **P1 — Plan next** (strategic) |
| **Low Impact**  | **P2 — Do if time** (fillers)  | **P3 — Avoid / defer** (time sinks) |

**Impact:** User value, revenue, workload coverage, architectural leverage, or debt reduction. Measured by workload matrix (e.g., `core/kern` control plane > single `cli` flag).

**Effort:** Dev time, complexity, risk, and dependencies. `XS` (<1h), `S` (half-day), `M` (1-2d), `L` (week), `XL` (split).

## Dependency Chain

After sorting by impact-to-effort, reorder by **foundations before dependents**:

```
libs/core (Kind/Workload) → libs/kern (Kernel) → framework/* → krewire → docs/landing
spec → implementation → test → docs sync → arch-guard
```

A high-impact, low-effort item that depends on a foundation still waits for the foundation.

## Krewire-Specific Ordering

1. **Foundations first:** `KWL-K1N2Q` (core business rules) before `KWL-KERN-X8P3L` (kernel) before `KWF-T4X9P` (runtime) / `KWF-B7N3D` (infra) / `KWF-L5H2F` (service/worker)
2. **Spec before code:** A `Must` requirement row (`FRK-*`) must exist before implementation
3. **State the order upfront:** Per `AGENTS.md` Sorting rule, every plan starts with "Sorted by impact-to-effort (P0→P3) then dependency" so the human can correct

## How to Use

1. List all tasks with `Impact (H/L)` and `Effort (XS-XL)`
2. Assign `P0-P3` via matrix
3. Sort `P0 → P1 → P2 → P3`, then reorder within each `P` by dependency
4. Present the ordered list and ask for correction before starting (per `AGENTS.md`)
5. When the human gives multiple instructions, reorder them the same way and complete one before starting the next

### Example Backlog

| Task | Impact | Effort | P | Depends On | Order |
|------|--------|--------|---|------------|-------|
| Add `core.Kind` registry | H | S | P0 | — | 1 |
| Add `kern.Supervisor` | H | M | P0 | core | 2 |
| WASM widget `Button` | H | M | P1 | runtime VDOM | 4 |
| Infra AWS provider S3 | H | L | P0 | kern | 3 |
| Docs typo fix | L | XS | P2 | — | 5 |

## Checklist Before Starting

- [ ] Every item has `Impact` + `Effort` + `P` + `Depends On`
- [ ] Ordered list is `P0 → P3` then dependency, with order stated upfront
- [ ] Foundations (`core`/`kern`/`spec`) before dependents (`runtime`/`service`)
- [ ] No `XL` items — split before starting
