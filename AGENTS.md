# AGENTS.md — Krewire Core Monorepo

Agent guide for AI agents working inside the `krewire` repository
(module = `github.com/krewire/krewire`).

This repository contains the core monorepo: `packages/` (domain libraries),
`apps/` (services), `tools/kiw` (developer CLI), and `templates/` (Boost AI & starters).

## Language

- **Write all stored artifacts in English** — code, docs, commit messages, specs, reports.
- If the human writes in Indonesian or another language, reply in the human's language in chat, but keep all repository files in English.

## Architecture Rules

Strict downward-only dependencies:
```
templates/boost → tools/kiw → packages/* → packages/model, packages/errs
```
- `packages/model` may depend only on standard library packages and `packages/errs`.
- No package or library may **ever** import `kiw`.
- `tools/kiw` imports packages from `packages/*` directly (e.g. `packages/hub/resolver`). Never duplicate logic between `tools/kiw` and `packages/`.
- Single Source of Truth (SSOT): no duplicate literals, tables, or constants.

## Task & Progress Tracking (GitHub Issues SSOT)

To keep progress tracking fully automated and deterministic for both AI agents and human developers:

1. **GitHub Issues as Single Source of Truth:**
   - Active tasks, features, and fixes are tracked as GitHub Issues under `krewire/krewire`.
   - Every issue body contains a structured checklist with `- [ ]` and `- [x]` checkboxes.

2. **Automated Agent & Developer Progress Workflow:**
   - **Discovering tasks:** Query active issues via CLI/API:
     ```bash
     gh issue list -R krewire/krewire --state open --json number,title,url
     gh issue view <number> -R krewire/krewire --json title,body
     ```
   - **Verifying completion:** Run tests:
     ```bash
     go test ./...
     ```
   - **Updating status:** When a task is implemented and verified by tests, check off the checklist item (`- [ ]` → `- [x]`) in the GitHub Issue body. Commits must reference the issue number (`feat(scope): implement feature (ref #<IssueNumber>)`). Close the issue once all tasks are completed.

## Quality Gates

Before finishing any task, run:
```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
go build -o bin/kiw ./tools/kiw/cmd/kiw
```

## Commit Discipline

- **Conventional Commits**: `feat|fix|docs|test|refactor|perf|security|chore(scope): desc`.
- Atomic commits (≤~300 LOC).
- Run `git` inside `krewire/`.
