---
description: Help create a Conventional Commit message from the current changes and stage them for review. Optionally give a type: /commit [type].
agent: build
---

Prepare a commit for the current changes.

1. Inspect the diff and status of the working tree. Understand what changed and why.
2. Determine the appropriate Conventional Commit type: feat, fix, refactor, docs, test, chore (scope optional).
3. If $ARGUMENTS is provided, use it as the type (e.g. `feat`). Ask the human if the intent is ambiguous.
4. Write (but do not yet run) the commit message in this format: `<type>(<scope>): <short description>`. Show the staged summary and the proposed message, then ask the human to confirm before committing.

Never include unrelated files or secrets. Only stage what belongs to this logical change.