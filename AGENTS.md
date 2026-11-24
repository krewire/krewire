# AGENTS.md — Krewire Core Monorepo

Agent guide for AI agents working inside the foundational `krewire` repository.

This repository is structured as a **Pure Multi-Module Monorepo** comprising **37 independent domain modules** coordinated locally via `go.work`, with zero `go.mod` at the repository root.

## Language Standards

- **Write all stored artifacts strictly in English** — code, comments, docstrings, docs, commit messages, specifications, and reports.
- If the human writes in Indonesian or another language, reply in the human's language in chat dialogue, but maintain all repository files and code artifacts in English.

## Architecture Rules & Invariants

Strict downward-only dependencies (Pyramid Architecture ADR-E8P9D, ADR-B7D3A):
```text
templates/boost ──► krew ──► <module> (web, orm, sec, ai) ──► model, errs ──► Go Stdlib
```
- `model` (Layer 1) may depend only on standard library packages and `errs`.
- No module or library under `krewire/` may **ever** import `krew` or CLI tooling.
- **Port & Adapter Rule**: Foundational monorepo is 100% Go Standard Library (pure-Go, CGO-free). Third-party database drivers (`sqlite`, `postgres`, `mysql`, `redis`) and frontend compiler plugins (`svelte`, `tailwind`, `vue`) reside exclusively in `krewire/hub/*`.
- Single Source of Truth (SSOT): no duplicate literals, tables, or constants.

## Multi-Module Monorepo Layout

Every package is an independent Go module with its own `go.mod`:
```bash
# Testing a specific module:
cd orm && go test -v ./...
# Running all tests via go.work:
go test ./...
```

## Task & Progress Tracking (GitHub Issues SSOT)

1. **GitHub Issues as Single Source of Truth:**
   - Active tasks, features, and fixes are tracked as GitHub Issues under `krewire/krewire`.
   - Every issue body contains a structured checklist with `- [ ]` and `- [x]` checkboxes.

2. **Automated Agent & Developer Progress Workflow:**
   - **Discovering tasks:**
     ```bash
     gh issue list -R krewire/krewire --state open --json number,title,url
     gh issue view <number> -R krewire/krewire --json title,body
     ```
   - **Verifying completion:** Run all 4 mandatory quality gates.
   - **Updating status:** When a task is implemented and verified by tests, check off the checklist item (`- [ ]` → `- [x]`) in the GitHub Issue body. Commits must reference the issue number (`feat(scope): implement feature (ref #<IssueNumber>)`). Close the issue once all tasks are completed.

## Quality Gates

Before finishing any task, run:
```bash
gofmt -l .        # must return 0 lines
go vet ./...      # must exit with code 0
go test ./...     # all tests must pass
CGO_ENABLED=0 go build ./... # pure-Go compilation must succeed
```
