---
name: ci-cd
description: Use to set up or fix continuous integration / continuous delivery for a project. Triggers: "CI", "CD", "GitHub Actions", "GitLab CI", "pipeline", "automate build and tests", "generate a pipeline".
---

# CI/CD Setup

Create a reliable, fast CI/CD pipeline that runs on every change and ships releases deliberately.

## 1. Detect the stack and existing setup

- Read `go.mod` (Go) and the project kind (`krewire.yaml`: app/cli/site/book/worker/service/infra), existing CI config in `.github/workflows/`, `.gitlab-ci.yml`, `azure-pipelines.yml`, etc.
- Note the exact commands the project already has. For Krewire Go projects these come from AGENTS.md Quality Gates: `gofmt -l .`, `go vet ./...`, `go test ./...`; per-kind build gate `go build .` (app/cli/worker/service) or `krewire build` (site/book) or `krewire build --plan` (infra).

## 2. Pick the provider

- Default: GitHub Actions (`.github/workflows/`) unless the project is hosted elsewhere. Match the platform the team already uses.

## 3. Pipeline design (CI)

Run on push/PR, gated and fast:

1. **Setup** — checkout, `actions/setup-go` with the go-version from `go.mod` plus a Go module cache.
2. **Quality** — `gofmt -l .` (must output nothing) and `go vet ./...`.
3. **Test** — `go test ./...`. Add coverage reporting if configured.
4. **Build** — `go build .` for app/cli/worker/service; `krewire build` for site/book; `krewire build --plan` (or `krewire deploy --plan`) for infra.
5. **Security** — Go dependency audit (`govulncheck` or `go vet`-based audit; see `dependency-audit` skill) on a schedule or on release, not on every push by default.

Keep CI under ~10 minutes where possible: cache aggressively, split matrix jobs only when the stack truly needs it, avoid installing everything when a subset suffices.

## 4. Release stage (CD)

- Build release artifacts on tag or on push to the default branch (`go build` / `krewire build` / `krewire deploy --plan` for infra).
- Docs-based repos (`docs/`, `krewire.github.io`) publish to `gh-pages`; other Krewire repos ship tagged Go modules consumed via downstream `go.mod` `require` directives.
- Attach artifacts, then deploy via the environment's approved path (see `deploy` agent and `release` skill).
- Protect production: required status checks, branch protection, and an approval step for prod environments.

## 5. Verify

- Never claim a pipeline works without testing it: push a branch/PR and confirm the jobs run green (or provide exact steps to trigger it).
- Report the files created/edited and the expected trigger behavior.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.