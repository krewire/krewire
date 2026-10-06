# Specifications Index — Krewire Forge

This directory holds the formal specifications for Forge, the programmatic
app/UI builder.

| SpecID    | Title                                  | Status | Impl Status | Depends On |
| --------- | -------------------------------------- | ------ | ----------- | ---------- |
| [KWF-M4R8T](./KWF-COMP-M4R8T-forge-atomic-components.md) | Forge Atomic `.kiw` Component Library | Draft | Shipped | KWL-KIW-001 |

## Conventions

- Each specification is stored as a single Markdown file named
  `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
- SpecIDs are unique, random 5-character alphanumeric codes (e.g. `KWF-M4R8T`).
- New specifications must be added to this index when created.
- Requirements use the `FRK-FA-*` prefix (Forge — Atomic components); scope is
  declared per requirement row.
- Ordering: impact-to-effort (high impact, low effort first), then dependency
  chain (foundations first).

## Component Contract Summary

The rules a new component must satisfy are in
[KWF-M4R8T](./KWF-COMP-M4R8T-forge-atomic-components.md). The short version:

- A component is one `.kiw` file in `components/`, embedded automatically.
- It carries a `data-kiw-component="<Name>"` hook and its own `<style>` block.
- It emits no JavaScript; behaviour comes from CSS or the host.
- Props are addressed by name, and `Class` is always accepted.
- Status variants are `info`, `success`, `warning`, `error`. `danger` and
  `warn` were removed (FRK-FA-031); use `error` and `warning`.

A component that does not parse is skipped silently by the loader, so
`framework/web/ssg` asserts the whole set loads — see `forge_atoms_test.go`.