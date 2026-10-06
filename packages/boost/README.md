# Krewire Boost

A guild of AI agents tuned for the **unified Krewire framework** — one install gives any Krewire project (covering all eight kinds: `app`, `cli`, `site`, `book`, `worker`, `service`, `infra`, `kernel`) a set of agents, commands, and skills that already know the ecosystem's conventions: the full `kiw` command matrix, `krewire.yaml`-only config, spec-driven development, and quality gates.

Krewire Boost is distributed as a Go module (`github.com/krewire/boost`) whose template is embedded and installed through the [`kiw` CLI](https://github.com/krewire/kiw).

> Unified vision: [`KWF-M8K2Q`](../framework/docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md)

## Features

- **Unified constitution** — `template/AGENTS.md` encodes the 8-kind detection table, the complete `kiw` command matrix (`build`/`serve`/`run`/`dev`/`worker`/`deploy`/`dashboard`/`generate`), spec conventions, and quality gates; projects add their own rules under "Project-Specific Customization".
- **Agents & subagents (20)** in `.agents/agents/` — `build` (orchestrator), `plan` (vision-aware), `orchestrator` (parallel router), `scout` (8-kind), `reviewer` (vision-aware), `tester` (per-kind), `debugger`, `refactor`, `docs` ( specs), `security` (secrets/WASM/mTLS-aware), `deploy` (unified), `runtime` (Go→WASM), `infra` (AWS/K8s), `service` (microservice), `worker` (queues).
- **Slash commands (16)** in `.agents/commands/` — `/kickoff`, `/new-project` (8 variants), `/vision` (workload matrix), `/runtime`, `/infra`, `/service`, `/worker`, `/review`, `/test`, `/fix`, `/commit`, `/spec`, `/refactor`, `/deploy` (unified), `/release`, `/triage`.
- **Skills (26)** in `.agents/skills/` — `vision` (shared loader), `context-awareness` (4-layer situational intelligence), `agent-workflow` (handoff & lifecycle), `project-init`, `kickoff`, `requirement-gathering`, `spec-writing`, `test-driving`, `quality-gate` (vision-aware), `ci-cd` (8-kind), `release`, `bug-triage`, `conventional-commit`, `security-review`, `dependency-audit`, plus unified-workload skills `wasm-runtime` (`KWF-T4X9P`), `infra-provision` (`KWF-B7N3D`), `service-mesh` (`KWF-L5H2F`), `worker-queue` (`KWF-L5H2F`), `agent-optimization` (context compaction & parallel orchestration).
- **opencode.json** — base opencode configuration (included).
- **Go module** — template shipped as an `embed.FS` with a typed install library.
- **docs/** — conventions, workflow, and a spec template.

## Supported AI Agents (Plug-and-Play)

| AI Agent | Status |
|----------|--------|
| **OpenCode** | ✅ Native — install sets up `opencode.json` with plugin; skills load natively, agents & commands via `@capybearista/opencode-agents-loader`. |

> Other agents (Claude Code, Cursor, Copilot, Codex, Windsurf, etc.) can use the `.agents/` directory as a reference, but require manual adaptation (copying files to their expected locations). They are not plug-and-play.

## Quickstart

Install into a target project with the kiw CLI:

```bash
kiw boost install /path/to/your/project
```

With no target, an interactive wizard asks where to install and confirms before
overwriting existing managed files:

```bash
kiw boost install
```

Options: `--force` (overwrite without prompting) and `--dry-run` (preview
without writing).

Then, in your project:

1. Open OpenCode in the project directory.
2. Run `/kickoff` so the agent maps the project kind (`app`/`cli`/`site`/`book`/`worker`/`service`/`infra`/`kernel`), its `kiw` workflow, structure, and conventions.
3. Start working — the agent already knows how to think and work in your project.

## Structure

```
.
├── go.mod                      # github.com/krewire/boost
├── boost.go                    # embedded template + Install library
├── template/                   # the installable template (source of truth)
│   ├── AGENTS.md               # Unified Krewire constitution (8 kinds, full command matrix)
│   ├── opencode.json           # Base opencode configuration
│   └── .agents/                # Canonical agent directory
│       ├── agents/             # 15 agents (build, plan, orchestrator, scout, tester, ...)
│       ├── commands/           # 16 commands (kickoff, vision, runtime, infra, service, worker, ...)
│       ├── skills/             # 20 skills (vision, context-awareness, agent-workflow, wasm-runtime, ...)
└── docs/
    ├── index.md               # Documentation index
    ├── architecture.md        # Module structure & design decisions
    ├── philosophy.md          # Principles & contribution guidance
    ├── conventions.md         # Cross-language coding & repo conventions
    ├── workflow.md            # End-to-end agent workflow
    ├── sdlc.md                # SDLC phase coverage
    └── specs/                 # Specs (KWG-*) & template
```

## Using as a Library

```go
import "github.com/krewire/boost"

created, err := boost.Install("./my-project")
if errors.Is(err, boost.ErrConflicts) {
    // managed files exist; add boost.WithForce() to overwrite
}
```

The embedded tree is available as `boost.Template` for custom installers.

## Project Workflows

This template supports two workflows:

**Single project** — Install directly into your project root:
```bash
kiw boost install /path/to/your/project
```

**Multi-project workspace** — Create a workspace directory with multiple projects (you choose the layout):
```
my-workspace/
├── project-a/    # Each is a full install of this template
├── project-b/
└── project-c/
```
The `projects/` directory name is just a convention — use any structure you prefer.

## Customization

- **Set the model** — add `"model": "provider/model-id"` to `opencode.json` (e.g. `"anthropic/claude-sonnet-4-6"`). Default is empty; opencode will ask you to pick one on first use.
- **Add an agent** — create `.agents/agents/<name>.md`.
- **Add a command** — create `.agents/commands/<name>.md`.
- **Add a skill** — create `.agents/skills/<name>/SKILL.md`.
- **Strengthen project conventions** — add project-specific rules at the bottom of `AGENTS.md`.

## License

MIT
