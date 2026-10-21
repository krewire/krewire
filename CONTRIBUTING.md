# Contributing to Krewire

This guide covers *how* to contribute: the workflow, conventions, and quality
gates every change must pass. It assumes you've completed
[internal/docs/getting-started.md](./internal/docs/getting-started.md) and have a working local workspace.

> All chat, code, commit messages, specs, and repository docs are in **English**.
> If you write in Indonesian or another language, reply in that language in chat,
> but keep all repository files in English.

## Communication & scope

- Write and discuss in English; the human may write in Indonesian (or any
  language) — always reply in English.
- Work in small, focused increments. One logical change per PR. Don't bundle
  unrelated work or secrets.
- The agent (and you) may push back on a request that conflicts with the project's
  current condition. Respectful dissent is part of planning, not insubordination.

## Branching model (trunk-based)

- `main` is protected; short-lived feature branches are created from it.
- Branch names are descriptive (e.g. `fix/ssg-scoped-css`, `feat/wasm-events`).
- Open a PR against `main`; require review before merge.
- Non-trivial work is tracked via GitHub Issues.

## Conventional Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat:     new feature
fix:      bug fix
refactor: change without behavior change
docs:     documentation
test:     tests
chore:    other supporting work
```

The PR title/commit becomes the release note (used by `kiw release --notes`).

## Quality gates (per module)

Run these before every PR; CI runs the same:

```bash
gofmt -l .                      # must output nothing
go vet ./...                    # must pass
go test ./...                   # must pass
kiw compat                      # must print ok (after version.go changes)
```

For `site`/`book` projects additionally run `kiw build`/`kiw serve` and spot-check
the `.krewire/build` output.

## The `version.go` compatibility contract

Each module declares its version and its minimum required versions in
`<module>/version.go`:

- `Version` — the module's own semantic version (kept in sync with its `go.mod` pin).
- `EcosystemRequires` — `map[ModuleName]Version` of the minimum versions of the
  modules it depends on.

Rules:

- Never leave `Version` and the `go.mod` pin disagreeing.
- When you bump a module, bump every dependent module's `EcosystemRequires` that
  references it. `kiw release` does this for you (maintainer mode).
- Run `kiw compat` after any `version.go` change — it must print `ok`.

## Releasing

`kiw release` is the single release orchestrator and is a **dry run by default**
(pass `--apply` to write). It serves two audiences:

- **Maintainer mode** (inside the workspace, `--module`/`--all`) — bumps a module's
  `version.go` `Version` and propagates the new minimum into dependent contracts.
  Example: `kiw release --all --bump minor --apply`.
- **Developer mode** (inside a Krewire project, no `--module` flag) — bumps the
  project's `version:` in `krewire.yaml` and builds it. Example:
  `kiw release --bump patch --apply`.

Run `kiw compat` after version updates to ensure full compatibility across the
ecosystem.

## Code style

- Idiomatic Go: clear names, `(value,error)` signatures, zero-value usability,
  `Option`/functional options where suitable.
- Don't add comments unless asked; clear code beats comments.
- Respect each package's existing conventions; mimic neighboring files.
- Keep modules cohesive — one module, one reason to change (SRP). Don't build a
  "god module".
- Prefer safe, stdlib-first code; reach for `unsafe` only with a measured,
  documented need.

## Agent notes (for AI-assisted work)

- The workspace `AGENTS.md` is the constitution — read it first.
- Useful skills: `specwriting`, `featurebuilding`, `testdriving`, `qualitygate`,
  `release`.
- Never commit unless explicitly asked; never commit secrets or large unrelated
  changes.

## Need help?

Ask in the project's discussion channels or open an issue. Start from
[internal/docs/getting-started.md](./internal/docs/getting-started.md) if your environment isn't set up yet.
