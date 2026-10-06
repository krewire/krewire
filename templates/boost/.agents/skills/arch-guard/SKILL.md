---
name: arch-guard
description: Guard Krewire architecture — enforce control plane (libs/core + libs/kern), workload matrix, module boundaries, and dependency rules. Triggers: "arch", "architecture", "guard", "boundary", "violation", "core/kern", "workload".
---

# Arch Guard — Architecture Enforcement

Prevent architectural drift before it ships. This skill enforces the control plane and unified workload matrix.

## 1. Load the control plane

- Read `docs/project-vision.md` (workload matrix) + `libs/core/workload.go` (`core.Kind`, `core.Workloads`) + `libs/kern/kern.go` (`Kernel`, `Module`, `Registry`).
- Check `docs/specs/index.md` for `Spec vs Impl Status` — planned specs must not be imported as code.
- Remember: `libs/core` is declarative (what is valid, stdlib-only); `libs/kern` is imperative (how it runs, stdlib + `core`). See `KWL-K1N2Q`, `KWL-KERN-X8P3L`, `KWF-M8K2Q`.

## 2. Run the 7 guard checks

| # | Check | Tool | Failure |
|---|-------|------|---------|
| 1 | Workload integrity | `core.ParseKind`, `core.WorkloadFor` | Unknown `project.kind` must return `core.ExitCodeUsage` |
| 2 | Control plane boundaries | `go list -f '{{.Imports}}' ./...` | `libs/core` imports `framework`/`krewire` → blocker; `libs/kern` imports `framework` → blocker |
| 3 | Framework package boundaries | `grep -r framework/` vs `docs/specs/index.md` | `framework/app` importing `service`/`infra` without `service`/`infra` kind → opt-in violation (`core.IsOptIn`) |
| 4 | Spec traceability | `ls docs/specs/` vs `docs/specs/` | New feature without spec in `docs/specs/` + requirement row (`FRK-*`) → blocker; original `docs/specs/` contains specs not `MOVED.md` → blocker |
| 5 | Single config | `ls ssg.yaml` | Presence of `ssg.yaml` → violation (`core.ValidateKrewireYamlPath`) |
| 6 | Module structure | `ls framework/` vs `framework/docs/architecture.md` | Missing `tui` (not `cli`), `runtime`/`worker`/`service`/`infra` layout drift → major |
| 7 | Version compatibility | `core.ParseVersion` + `core.CheckEcosystemCompatibility` | `go.mod` requires violate `core.EcosystemVersions` → major |

## 3. Fix workflow

1. Run `arch-guard` subagent (read-only) to list violations with `file:line`.
2. For each blocker, create an issue via `issue-writing` skill or fix directly: move business rule to `libs/core`, move executor to `libs/kern`, or move module registration to `kern.Registry`.
3. Re-run `go vet ./...` and `core.IsOptIn` checks.

## 4. Gates

- All 7 checks must be `Pass` before `reviewer` approves.

## Rules

Read `rules/key-rules.md` before acting.
