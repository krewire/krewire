# Conflict Resolution — Key Rules

- Every design conflict must result in an ADR (`docs/adr/XXXX-<slug>.md`) or a `FRK-CONFLICT-XXX` spec row; never decide verbally.
- Decision owner is the Accountable person (RACI); if unclear, escalate to human.
- Evaluation criteria: vision alignment (opt-in, stdlib-first, zero-cost) > impact-to-effort (P0→P3) > dependency chain.
- If conflict spans workloads, use `orchestrator` to align slices before deciding.
- `arch-guard` rules may be updated post-decision if boundary rules change.
- `arch-guard` and `sync-docs` must still pass after decision implementation.
- For small decisions (<1h), record as `FRK-CONFLICT-XXX` in spec instead of full ADR.