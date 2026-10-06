---
description: Bootstrap an unfamiliar Krewire project. Detect the kind, map the krewire commands that drive it, structure, and conventions.
agent: build
---

You are starting work in a Krewire project. Before doing anything else:

1. Detect the project kind by reading `krewire.yaml` (`project.kind`, `ssg:` key, `worker:`/`service:`/`infra:`) and the root files (`main.go`, `manuscript/`, `infra/`). Cross-check with `krewire info` when available.
2. Identify the `krewire` commands that drive this kind: `krewire run`/`krewire dev` for app/cli/worker/service; `krewire build`/`krewire serve` for site/book; `krewire worker` for worker; `krewire deploy` (with `--plan`/`--preview`) for infra and all deployable kinds; `krewire init` for a pre-`init` kernel.
3. Map the structure and read the documentation (README, AGENTS.md, `docs/specs/`), noting the spec-driven conventions.
4. Report a compact summary of the project map (kind, krewire commands, structure, conventions) and state what you will assume for the rest of the session.

Conclude by asking the human to confirm the plan or to give you the first task.