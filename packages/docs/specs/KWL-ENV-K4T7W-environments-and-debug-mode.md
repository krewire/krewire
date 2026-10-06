# Specification — Environments & Debug Mode

| Field       | Value                                        |
| ----------- | -------------------------------------------- |
| SpecID      | KWL-K4T7W                                    |
| Title       | Environments & Debug Mode                    |
| Status      | Draft                                        |
| Date        | 2026-08-23                                   |
| Author      | Krewire Contributors                          |
| Domain      | Core — Environment & Configuration           |

## 1. Context

Applications behave differently across development and production: log
verbosity, external services, and safety rails all change. The Krewire
ecosystem needs one shared vocabulary for the target environment and a debug
switch, declared in `krewire.yaml` (the only config file) so every workload —
app, CLI, site, book — reads the same truth.

## 2. Problem Statement

Today there is no first-class notion of environment or debug mode. Developers
invent ad-hoc env vars per project, devtool commands cannot tailor behavior,
and nothing in `kiw info` tells an operator which mode a build targets — making
production mistakes cheap to make and expensive to notice.

## 3. Goals

- G1 — One canonical environment type with exactly three values.
- G2 — Declare environment and debug in `krewire.yaml`.
- G3 — Deterministic override precedence for CI and local overrides.
- G4 — Surface the resolved values in `kiw` output and child processes.

## 4. Non-Goals

- NG1 — Additional environments (staging, preview); the set is closed and may
  grow via a new spec.
- NG2 — Per-environment config files or overlays (e.g. `krewire.prod.yaml`).
- NG3 — Framework-side consumption (`framework/app` reading KIW_DEBUG);
  tracked separately.
- NG4 — Secret management.

## 5. Requirements

| ID            | Requirement                                                                                                   | Priority | Scope    |
| ------------- | ------------------------------------------------------------------------------------------------------------- | -------- | -------- |
| KWL-ENVV-001  | `core.Env` admits exactly `local`, `production`, `testing`; empty resolves to `local`.                         | Must     | Package  |
| KWL-ENVV-002  | Parsing an unknown environment fails with a usage error naming the allowed set.                                | Must     | Package  |
| KWL-ENVV-003  | `krewire.yaml` accepts top-level `env:` and `debug:` keys, defaulting to `local` / `false`.                     | Must     | Domain   |
| KWL-ENVV-004  | Precedence is strictly: CLI flag > `KIW_ENV` / `KIW_DEBUG` > `krewire.yaml` > default.                            | Must     | Domain   |
| KWL-ENVV-005  | `kiw info` prints the resolved environment and debug state under Environment.                                   | Must     | Func     |
| KWL-ENVV-006  | `kiw run` and `kiw dev` export `KIW_ENV` and `KIW_DEBUG` into child-process environments.                          | Must     | Func     |
| KWL-ENVV-007  | An invalid resolved environment exits with usage code 2 and names the allowed set.                             | Must     | Func     |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Stdlib-only core.** `libs/core` remains dependency-free.
- NFR3 — **Determinism.** Identical inputs resolve identically across OSes.
- NFR4 — **Quality gates.** `gofmt -l .`, `go vet ./...`, `go test ./...`
  pass in every touched repo.

## 7. Success Criteria

- S1 — A project with `env: production` shows `Env production` in `kiw info`
  without any flag.
- S2 — `kiw run --env testing` starts the app with `KIW_ENV=testing` visible to
  the child process.
- S3 — `env: staging` (unknown value) fails fast with exit code 2.

## 8. Related Specifications

| SpecID      | Title                                   |
| ----------- | --------------------------------------- |
| KWL-M1ZKS   | Krewire Libraries — Initial Specification |
| KWL-W0J2X   | Core Errors & Exit Codes                |
| KWL-2X1QZ   | Configuration Loading (YAML + env overlay) |
| KWN-BNKJC   | Project Information                     |
