# Contribution & Community Guide

Guidelines for contributing to the Krewire core monorepo and its constituent packages.

---

## 1. Ground Rules

- **English for Stored Artifacts**: Code, documentation, PR titles, and commit messages must be written in English.
- **Conventional Commits**: Format commit messages as:
  ```text
  feat(web): add websocket HMR handler (ref #12, req KWF-HMR-001)
  fix(cli): handle missing config gracefully
  docs(foundation): clarify layering invariants
  ```
- **Atomic Commits**: Keep changes atomic ($\le$ ~300 LOC per commit).

---

## 2. Development Workflow

1. Fork and clone the repository.
2. Build the local CLI:
   ```bash
   go build -o bin/kiw ./tools/kiw/cmd/kiw
   ```
3. Make atomic changes adhering to Single Responsibility Principle (SRP).
4. Run all quality gates:
   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   go build -o bin/kiw ./tools/kiw/cmd/kiw
   ```
5. Submit a pull request referencing the relevant GitHub Issue.
