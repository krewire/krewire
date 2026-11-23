# AGENTS.md — Krewire Core Monorepo

Agent guide for AI agents working inside the `krewire` repository
(module = `github.com/krewire/krewire`).

This repository contains the core monorepo: `pkg/` (domain libraries),
`apps/` (services), and `templates/` (Boost AI & starters).

## Language

- **Write all stored artifacts in English** — code, docs, commit messages, specs, reports.
- If the human writes in Indonesian or another language, reply in the human's language in chat, but keep all repository files in English.

## Architecture Rules

Strict downward-only dependencies:
```
templates/boost → krew → pkg/* → pkg/model, pkg/errs
```
- `pkg/model` may depend only on standard library packages and `pkg/errs`.
- No package or library under `pkg/` may **ever** import `krew` or CLI tooling.
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
CGO_ENABLED=0 go build ./...
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./pkg/runtime/...
go mod tidy -diff
```
These are the same gates CI (`.github/workflows/ci.yaml`) enforces on every pull
request; see `docs/guides/testing-and-qa.md` §6. Release artifacts are built only
by the release pipeline (`docs/foundation/release-and-compatibility.md` §6).

## Commit Discipline

- **Commit and push automatically.** Once the quality gates pass, stage, commit, and push the change without waiting to be asked.
- **Conventional Commits**: `feat|fix|docs|test|refactor|perf|security|chore(scope): desc`.
- Atomic commits (≤~300 LOC).
- Run `git` inside `krewire/`; push to `origin` when the tree is green.

## Versioning, Release & Hotfix Protocol

- **Module Version Contract**: Every module declares its own version and dependencies in `<module>/version.go` (`Version` and `EcosystemRequires`).
- **Compatibility Verification**: `krew compat` validates cross-module compatibility dynamically across all workspace modules.
- **Release Bumping**: Performed using `krew release --all --bump <kind> --apply`.
- **Production vs Hotfix Deployment**:
  - Production releases require stable SemVer tags (`vX.Y.Z`) on `main`.
  - Emergency hotfixes bypass the release ceremony and tags entirely; they are deployed directly via the dedicated `hotfix` branch (`git checkout hotfix && git merge main --ff-only && git push origin hotfix`). Never tag hotfixes.
- Never force-push or rewrite published history unless explicitly asked.
