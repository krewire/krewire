---
name: conventional-commit
description: Use when creating commit messages so commits follow the Conventional Commits spec. Trigger on "commit", "commit message", "write a commit".
---

# Conventional Commits

Standardize commit messages for a clear, machine-readable history.

## Format

```
<type>(<optional scope>): <short summary>

<optional body>

<optional footer(s)>
```

## Types

- `feat` — a new feature
- `fix` — a bug fix
- `refactor` — a code change that neither fixes a bug nor adds a feature
- `docs` — documentation only
- `test` — adding or correcting tests
- `chore` — other supporting changes (deps, build tooling, misc)
- `perf` — a performance improvement
- `style` — formatting, whitespace, no logic change
- `build` / `ci` — build system or CI config changes
- `revert` — reverts a previous commit

## Rules

Read `rules/key-rules.md` before acting and follow those rules.