---
description: Write or update a design spec following the ecosystem conventions before implementing it. Optionally give a feature name: /spec [name].
agent: plan
---

Produce a design spec for the requested feature or change.

1. Clarify the goal and constraints with the human if needed.
2. Study the current codebase enough to ground the design in reality.
3. Write the spec at `docs/specs/` following the ecosystem conventions:
   - File name `{ProjectId}-{Scope}-{SpecID}-{slug}.md`; pick a unique random 5-character SpecID (do not reuse or sequence).
   - Mandatory metadata table: `SpecID`, `Title`, `Status`, `Date`, `Author`, `Domain` — no `Version`, `Last updated`, or `Changes` fields.
   - Requirements table with IDs that trace to implementation and tests (e.g. `RND-BLD-001`).
   - Add the spec to `docs/specs/index.md`.
4. Summarize the design decisions and ask for approval before any implementation starts.

$ARGUMENTS: optional spec/feature name to use in the file name slug.