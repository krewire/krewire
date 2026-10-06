# Specification — Guild Module & Install Library

| Field       | Value                                      |
| ----------- | ------------------------------------------ |
| SpecID      | KWG-P9ZT4                                  |
| Title       | Guild Module & Install Library             |
| Status      | Draft                                      |
| Date        | 2026-08-20                                 |
| Author      | Krewire Contributors                        |
| Domain      | Agents — Template & Tooling                |

## 1. Context

Krewire Guild ships a reusable AI agent setup (AGENTS.md, opencode.json, and a
`.agents/` preset of agents, commands, and skills) installable into any
software project. Historically it was distributed as a tarball-plus-shell
script (`scripts/install.sh`) that copied files by hand.

The Guild is now part of the Krewire ecosystem: `krewire/krewire` is the single
CLI entry point for the whole ecosystem. To follow that pattern, the Guild
template must be distributed as a Go module (`github.com/krewire/guild`) that
exposes the template as an embedded filesystem and provides a small, typed
install library. The CLI (`krewire guild install`) becomes the only installation
surface; the hand-rolled `install.sh` is retired.

## 2. Problem Statement

The shell installer is untyped, untested, and duplicated logic that cannot be
validated with `go test`. It couples installation to a file-system checkout of
this repository and cannot be consumed by the ecosystem CLI. There is no
canonical, testable contract for "what does installing Guild into a project
write and refuse to overwrite".

## 3. Goals

- G1 — Distribute the Guild template as a Go module `github.com/krewire/guild`.
- G2 — Expose the template as an `embed.FS` so CLI binaries are self-contained.
- G3 — Provide an idiomatic, typed install API with functional options.
- G4 — Preserve the managed-file semantics of the current installer
      (AGENTS.md, opencode.json, `.agents/`), including refusal to overwrite
      without `--force`.
- G5 — Fully retire `scripts/install.sh` and its scripts directory.

## 4. Non-Goals

- NG1 — Interactive prompts or wizard flow (owned by the krewire CLI, KWN spec).
- NG2 — Validating the content of agent files beyond existence/ownership.
- NG3 — Installing on behalf of specific AI vendors (OpenCode, Claude, etc.).
- NG4 — Any runtime dependency beyond the Go standard library.

## 5. Requirements

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| GLD-IN-001 | Declare module path `github.com/krewire/guild` (Go 1.22, stdlib only). | Must   |
| GLD-IN-002 | Embed the installable template tree (AGENTS.md, opencode.json, `.agents/`) and expose it as `embed.FS`. | Must |
| GLD-IN-003 | Provide `Install(target string, opts ...Option) ([]string, error)`; return the created paths deterministically. | Must |
| GLD-IN-004 | Support the `WithForce()` option, overwriting existing managed files atomically. | Must   |
| GLD-IN-005 | Support the `WithDryRun()` option: validate and report without writing.      | Must |
| GLD-IN-006 | Reject a missing/empty target directory with a typed sentinel error.          | Must |
| GLD-IN-007 | Refuse to install when a managed file exists and neither `WithForce()` nor `WithDryRun()` is set, without writing anything. | Must |
| GLD-IN-008 | Install must create parent directories as needed (`opendir` style copy).      | Must |
| GLD-IN-009 | Copy preserves file modes for executable members of the tree.                  | Should |
| GLD-IN-010 | Expose `Managed()` returning the canonical managed top-level paths.            | Should |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Determinism.** Same input + options produce the same reported paths and the same bytes.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.
- NFR5 — **No deps.** The module may only import from the standard library.

## 7. Success Criteria

- S1 — A consumer can `go get github.com/krewire/guild` and read the template via `guild.Template`.
- S2 — `Install` into a fresh empty directory writes all managed paths; re-running without force returns a typed error and writes nothing.
- S3 — `WithDryRun` reports would-be paths without touching the target.
- S4 — `scripts/install.sh` no longer ships.

## 8. Related Specifications

| SpecID    | Title                                                |
| --------- | ---------------------------------------------------- |
| [KWG-P9ZT4](./KWG-INSTALL-P9ZT4-guild-module-install.md) | Guild Module & Install Library (this spec) |
| KWN (krewire) | guild install command specification (companion, written in krewire/krewire) |
| [KWF-5XJFC](https://github.com/krewire/framework/blob/main/docs/specs/KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model |

## 9. References

- [krewire](https://github.com/krewire/krewire) — the CLI that consumes this module.
- [guild](https://github.com/krewire/guild) — this module.