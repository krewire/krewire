# Declarative Core Model (`pkg/model`)

The `pkg/model` package defines pure data models, the workload matrix, scope hierarchies, licensing tiers, and invariant validation rules for the Krewire ecosystem.

## Dependencies

- Go Standard Library
- `github.com/krewire/krewire/pkg/errs`

## Key Capabilities

- **Workload Kinds (`model.Kind`)**: `app`, `cli`, `site`, `book`, `worker`, `service`, `gateway`, `infra`.
- **Scope Hierarchy (`model.Scope`)**: `ScopeWorkspace`, `ScopeModule`, `ScopeDomain`, `ScopeService`, `ScopeUnit` (ordered coarse → fine). `model.AllScopes` exposes the ladder; `ParseScope` and `(*Scope).Less`/`Level` resolve it.
- **Tiers (`model.Tier`)**: `TierFree`, `TierPro`, `TierTeam`, `TierEnterprise` (`model.AllTiers`).
- **Workload Matrix (`model.Matrix[P]`)**: Generic kind-to-package workload mapping. `model.NewMatrix(cells...)`, `(*Matrix).For(kind)`, `(*Matrix).Packages()`, `(*Matrix).All()`.
- **Opt-In Rules (`model.ViolatesOptIn[P]`)**: Declarative package import constraints per workload kind, given a `Kind`, the imported packages, and the `[]OptInRule[P]` set.
