# Boost Agent Conventions & Safety Boundaries

Engineering rules and behavioral guardrails that agents enforce across all projects.

---

## 1. Code Style & Integrity

- **Preserve Documentation**: Preserve existing comments and docstrings unrelated to current edits.
- **Single Responsibility Principle**: Favor small, focused files and functions. Clear names over cleverness.
- **Zero Speculative Abstraction**: Introduce abstractions only when a second concrete use case emerges.

---

## 2. Commit Discipline

- **Conventional Commits**: Format commit messages as `feat|fix|docs|test|refactor|perf|security|chore(scope): desc`.
- **Atomic Commits**: Keep changes atomic ($\le$ ~300 LOC per commit).
- **Issue Reference**: Always reference corresponding GitHub Issue numbers and requirement IDs.

---

## 3. Safety Boundaries

- **Destructive Operations**: Prohibit unconfirmed destructive commands (`rm -rf`, force-push, recursive delete).
- **Secrets Management**: Never log, display, or commit tokens, passwords, private keys, or API secrets.
- **PII Protection**: Mask personal identification information in transcripts and diagnostic logs.
