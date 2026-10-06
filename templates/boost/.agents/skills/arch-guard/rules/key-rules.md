# Arch Guard — Key Rules

- `libs/core` is stdlib-only and never imports `framework` or `krewire`; `libs/kern` may import `core`/`config`/`validate` but never `framework`.
- `framework` and `krewire` compose via `kern` (`Kernel`/`Module`/`Registry`); business rules live in `core` (`Kind`/`Workload`/`SpecID`).
- Every `project.kind` must be one of 8 `core.Kind` values; unknown → `ExitCodeUsage`.
- `framework/app` importing `service`/`infra` without `service`/`infra` kind is an opt-in cost violation (`core.IsOptIn` must be true for `KindApp`).
- Original `<project>/docs/specs/` must contain only `MOVED.md` + redirect `index.md`; specs live in `docs/specs/`.
- `ssg.yaml` must not exist; config is `krewire.yaml` only (`core.ValidateKrewireYamlPath`).
- All new features need a spec in `docs/specs/` with unique 5-char `SpecID` and `FRK-*` rows before code.
