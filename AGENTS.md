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
templates/boost → tools/kiw → packages/* → packages/kern (model/lifecycle)
```
- `packages/kern/model` may depend only on standard library packages.
- No package or library may **ever** import `kiw`.
- `tools/kiw` imports packages from `packages/*` directly (e.g. `packages/hub/resolver`). Never duplicate logic between `tools/kiw` and `packages/`.
- Single Source of Truth (SSOT): no duplicate literals, tables, or constants.

## Specs & Progress Tracking (GitHub Issues SSOT)

To keep progress tracking fully automated, resilient against markdown parsing drift, and deterministic for both AI agents and human developers:

1. **GitHub Issues as Single Source of Truth:**
   - Active specifications and their implementation progress are tracked as GitHub Issues under `krewire/krewire` with label `spec`.
   - Title format: `[Spec: <SpecID>] <Title>` (e.g. `[Spec: KWN-INFO-BNKJC] Project Information`, `[Spec: KWL-CORE-M1ZKS] Krewire Libraries`).
   - Every issue body contains the structured requirement checklist with `- [ ]` and `- [x]` checkboxes citing `ReqID`, priority, and scope (e.g. `- [ ] RND-IF-001 [Must] ...`).
   - Index files in `packages/docs/specs/index.md` and `tools/kiw/docs/specs/index.md` link directly to these GitHub Issues.

2. **Spec-Driven Traceability in Code:**
   - ID prefixes: `KWF-*` (framework/core), `KWN-*` (kiw), `KWL-*` (libs/packages), `KWG-*` (boost).
   - Traceability: test files must cite `// Tests for <SpecID>` and individual tests cite `// Spec: <SpecID> <ReqID> Scope: <Scope>` or follow `Test<SpecID>_<ReqID>_<Scenario>`.

3. **Automated Agent & Developer Progress Workflow:**
   - **Discovering tasks:** Query active specs via CLI/API:
     ```bash
     gh issue list -R krewire/krewire --label spec --json number,title,url
     gh issue view <number> -R krewire/krewire --json title,body
     ```
   - **Verifying completion:** Run tests filtered by spec or requirement:
     ```bash
     kiw test --spec <SpecID>
     go test -run Test<ReqID> ./...
     ```
   - **Updating status:** When a requirement is implemented and verified by tests, check off the requirement box (`- [ ]` → `- [x]`) in the GitHub Issue body. Commits must reference the issue number and requirement ID (`feat(scope): implement <ReqID> (ref #<IssueNumber>, req <ReqID>)`). Close the issue once all requirements are completed.

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
