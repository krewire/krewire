# Contributing to Krewire

Welcome! This guide covers *how* to contribute to the **Krewire Core Monorepo** (`github.com/krewire/krewire`): the workflow, architectural invariants, conventions, and quality gates every change must pass.

> All stored code, comments, docstrings, commit messages, specifications, and repository docs are strictly in **English**.
> If communicating in Indonesian or another language in chat dialogue, reply in that language, but keep all repository files and code artifacts strictly in English.

---

## 1. Architectural Rules & Invariants (Do Not Violate)

Strict downward-only dependencies are strictly enforced across the ecosystem (ADR-E8P9D, ADR-B7D3A):

```text
Layer 3: krew (CLI Toolchain) ──► Layer 2: <domain-modules> ──► Layer 1: model, errs ──► Layer 0: Go Stdlib
```

1. **Pure Multi-Module Monorepo**:
   - The repository contains 37 independent Go modules (`ai/`, `orm/`, `web/`, etc.).
   - Never add a `go.mod` file to the root of the repository; local resolution is managed via `go.work`.
2. **Third-Party Isolation Guard (Port & Adapter Invariant)**:
   - The foundational monorepo is **100% Go Standard Library** (pure Go, CGO-free, zero third-party drivers).
   - Third-party database drivers (`sqlite`, `postgres`, `mysql`, `redis`) and frontend compilers (`svelte`, `tailwind`, `vue`) must enter the ecosystem exclusively via `github.com/krewire/hub/*`.
3. **Kernel Boundary (`model`, `errs`)**:
   - `model` may depend only on standard library packages and `errs` — never on higher-level packages and never on `krew`.
4. **Complete Tooling Isolation**:
   - No module or library under `krewire` may ever import the CLI toolchain (`krew`). Enforced via conformance layering tests.
5. **Pure Go / Zero CGO (`CGO_ENABLED=0`)**:
   - All Go code must compile cleanly with `CGO_ENABLED=0`. Never introduce native C dependencies or shared library linkages.
6. **Decoupled Peers**:
   - Sibling domain modules are peers, not an arbitrary hierarchy: `web` may use `ui`, but `ui` may not use `web`.

---

## 2. Development Workflow

1. **GitHub Issues Single Source of Truth**:
   - Every pull request must be linked to an active GitHub Issue (`ref #<IssueNumber>`).
   - Mark off completed checklist items (`- [ ]` to `- [x]`) in the issue body.

2. **Branching Model (Trunk-Based)**:
   - `main` is protected; short-lived feature and bugfix branches are created from it.
   - Branch naming: `feat/<name>`, `fix/<name>`, `refactor/<name>`, `docs/<name>`.
   - Work in small, focused increments. One logical change per PR with atomic commits ($\le$ ~300 LOC per commit).

3. **Conventional Commits**:
   Commit messages strictly follow [Conventional Commits](https://www.conventionalcommits.org/):
   ```text
   feat(web): add support for chunked transfer encoding (ref #102)
   fix(orm): resolve relative path handling in sqlite migrations
   test(model): assert workload kind detection precedence
   docs(core): update monorepo layout and architecture index
   ```

---

## 3. Mandatory Quality Gates

Before submitting code, all 4 gates must pass:

```bash
gofmt -l .        # Must return 0 lines
go vet ./...      # Must exit code 0
go test ./...     # All unit and layering tests pass
CGO_ENABLED=0 go build ./... # Pure-Go compilation succeeds
```
