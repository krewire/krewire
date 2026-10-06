---
name: kickoff
description: Use when starting in an unfamiliar Krewire project, entering a new repository, or when AGENTS.md/README are unknown. Detects the project kind and maps the krewire workflow before doing any work. Trigger on words like "kickoff", "map the project", "new project", "where do I start".
---

# Kickoff: Map the Krewire project first

Use this skill at the start of any session in a project you have not worked in before.

## Goal

Build a reliable mental model of the project — its kind and its `krewire` workflow — so you do not guess, break things, or follow wrong conventions.

## Steps

1. **Detect the kind** — read `krewire.yaml` first: `project.kind` (`app`/`cli`/`site`/`book`/`worker`/`service`/`infra`), an `ssg:` key (site), a `manuscript/` directory (book), `worker:`/`service:`/`infra:` keys, or a root `main.go` (app). Cross-check with `krewire info` when available.
2. **Map the krewire workflow** — which commands drive this kind: `krewire run`/`krewire dev` (app/cli/worker/service), `krewire build`/`krewire serve` (site/book), `krewire worker` (worker), `krewire deploy`/`krewire build --plan` (infra), `krewire init` (kernel).
3. **Read the entry points** — AGENTS.md (if present), README, and `docs/specs/`. Capture the stated conventions (config only in `krewire.yaml`, spec-driven development).
4. **Record the map** — if the project allows writing, save the essentials into AGENTS.md under "Project-Specific Customization" or a `docs/PROJECT_MAP.md` so the next session reuses it.

## Output

A compact summary: kind, krewire commands, structure, conventions, and what remains unknown. Ask the human to confirm or correct.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.