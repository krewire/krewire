# Contributing to Krewire Core Monorepo

Welcome to the **Krewire Core** contributor guide. This document explains the development flow, architectural invariants, and quality gates for contributing to `github.com/krewire/krewire`.

## 1. Architectural Rules (Do Not Violate)

Strict downward-only dependencies are strictly enforced:

```text
templates/boost → tools/kiw → pkg/* → pkg/model, pkg/errs
```

- `pkg/model` may depend only on standard library packages and `pkg/errs` — never on higher-level packages and never on `kiw` (ADR-B7D3A).
- No package or library under `pkg/` may ever import `kiw`.
- Generated projects depend on Krewire packages only — never on `kiw`.
- Sibling packages under `pkg/` are peers, not an arbitrary hierarchy: `web` may use `ui`, but `ui` may not use `web`.
- Control flow is 100% Go; boundary dialects (HTML, CSS, SQL, YAML, Markdown) remain native at their layer.

## 2. Development Workflow

1. **GitHub Issues Single Source of Truth**:
   - Every pull request must be linked to an active GitHub Issue.
   - Mark off completed checklist items (`- [ ]` to `- [x]`) in the issue body.

2. **Branching & Commits**:
   - Create feature or fix branches from `main`: `feat/<name>`, `fix/<name>`, `refactor/<name>`.
   - Conventional Commits:
     ```text
     feat(web): add support for chunked transfer encoding (ref #102)
     fix(kiw): resolve relative path handling in static builds
     test(model): assert workload kind detection precedence
     ```
   - Atomic commits ($\le$ ~300 LOC per commit).

## 3. Quality Gates

Run these commands locally before submitting any PR:

```bash
# 1. Format verification (must produce no output)
gofmt -l .

# 2. Static analysis
go vet ./...

# 3. Comprehensive tests
go test ./...

# 4. Binary build verification
go build -o bin/kiw ./tools/kiw/cmd/kiw
```

## 4. Single Source of Truth (No Hardcoding)

- A fact or configuration default that the ecosystem agrees on is declared once in the owning layer (`pkg/model` for domain rules, owning packages for behaviors).
- Copying a literal or constant into a second package is prohibited. Export and call the resolver function instead.
- All code, documentation, and commit messages must be written in English.
