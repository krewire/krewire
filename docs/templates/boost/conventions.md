# Boost Agent Conventions & Safety Boundaries

Engineering rules, commit standards, and behavioral guardrails that autonomous agents must enforce across all Krewire projects.

---

## 1. Code Style & Architectural Invariants

- **Standard Library First**: Always prefer Go standard library packages (`net/http`, `crypto`, `context`, `os`, `io/fs`) over third-party dependencies.
- **Strict Downward Layering**: Dependencies flow only downward (`Layer N → Layer < N`). Higher layers must never be imported by lower layers.
- **Single Responsibility Principle (SRP)**: Each unit and package has exactly one reason to change. Decompose god-classes and oversized files into cohesive sub-packages.
- **Preserve Existing Documentation**: Never delete or overwrite existing comments, ADRs, or docstrings unrelated to current edits.
- **Zero Speculative Abstractions**: Introduce interfaces and abstractions only when a concrete second use case requires it.

---

## 2. Commit Discipline & Automation

- **Conventional Commits**: Format commit messages strictly as:
  - `feat(<scope>): description`
  - `fix(<scope>): description`
  - `docs(<scope>): description`
  - `refactor(<scope>): description`
  - `test(<scope>): description`
  - `chore(<scope>): description`
- **Atomic Commits**: Keep changes granular ($\le 300$ lines of code per commit).
- **Issue Traceability**: Always reference corresponding GitHub Issue IDs (e.g. `feat(auth): add PKCE verification (ref #12)`).
- **Automated Commit Execution**: Once quality gates pass, stage and commit atomically without waiting for extra user confirmations.

---

## 3. Defensive Safety Guardrails

- **Zero Hardcoded Secrets**: Never commit or log API keys, private certificates, or passwords. Always use environment variables (`${env:KEY_NAME}`).
- **Path-Traversal Guard (`SafePath`)**: All file system operations with untrusted input must be sanitized via `SafePath` to prevent directory traversal attacks (CWE-22).
- **Prohibited Destructive Actions**: Agents must never run unconfirmed destructive commands (`rm -rf /`, force-pushing `git push --force`, or rewriting shared history).
- **PII Redaction**: Personal identifying information (passwords, JWTs, credit card numbers) must be masked in log outputs using `sec.NewPIIHandler`.
