# Release Rules

- Derive the version bump from Conventional Commits (feat=minor, fix/perf=patch, BREAKING=major).
- Respect the project's existing versioning scheme and manifest version source.
- Changelog includes only user-visible changes; skip chore noise.
- Run the quality gates after any version bump.
- Publish artifacts only if that is part of the project's flow.
- Verify the release: tag exists, build green, artifact live.