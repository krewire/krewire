# Agent Workflow — Key Rules

- Lifecycle is Understand → Plan → Implement → Verify → Summarize; ask if ambiguous before Plan.
- Handoffs pass compact state (Task/Kind/Vision/Spec/Files/Gates), not full history.
- Order by impact-to-effort then dependency chain; state order upfront; parallelize independent slices, sequence dependent ones.
- Batch `Read` calls in one turn; launch independent subagents in parallel in one turn.
- Never claim a gate passes without running it (`gofmt -l .`, `go vet ./...`, `go test ./...` + per-kind gate).
