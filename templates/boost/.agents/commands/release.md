---
description: Cut a new release from conventional commits (version bump, changelog, tag). Optionally give the bump: /release [major|minor|patch].
agent: build
---

Cut a release for this project.

1. Load the `release` skill.
2. Determine the version bump from the git history since the last tag ($ARGUMENTS overrides; infer from feats/fixes/breaking changes otherwise).
3. Bump the version in the manifest(s), generate the changelog, run the quality gates, and tag `vX.Y.Z`.
4. Push the tag and (if part of the project's flow) publish the artifact.

Report: previous → new version, changelog summary, tag, and verification evidence.