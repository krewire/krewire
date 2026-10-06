---
description: Review changes for correctness, security, and conventions. Optionally scope to a path or files: /review [path].
agent: plan
---

Review the current changes in this project.

Scope: $ARGUMENTS if provided (a file path, glob, or range like "src/api"). Otherwise review the current uncommitted diff; if there is none, review the latest commit.

Run the `reviewer` subagent. It will produce a report with severity-based findings and a verdict. Present the report and highlight any Blockers/Majors for the human to decide on.