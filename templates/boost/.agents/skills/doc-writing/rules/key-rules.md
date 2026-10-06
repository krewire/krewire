# Doc Writing — Key Rules

- `framework/tui` is the package; `cli` is the `project.kind` and `--cli` flag. Docs must use `framework/tui` for imports and `tui.NewApp`.
- Use `rules/doc-template.md` for the doc type; keep workload matrix to 9 rows and architecture tree to one page.
- `sync-docs` must report `In-sync` before docs PR is done.
