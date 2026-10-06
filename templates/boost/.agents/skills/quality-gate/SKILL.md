---
name: quality-gate
description: Use to define or run the quality gates for a Krewire project before declaring a change done. Triggers: "quality gate", "gates", "define done", "is it done", "check quality", "verify", "lint", "test everything".
---

# Quality Gate (Krewire)

Gates are binary, measurable checks that must pass before work is considered done.
Run the real commands and record the concrete command and result (pass/fail).

## 1. Run the gates

For every Go repo in a Krewire project — optimize by running independent gates in parallel:

1. **Format** — `gofmt -l .` (must output nothing).
2. **Vet** — `go vet ./...`.
3. **Test** — `go test ./...` (or `krewire test`). Use fakes/in-memory for `infra`/`service`/`worker`; live backends behind build tags only.
4. **Per-kind build gate** (vision-aware):
   - `app`/`cli`/`worker`/`service` — `go build .` (root `main.go` must compile).
   - `site`/`book` — `krewire build`, then spot-check `site/` (`curl` output readable without JS).
   - `infra` — `krewire build --plan` (or `krewire deploy --plan`) then review plan (pure, idempotent).
   - `runtime` — `krewire build` produces `site/_assets/runtime.*.wasm`; verify ≤ 800KB gzipped and hydration parity.

All gates reference `docs/specs/index.md` (Impl Status) — planned specs (`🔜`) require spec-first before code.

## 2. Spec traceability

When the repo tracks spec requirement IDs, verify each new behavior has a
requirement row traced spec → implementation → test. A feature with no
requirement ID and no test is not done.

## 3. Report

Flag each failed gate with evidence and what is needed to pass. Do not claim a
phase is done while any gate for it is red.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.