# Declarative Core Model (`packages/model`)

The `packages/model` package defines pure data models, the workload matrix, scope hierarchies, licensing tiers, and invariant validation rules for the Krewire ecosystem.

## Dependencies

- Go Standard Library
- `github.com/krewire/krewire/pkg/errs`

## Key Capabilities

- **Workload Kinds (`model.Kind`)**: `app`, `cli`, `site`, `book`, `worker`, `service`, `infra`, `kernel`.
- **Scope Hierarchy (`model.Scope`)**: `ScopeSystem`, `ScopeOrg`, `ScopeWorkspace`, `ScopeApp`, `ScopeService`, `ScopeFeature`, `ScopeComponent`, `ScopeUnit`.
- **Tiers (`model.Tier`)**: `TierFree`, `TierPro`, `TierTeam`, `TierEnterprise`.
- **Workload Matrix (`model.Matrix`)**: Generic kind-to-package workload mapping.
- **Opt-In Rules (`model.ViolatesOptIn`)**: Declarative package import constraints per workload kind.
