---
description: Run the project tests. Optionally target a package or file: /test [target]. Adds tests when patterns expect it.
agent: build
---

Run the tests for this Krewire project.

1. Run the suite with `krewire test` (`go test ./...`). If $ARGUMENTS is provided, target that package or file first, e.g. `go test ./internal/scaffold/...`.
2. If tests fail, run the `debugger` subagent to find the root cause and fix it.
3. If behavior was changed and no test covers it, add/update tests following the repo's patterns (use the `tester` subagent), linking to the spec requirement ID when the repo tracks them.

Report: what was run, results, and any tests added or changed.