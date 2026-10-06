---
name: project-init
description: Use when a new Krewire project must be created inside this agent workspace — a fresh app, CLI, site, or book based on a user request or a spec document. Use ONLY for starting a new project; do not use for work inside an existing project. Triggers: "new project", "init project", "start a project", "bootstrap", "make project", "buat proyek baru".
---

# Project Init (Krewire)

Initialize a brand new Krewire project in this agent workspace from a user request or a spec.
Each project gets its own isolated directory (workspace or repo).

## 1. Understand the request

Determine:

- **Name** — safe slug: lowercase, kebab-case; must match `krewire new`'s accepted set (letters, digits, dashes, dots, underscores).
- **Variant** — which shape: `app` (default fullstack monolith), `site` (`--static`), `book` (`--book`), `cli` (`--cli`), `worker` (`--worker`), `service` (`--service`), `infra` (`--infra`), or a remote starter (`--template <git-url>`).
- **Purpose** — what the project does at a high level.
- **V1 scope** — what must work first; everything else is out of the initial scaffold.

If the request is complex (multi-part, unclear requirements, or non-trivial architecture),
write a spec first with the `spec-writing` skill and get approval before scaffolding.

## 2. Confirm and create the directory

Ask the user where to create the project:
- Single project: current directory (no extra folder).
- Workspace: create under a workspace directory (e.g., `projects/<name>/`, `apps/<name>/`).

Confirm name, variant, and location with the user before writing anything.

## 3. Scaffold with krewire

- Verify `krewire` is installed (`krewire version`).
- `krewire new <name>` — creates the minimal kernel (go.mod, krewire.yaml, main.go, .gitignore).
- `krewire init <name>` or `krewire init [--static|--book|--cli|--worker|--service|--infra|--template <git-url>] <name>` — equips the chosen variant.
- Adjust `krewire.yaml` only when the variant requires it. Config lives exclusively in `krewire.yaml` — never introduce `ssg.yaml`.

## 4. Install the guild template

Run `krewire guild install <project-dir>` so the new project inherits AGENTS.md, opencode.json,
and the `.agents/` preset (agents, commands, skills). The wizard refuses to overwrite an
existing AGENTS.md; use `--force` only when you intentionally reinstall.

## 5. Map and hand off

- Run the `kickoff` workflow (or `scout`) and save the project map under the new project's
  AGENTS.md "Project-Specific Customization" so later sessions reuse it.
- Show the created tree and suggest the first task.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.