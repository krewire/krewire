# Issue Writing — Key Rules

- One issue per initiative; never batch unrelated changes.
- Title is `type(scope): imperative summary` (Conventional Commits) and searchable.
- Context must link `project-vision.md` and relevant spec `file:line`; never invent architecture.
- Acceptance criteria are binary, include `gofmt`/`go vet`/`go test` + per-kind gate, and trace `FRK-*` → implementation → test.
- For arch-guard violations, include `file:line` and `core` validation result (`IsOptIn`, `ParseKind`).
- Suggest workload-aware labels (`kind/<workload>`, `area/<repo>`), not generic `enhancement`.
- Use `issue-writer` subagent to draft, then human creates via `gh issue create`.
