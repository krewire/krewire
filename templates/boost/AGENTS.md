# Krewire Guild — Agent Constitution

This document is the primary operating ruleset for the AI agent in projects
that install this template. It is tuned for the **Krewire ecosystem** (Go
projects driven by the `kiw` CLI). If the current project is not a Krewire
project, follow the generic principles below and skip the Krewire-specific
sections marked *Krewire*.

## Language

- **Use English in all agent output** — responses, reports, summaries, commit messages, logs.
- The human may write to you in Indonesian (or any language); always reply in English.
- Code, identifiers, and documentation follow the project's own conventions.

## Mission

Help developers get work done correctly, quickly, and safely — without breaking
what already works. When facing an unfamiliar project, never assume: map it out
before acting.

## Identify the Project Kind (Krewire)

The Krewire ecosystem has exactly one config file, `krewire.yaml`, and eight
project shapes covering the full web-service spectrum. Detect which one you are
in before doing anything:

| Kind      | Detection                                                   | Driven by                              |
| --------- | ----------------------------------------------------------- | -------------------------------------- |
| `app`     | `project.kind: app` or a root `main.go` (fullstack monolith)| `kiw run`, `kiw dev`             |
| `cli`     | `project.kind: cli`                                         | `kiw run <args...>`                 |
| `site`    | `project.kind: site` or an `ssg:` key in `krewire.yaml`     | `kiw build`, `kiw serve`         |
| `book`    | `project.kind: book` or a `content/` directory            | `kiw build`, `kiw serve`         |
| `worker`  | `project.kind: worker` (job queues, cron, retries)          | `kiw run`, `kiw worker`, `kiw dev` |
| `service` | `project.kind: service` (microservice)                      | `kiw run`, `kiw dev`             |
| `infra`   | `project.kind: infra` (cloud IaC)                           | `kiw deploy`, `kiw build --plan` |
| kernel    | only `go.mod`, `krewire.yaml`, `main.go`, `.gitignore` (pre-`init`) | `kiw init` |

Validate with `kiw info` (prints the detected kind). Never guess: read
`krewire.yaml` and check for `content/` before deciding.
See the unified vision: [`KWF-M8K2Q`](framework/docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md).

## The `kiw` Command Matrix (Krewire)

`kiw` is the single entry point for the whole ecosystem. Never invoke
project-specific `cmd/` binaries for build/serve/run.

| Command | Purpose | Works on |
| ------- | ------- | -------- |
| `kiw new <name>` | Scaffold a minimal kernel (go.mod, krewire.yaml, main.go, .gitignore) | any new project |
| `kiw init` | Equip a kernel in place (default: fullstack app) | kernel |
| `kiw init --site` | Equip a declarative static site (`ssg:` in krewire.yaml) | kernel |
| `kiw init --book` | Equip a markdown content book (mdbind) | kernel |
| `kiw init --cli` | Equip a command-line application (framework/tui) | kernel |
| `kiw init --template <git-url>` | Clone a starter repository | empty dir |
| `kiw build` | Build the project (binary for app/cli/worker/service, `.krewire/build` for site/book, plan for infra) | all |
| `kiw serve` | Start the project locally for any kind: compile & listen (`app`), execute with args (`cli`), preview static output (`site`, `book`) | all |
| `kiw run [args...]` | Build and run the app/CLI/worker/service binary | app, cli, worker, service |
| `kiw dev` | Rebuild + auto-restart on change (incl. WASM for frontend) | app, cli, worker, service |
| `kiw worker` | Run background workers / job queues | worker |
| `kiw deploy` | Provision infra + deploy (`--plan`, `--preview`, `--destroy`) | app, site, book, worker, service, infra |
| `kiw dashboard` | Local dev dashboard (services, logs, traces, infra) | worker, service, infra |
| `kiw generate` | Generate code (OpenAPI, config, etc.) | all |
| `kiw test` | Run `go test ./...` of the current module (spawn Go toolchain) | all |
| `kiw vet` | Run `go vet ./...` of the current module (spawn Go toolchain) | all |
| `kiw fmt` | Check/format with `gofmt -l` / `go fmt ./...` (spawn Go toolchain, `--write` to fix) | all |
| `kiw info` | Print environment and detected project kind | all |
| `kiw version` | Print CLI and framework versions | all |
| `kiw boost install` | Install this template | any project |
| `kiw help <command>` | Show help for a command | all |
| `kiw <command> help` / `kiw <command> --help` / `kiw <command> -h` | Show help for a command (aliases) | all |

## Core Conventions (Krewire)

- **Config lives in `krewire.yaml` only.** There is no `ssg.yaml`; a declarative
  site uses the `ssg:` key inside `krewire.yaml`.
- **Entry point** — `app` and `cli` projects build from the root `main.go`
  (`go build .`). `site`/`book` projects have **no entry point**.
- **Canonical layout (app)** — `main.go` (thin entry), `internal/` (app
  assembly, config, http), `web/` (layouts, pages, theme), `assets/` or
  `public/` (embedded static assets).
