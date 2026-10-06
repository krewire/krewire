# Specification — Krewire-Native Guild Template

| Field       | Value                                      |
| ----------- | ------------------------------------------ |
| SpecID      | KWG-K2N7Q                                  |
| Title       | Krewire-Native Guild Template               |
| Status      | Done                                       |
| Date        | 2026-08-20                                 |
| Author      | Krewire Contributors                        |
| Domain      | Agents — Project Template                  |

## 1. Context

Krewire Guild ships a reusable AI agent setup (AGENTS.md, opencode.json, and the
`.agents/` preset of agents, commands, and skills) installed through
`kiw guild install`. Today the template is intentionally project-agnostic:
it teaches generic stack detection, generic workflows, and nothing about the
ecosystem it is installed into.

The ecosystem now defines distinct project shapes (`app`, `cli`, `site`,
`book`, and `--template` bootstrapping via `kiw init`), a single CLI
(`kiw`) that owns the whole workflow, one project config file
(`krewire.yaml`), canonical layouts, exit codes, per-repo quality gates, and a
spec-driven development process. This spec matures the installed template so
that it is **Krewire-native**: the agent detects the project kind, drives the
`krewire` command matrix, enforces ecosystem conventions, and follows the
spec-first workflow — instead of re-discovering generic tooling by manifest
guessing.

## 2. Problem Statement

An agent installed into a Krewire project has no usable Krewire guidance. It
guesses a generic test command, a generic init flow, and generic conventions.
Worse, it may invent things the ecosystem explicitly rejects — a separate
`ssg.yaml`, a wrong entry point, wrong routing — and it does not know the
exit-code contract, the quality gates per repo, or the spec-id conventions.
"Mature and optimize Guild for the basic needs of the various project types
built with this ecosystem" means turning the template's operating core
(constitution, kickoff/scout, init, spec-writing, quality-gate, ci-cd,
release, test) into a Krewire-native set while keeping the loader mechanics
(`opencode.json`, the `.agents/` layout) vendor-agnostic.

## 3. Goals

- G1 — The installed `AGENTS.md` is a Krewire-native constitution: project-kind
       detection, `kiw` command matrix, canonical layout, exit codes,
       quality gates, and the spec-driven workflow.
- G2 — Kickoff/scout map a Krewire project by **kind** (app/cli/site/book)
       automatically, not by generic manifest guessing.
- G3 — Commands and skills (project-init, spec-writing, quality-gate, ci-cd,
       release, test, new-project) use the `kiw` CLI and ecosystem
       conventions.
- G4 — Keep the template usable in non-Krewire projects by clearly scoping the
      Krewire-only guidance.
- G5 — Preserve determinism, idempotence, and the managed-file contract;
      module stays stdlib-only.

## 4. Non-Goals

- NG1 — Vendor-specific installers beyond the existing `opencode.json`.
- NG2 — Per-kind templating inside `Install` (no `WithKind`); the static
      constitution covers all kinds.
- NG3 — Rewriting generic best-practice files (dependency-audit,
      security-review, conventional-commit) into Krewire-only guides.

## 5. Requirements

| ID           | Requirement                                                                      | Priority |
| ------------ | -------------------------------------------------------------------------------- | -------- |
| GLD-ECO-001  | `template/AGENTS.md` is fully Krewire-native: kind-detection matrix, `kiw` command matrix, canonical layout, exit codes, quality gates, spec-driven workflow. | Must |
| GLD-ECO-002  | `template/AGENTS.md` documents the `kiw` command set (`version`, `info`, `new`, `init`, `build`, `serve`, `run`, `dev`, `test`, `deploy`, `guild`) with per-kind applicability. | Must |
| GLD-ECO-003  | The `scout` agent and `kickoff` skill detect the Krewire kind first (`project.kind`, `ssg:`, `content/`, `main.go`) and map commands from `krewire.yaml`. | Must |
| GLD-ECO-004  | The `project-init` skill scaffolds with `kiw new` + `kiw init --site\|--book\|--cli\|--template` before installing the guild. | Must |
| GLD-ECO-005  | The `quality-gate` skill enforces `gofmt -l`, `go vet ./...`, `go test ./...` plus per-kind build gates (`go build .` for app/cli; `kiw build` for site/book). | Must |
| GLD-ECO-006  | The `spec-writing` skill and `/spec` command follow ecosystem spec conventions (`docs/specs/`, `SpecID`, requirement IDs, mandatory metadata table, no in-file version history). | Must |
| GLD-ECO-007  | The `ci-cd` skill targets Go/Krewire: gofmt/vet/test gates, `kiw build`, docs/landing gh-pages deploy, tag-driven releases. | Must |
| GLD-ECO-008  | The `release` skill follows the ecosystem release flow: tag the owning repo, bump downstream `go.mod`, propagate the update ecosystem-wide, deploy docs/landing. | Must |
| GLD-ECO-009  | Managed-file paths and install semantics are unchanged; quality gates pass in the guild and krewire repos. | Must |
| GLD-ECO-010  | Krewire-only guidance in `template/AGENTS.md` is clearly scoped so non-Krewire projects remain usable. | Should |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Determinism.** Same input + options produce the same reported paths and the same bytes.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.
- NFR5 — **No deps.** The module may only import from the Go standard library.

## 7. Success Criteria

- S1 — Installing into a scaffolded `app` project yields an AGENTS.md that names
       `kiw run`/`kiw dev` and the canonical layout.
- S2 — Installing into `site`/`book` projects yields an AGENTS.md referencing
       `kiw build`/`kiw serve` and `ssg:`/`content/` detection.
- S3 — `/kickoff` in any Krewire project reports the kind and the `kiw`
       commands without manual generic discovery.
- S4 — The guild module's quality gates pass; the krewire test suite keeps
      passing.

## 8. Related Specifications

| SpecID    | Title                                                |
| --------- | ---------------------------------------------------- |
| [KWG-P9ZT4](./KWG-INSTALL-P9ZT4-guild-module-install.md) | Guild Module & Install Library |
| [KWN-MZ4LE](../../../krewire/docs/specs/KWN-GUILD-MZ4LE-guild-install-command.md) | krewire guild install Command |
| [KWN-7QM2X](../../../krewire/docs/specs/KWN-INIT-7QM2X-init-project-variants.md) | Init Project Variants |
| [KWN-RD3WS](../../../krewire/docs/specs/KWN-SCAFFOLD-RD3WS-project-scaffolding.md) | Project Scaffolding (minimal kernel) |

## 9. References

- [kiw](https://github.com/krewire/kiw) — the CLI that consumes this module.
- [guild](https://github.com/krewire/guild) — this module.