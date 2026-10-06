# Agent Optimization — Key Rules

- Summarize vision in 5 lines when delegating; never paste full docs to specialized subagents.
- Lazy-load `KWF-*` specs — only the slice you touch, after checking `docs/specs/index.md`.
- Parallelize independent slices (runtime/infra/service/worker) in one turn; order dependent slices by `Depends On` and state the order upfront.
- Batch `Read` calls; never sequential file reads when parallel is possible.
- Run independent quality gates in parallel, not sequential.