- **CLI layout** — `main.go` (tui.App harness) + `internal/commands/`.
- **Site layout** — a `ssg:` key with `layouts`, `components`, `pages`; output
  to `.krewire/build` (default, configurable via `output` in `krewire.yaml` or `--output`/`-o` flag); links extensionless, each page emitted as a sibling `.html`
  file (file-based routing).
- **Book layout** — `content/**/*.md` chapters (subdirectories become
  chapters with subchapters), assembled by mdbind; `input` configurable in
  `krewire.yaml` (legacy `manuscript/` still accepted); README/readme notes
  are excluded by default, tunable via `build.include/exclude`.
- **Worker layout** — `worker:` key in `krewire.yaml`, jobs under `internal/worker/`, queues via `framework/worker`.
- **Service layout** — `service:` key, registry/config/gateway/resilience via `framework/service`; modular monolith default, opt-in extraction.
- **Infra layout** — `infra:` key, provider-agnostic declarations under `infra/` compiled by `framework/infra` to AWS/Kubernetes.
- **Config struct** — typed `krewire.yaml` structs loaded with `libs/config`,
  validated with `libs/validation` (`validate:"required"` tags) and business rules in `libs/core` (`Kind`/`Workload`/`SpecID`).
- **Exit codes** — `0` success, `1` runtime failure, `2` usage error
  (`libs/core.ExitCodeSuccess/Failure/Usage`).
- **Control plane** — `libs/core` (declarative: business rules, workload registry) + `libs/kern` (imperative: `Kernel`/`Module`/`Registry`/`Executor`/`Supervisor`) are the ecosystem center; `framework` and `kiw` compose via `kern`.
- **Modules** — `github.com/krewire/framework` (unified framework: `tui`, `web`+`ssg`, `ui`, `app`, `runtime`, `worker`, `service`, `infra`), `github.com/krewire/libs` (`core`+`kern`+`config`/`validate`/`term`), `github.com/krewire/mdbind`,
  `github.com/krewire/boost`. Cross-repo testing uses the hub `go.work`
  workspace; temporary `replace` directives only for single-repo clones outside
  the workspace.

## Optional — The Krewire Way (Spec-Driven Development)

This template is project-agnostic and imposes no development methodology.
Teams that like how the Krewire ecosystem itself is built may opt in to its
spec-driven approach ("the Krewire way"):

- Before building a complex feature, write a short specification in
  `docs/specs/`: problem background, identified need, and requirement rows
  with IDs (e.g. `APP-BLD-001`) and priorities (`Must`/`Should`).
- File name `{ProjectId}-{Scope}-{SpecID}-{slug}.md`; SpecID is a unique
  random 5-character code. Metadata table: `SpecID`, `Title`, `Status`,
  `Date`, `Author`, `Domain`. Revision history lives in git, not in the file.

Anything smaller can stay README-driven — adopt only what pays for itself.

## Workflow

1. **Understand** the request. Ask if ambiguous — better to ask than to go the wrong way.
2. **Plan** — for non-trivial tasks, make a short plan (this may use the
   `plan` agent, or a spec first for projects that adopted the Krewire way).
3. **Implement** — make the smallest change following conventions.
4. **Verify** — run the relevant tests/lint/build. Never finish a task without verification.
5. **Summarize** — report what changed, why, and how it was verified.

## Mapping the Project

Run `/kickoff` (or `kiw info`) when the project is unmapped. Record: detected
kind, the `kiw` commands it needs, layout, and conventions. Don't make
another agent re-map from scratch.

## Edit Discipline

- Use precise edit tools; avoid replacing whole files unnecessarily.
- Do not add comments unless asked — clear code beats comments.
- Do not change code unrelated to the task.
- Be careful with destructive operations (`rm`, force-push, overwrite) — ask for confirmation when unsure.

## Quality Gates

- Krewire Go projects: `gofmt -l .`, `go vet ./...`, `go test ./...` in each repo.
- Per-kind build gate — `app`/`cli`: `go build .`; `site`/`book`:
  `kiw build` then spot-check the `.krewire/build` output.
- Run the real commands and record results; do not claim a gate passes without evidence.

## Testing

- `kiw test` (Go: `go test ./...`). Find repo-local test patterns first.
- After changing behavior, add/update tests when the project's patterns expect it.
- If tests fail, use the `debugger` agent for root-cause analysis instead of trying random fixes.

## Commit

Use **Conventional Commits**:

```
<type>(<scope>): <short description>

feat:     new feature
fix:      bug fix
refactor: change without behavior change
docs:     documentation
test:     tests
chore:    other supporting work
```

Commit focuses on one logical change. Do not commit unrelated files or secrets.
Before committing, check `git status`/`git diff` and never commit secrets.
**Do not commit large changes in a single commit** — split large changes into multiple small, atomic commits, each with a clear scope and message. Large commits make review difficult and increase risk of conflicts.

## Documentation

- Update README/docs when public behavior changes.
- For complex features/design, projects following the Krewire way capture the
  design as a spec in `docs/specs/` before implementing.

## Project-Specific Customization

Add rules specific to this project below here: agreed architecture, concrete
naming conventions, specific test commands, compatibility targets, or the
`kiw` commands this project needs. Keep them minimal and verifiable.