---
name: release
description: Use to cut a release from conventional commits: determine the version bump, generate a changelog, tag, and publish. Triggers: "release", "changelog", "version bump", "cut a release", "tag", "publish".
---

# Release

Ship a versioned release derived from Conventional Commits.

## 1. Read the git history

Inspect the commit log since the last release tag (or the whole history if none):

- `feat` → **minor** bump
- `fix`/`perf` → **patch** bump
- `BREAKING CHANGE` or `!` (feat!/fix!) → **major** bump
- Scope-local conventions may change this; follow the project's explicit rules if documented.

Determine the version from the project's manifest. For Krewire Go modules the
version lives in the release tags (`vX.Y.Z`) and in downstream `go.mod` files
that `require` the module; the module path itself carries no version. Respect
the existing versioning scheme in use.

## 2. Generate the changelog

Group the commits into sections: Added / Changed / Fixed / Removed / Security. Reference the conventional type per entry. Only include user-visible changes; skip chore noise.

## 3. Apply versioning artifacts

- Note the staged tag for this release; for Krewire Go modules there is no in-file version to bump — the tag IS the version.
- Update the changelog (docs repos) or the release notes.
- Run the quality gates (tests/lint/build — see `quality-gate` skill) after the change.

## 4. Tag & publish

- Commit the version artifacts with a conventional commit (`chore(release): vX.Y.Z` or `docs(changelog)`).
- Create the tag `vX.Y.Z` (semver, prefixed `v` unless the project uses another convention).
- Push tag; trigger the CD pipeline (see `ci-cd` skill).
- **Krewire ecosystem propagation** — a release is not done until the whole ecosystem is updated: bump the version in every downstream `go.mod` (removing any temporary local `replace`), rebuild the `krewire` binary, regenerate `docs/` and `krewire.github.io` output, update cross-repo specs referencing the package, and verify the affected sites via `curl`. No downstream repo may keep pointing at the old tag.

## 5. Verify

Confirm the tag exists, the release build ran green, and any published artifact is live.
Report: previous → new version, changelog summary, tag, and verification evidence.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.