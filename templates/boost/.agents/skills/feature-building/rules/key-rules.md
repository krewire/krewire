# Feature Building — Key Rules

- **Spec-first is law.** No code before `Approved` spec with requirement IDs (`FRK-*`/`KWL-*`).
- **Traceability mandatory:** every `Must` row → spec → implementation file:line → test file:line.
- **Control plane:** `libs/core` for types/rules, `libs/kern` for Kernel/Supervisor; `framework`/`krewire` compose via `kern`.
- **Versioning:** bump `core.Version` or module `Version` per `libs/core/version.go` semver; update `EcosystemRequires` and `go.mod` downstream.
- **Per-kind gates must pass:** `go build .` (`app`/`cli`/`worker`/`service`), `krewire build` (`site`/`book`), `krewire build --plan` (`infra`), hydration parity (`runtime`).
- **Traceability matrix:** Spec row → implementation file:line → test file:line in spec.
- **Docs sync:** `sync-docs` In-sync, `arch-guard` Pass required before done.