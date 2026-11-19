# Contribution & Community Guide

Guidelines for contributing code, documentation, and architectural improvements to the Krewire core monorepo (`github.com/krewire/krewire`).

---

## 1. Ground Rules & Principles

- **English for Stored Artifacts**: All code, documentation, specs, commit messages, and PR descriptions must be written in English.
- **Single Source of Truth (SSOT)**:
  - Active tasks, features, and fixes are tracked as GitHub Issues under `krewire/krewire`.
  - Every issue body contains a structured checklist with `- [ ]` and `- [x]` checkboxes.
  - Commits and pull requests must reference the relevant issue number (`ref #<IssueNumber>`).
- **Conventional Commits**: Format commit messages according to the Conventional Commits specification:
  ```text
  feat(scope): add new feature (ref #12)
  fix(scope): resolve bug in component (ref #34)
  docs(scope): update API reference (ref #56)
  test(scope): add unit test coverage (ref #78)
  refactor(scope): optimize layout logic
  chore(scope): bump dependency versions
  ```
- **Atomic Commits**: Keep commits focused and atomic ($\le$ ~300 LOC per commit). Never mix formatting changes with functional logic.

---

## 2. Architectural Invariants (Do Not Violate)

Strict downward-only dependencies are strictly enforced across the codebase:

```text
templates/boost → tools/kiw → pkg/* → pkg/model, pkg/errs → Go Stdlib
```

1. **`pkg/model` & `pkg/errs` Boundary**: Sits at Layer 1. Must never import higher-level packages or external frameworks.
2. **Complete Tooling Isolation**: No library under `pkg/` may ever import `tools/kiw` or `github.com/krewire/kiw`.
3. **No Duplicate Literals**: Constants and configuration keys must be declared once in their owning package (e.g., `model.ConfigFileName = "krewire.yaml"`).
4. **Pure Go / Zero CGO**: All code must build cleanly with `CGO_ENABLED=0`.

---

## 3. Contribution Workflow

1. **Discover or Create an Issue**:
   Find an open issue on GitHub or create a new one outlining the proposal and acceptance criteria checklist:
   ```bash
   gh issue list -R krewire/krewire --state open
   ```

2. **Fork and Clone**:
   ```bash
   git clone https://github.com/krewire/krewire.git
   cd krewire
   ```

3. **Build the CLI**:
   ```bash
   go build -o bin/kiw ./tools/kiw/cmd/kiw
   ```

4. **Make Changes**:
   Implement the feature or fix adhering strictly to the package's layer boundaries. Add unit tests in colocated `*_test.go` files.

5. **Run Mandatory Quality Gates**:
   ```bash
   gofmt -l .        # Must print nothing
   go vet ./...      # Must report no errors
   go test ./...     # All tests must pass
   go build -o bin/kiw ./tools/kiw/cmd/kiw # CLI must compile cleanly
   ```

6. **Submit Pull Request**:
   Push to your branch and submit a PR referencing the issue. Ensure all CI checks pass.
