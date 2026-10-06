# Sync Docs — Key Rules

- `framework/tui` is the package; `cli` is the `project.kind` and `--cli` flag. Docs must use `framework/tui` for imports and `tui.NewApp`, not `framework/cli`/`cli.NewApp`.
- Original `<project>/docs/specs/` must stay as `MOVED.md` + redirect; real specs are `docs/specs/`.
- Manuscript `01-introduction.md` must describe unified 9-workload vision, not old meta-framework phrasing.
