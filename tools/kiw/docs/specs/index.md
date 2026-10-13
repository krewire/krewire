# Specifications Index — Krewire Devtool

This directory holds the formal specifications for the krewire devtool.

Ordered by **impact-to-effort** (high impact, low effort first) and **dependency chain** (foundations first).

| SpecID    | Title                                      | Status | Depends On |
| --------- | ------------------------------------------ | ------ | ---------- |
| [KWN-Z0VFC](https://github.com/krewire/krewire/issues/24)      | Krewire Devtool — Initial Specification | Draft | — |
| [KWN-6K41E](https://github.com/krewire/krewire/issues/30) | krewire run/dev/deploy              | Draft | KWN-Z0VFC |
| [KWN-SCRIPT-9F3KQ](https://github.com/krewire/krewire/issues/32) | kiw Script Runner — `kiw run <task>` and `kiw run file.go` | Draft | KWN-6K41E, KWN-Q7X4M, KWN-Z0VFC |
| [KWN-RD3WS](https://github.com/krewire/krewire/issues/31) | Project Scaffolding                  | Draft | KWN-Z0VFC |
| [KWN-7QM2X](https://github.com/krewire/krewire/issues/28) | Init Project Variants                    | Draft | KWN-RD3WS |
| [KWN-1QGI2](https://github.com/krewire/krewire/issues/20)   | Project Building                      | Draft | KWN-Z0VFC |
| [KWN-MLTB](https://github.com/krewire/krewire/issues/21) | Multi-Runtime Build Orchestration      | Draft | KWN-1QGI2, KWN-CONF-MRCN |
| [KWN-BNKJC](https://github.com/krewire/krewire/issues/26) | Project Information                  | Draft | KWN-Z0VFC |
| [KWN-P0FWA](https://github.com/krewire/krewire/issues/33)  | Project Validation                   | Draft | KWN-Z0VFC |
| [KWN-JB7PW](https://github.com/krewire/krewire/issues/27)   | Version Reporting                     | Draft | KWN-Z0VFC |
| [KWN-MZ4LE](https://github.com/krewire/krewire/issues/25) | krewire guild install Command        | Draft | KWN-Z0VFC |
| [KWN-Q3M8V](https://github.com/krewire/krewire/issues/34) | Workspace Command — Multi-Repo & Monorepo Workflows | Draft | KWN-Z0VFC |
| [KWN-Q7X4M](https://github.com/krewire/krewire/issues/23) | Config Directory & Dotenv (provider → config/ → .env → krewire.yaml) | Draft | KWL-K4T7W, KWL-2X1QZ, KWN-RD3WS |
| [KWN-CONF-MRCN](https://github.com/krewire/krewire/issues/22) | Multi-Runtime Unified Configuration | Draft | KWN-Q7X4M, KWN-RD3WS |
| [KWN-L4RVE](https://github.com/krewire/krewire/issues/18) | Krewire × Laravel Coexistence & Integration | Draft | KWN-GUILD-MZ4LE, KWN-RD3WS, KWN-RUN-6K41E, KWF-L5H2F |
| [KWN-NDJ5S](https://github.com/krewire/krewire/issues/19) | Krewire × Node.js Ecosystem Integration | Draft | KWN-GUILD-MZ4LE, KWN-RD3WS, KWN-RUN-6K41E, KWF-T4X9P, KWF-SVC-L5H2F |
| [KWN-PKG-7X9K2](https://github.com/krewire/krewire/issues/29) | kiw add/remove — scalable package@version (plugin → gomod → npm) | Draft | — |

## Conventions

- Each specification is stored as a single Markdown file named `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
- SpecIDs are unique, random 5-character alphanumeric codes (e.g., `KWN-Z0VFC`).
- New specifications must be added to this index when created.
- Ordering: impact-to-effort (high impact, low effort first), then dependency chain (foundations first).