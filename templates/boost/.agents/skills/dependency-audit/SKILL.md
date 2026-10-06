---
name: dependency-audit
description: Use when asked to check the health, updates, or vulnerabilities of project dependencies. Trigger on "dependency", "audit deps", "outdated", "update packages", "vulnerable".
---

# Dependency Audit

Assess the state of a project's dependencies and propose safe updates.

## 1. Detect the ecosystem

- Node: `package.json` / lockfile (`package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`).
- Python: `requirements.txt`, `pyproject.toml`, `Pipfile.lock`.
- Go: `go.mod` / `go.sum`.
- Rust: `Cargo.toml` / `Cargo.lock`.
- Other: use the matching manifest.

## 2. Check for vulnerabilities

Run the ecosystem's audit tool:

- Node: `npm audit` / `yarn audit` / `pnpm audit`
- Python: `pip-audit` or `pip check`
- Go: `govulncheck ./...`
- Rust: `cargo audit`

## 3. Check for outdated packages

- Node: `npm outdated` / `yarn outdated`
- Python: `pip list --outdated`
- Go: `go list -m -u all`
- Rust: `cargo outdated` (if installed)

## 4. Analyze & propose

- Separate findings into: critical (fix now) vs. major vs. minor/optional.
- For each, give the impacted package, the risk, and a concrete upgrade path.
- Do NOT upgrade blindly. Prefer compatible-range bumps unless a breaking upgrade is justified. Note breaking changes that alter the upgrade's scope.
- Never assume the project's CI or runtime accepts an upgrade — verify the tests after any change.

## 5. Verify

If you apply updates, run the project's test/lint/build afterwards and report results.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.