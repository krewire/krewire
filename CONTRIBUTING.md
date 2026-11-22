# Contributing to Krewire

Welcome! This guide covers *how* to contribute to the **Krewire Core Monorepo** (`github.com/krewire/krewire`): the workflow, architectural invariants, conventions, and quality gates every change must pass.

> All stored code, comments, docstrings, commit messages, specifications, and repository docs are in **English**.
> If communicating in Indonesian or another language in chat dialogue, reply in that language, but keep all repository files and code artifacts strictly in English.

---

## 1. Architectural Rules & Invariants (Do Not Violate)

Strict downward-only dependencies are strictly enforced across the ecosystem (ADR-E8P9D, ADR-B7D3A):

```text
Layer 3: krew (CLI Toolchain) ──► Layer 2: pkg/* ──► Layer 1: pkg/model, pkg/errs ──► Layer 0: Go Stdlib
```

1. **Kernel Boundary (`pkg/model`, `pkg/errs`)**:
   - `pkg/model` may depend only on standard library packages and `pkg/errs` — never on higher-level packages and never on `krew`.
2. **Complete Tooling Isolation**:
   - No package or library under `pkg/` may ever import the CLI toolchain (`krew`). Enforced via AST in `layering_test.go`.
3. **Pure Go / Zero CGO (`CGO_ENABLED=0`)**:
   - All Go code must compile cleanly with `CGO_ENABLED=0`. Never introduce native C dependencies or shared library linkages.
4. **Single Source of Truth (SSOT)**:
   - A fact or configuration default that the ecosystem agrees on is declared once in the owning layer (`pkg/model` for domain rules, owning packages for behaviors).
   - Copying a literal or constant into a second package is prohibited. Export and call the resolver function instead.
5. **Decoupled Peers**:
   - Sibling packages under `pkg/` are peers, not an arbitrary hierarchy: `web` may use `ui`, but `ui` may not use `web`.

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
   fix(kiw): resolve relative path handling in static builds
   test(model): assert workload kind detection precedence
   docs(core): update monorepo layout and architecture index
   ```

---

## 3. Quality Gates (Per Module)

Run these checks before submitting every PR:

```bash
# 1. Format check (must output nothing)
gofmt -l .

# 2. Static analysis
go vet ./...

# 3. Unit and layering tests
go test ./...

# 4. Cross-module compatibility check
krew compat
```

---

## 4. The `version.go` Compatibility Contract

Each module declares its version and its minimum required versions in `<module>/version.go`:

- `Version` — the module's own semantic version (kept in sync with its `go.mod` pin).
- `EcosystemRequires` — `map[ModuleName]Version` of the minimum versions of the modules it depends on.

Rules:
- Never leave `Version` and the `go.mod` pin disagreeing.
- When you bump a module, bump every dependent module's `EcosystemRequires` that references it. `krew release` does this for you (maintainer mode).
- Run `krew compat` after any `version.go` change — it must print `ok`.

---

## 5. Releasing

`krew release` is the single release orchestrator and is a **dry run by default** (pass `--apply` to write):

- **Maintainer mode** (inside the workspace, `--module`/`--all`) — bumps a module's `version.go` `Version` and propagates the new minimum into dependent contracts. Example: `krew release --all --bump minor --apply`.
- **Developer mode** (inside a Krewire project, no `--module` flag) — bumps the project's `version:` in `krewire.yaml` and builds it. Example: `krew release --bump patch --apply`.

---

## 6. Code Style & Engineering Discipline

- **Idiomatic Go**: Clear names, `(value, error)` signatures, zero-value usability, and functional options where suitable.
- **Anti-God-Module Rule**: Each unit and package must have exactly one reason to change. Never bundle unrelated helpers, handlers, and registries into single monolithic files.
- **Stdlib-First**: Always prefer Go standard library packages over third-party dependencies.
- **SafePath Traversal Guard**: Any file system operation receiving user or workspace input must be bounded via `SafePath`.

---

## 7. Additional Resources

- [**`README.md`**](README.md): Core monorepo overview and package map.
- [**`docs/index.md`**](docs/index.md): Master documentation hub for `pkg/` and runtime guides.
- [**`docs/getting-started.md`**](docs/getting-started.md): Onboarding and local development setup.
