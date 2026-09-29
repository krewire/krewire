---
title: "Agentic Development"
description: "Equip your Krewire projects with an autonomous AI agent guild using Krewire Boost — spec-driven development, unified constitutions, and automated quality gates."
date: "2026-09-29"
---

# Agentic Development

Modern software engineering is rapidly shifting toward collaboration between human developers and autonomous AI coding agents. However, unguided AI agents frequently hallucinate architectures, break conventions, or introduce dependency bloat.

Krewire solves this through **Krewire Boost** (`github.com/krewire/boost`)—an embedded agentic guild and configuration standard that equips AI agents with deep, intrinsic knowledge of the Krewire ecosystem, its 8 workloads, the `kiw` CLI, and strict spec-driven quality gates.

---

## 1. What is Krewire Boost?

**Krewire Boost** is distributed as a Go module embedded directly inside the `kiw` CLI. With a single command, you can bootstrap a complete agent guild into any Krewire repository:

```bash
kiw boost install
```

### Core Capabilities:

1. **Unified Constitution (`AGENTS.md`):** Encodes the 8-workload detection table, the full `kiw` command matrix, architectural boundaries (SRP, SoC), and immutable project rules.
2. **Specialized Subagent Guild:** 15+ dedicated agent personas (`scout`, `plan`, `build`, `reviewer`, `tester`, `debugger`, `refactor`, `docs`, `security`, `deploy`).
3. **Structured Slash Commands:** 16 standard commands (`/kickoff`, `/spec`, `/plan`, `/review`, `/test`, `/fix`, `/commit`, `/deploy`).
4. **Context-Aware Skills:** 20+ reusable procedural skills (`context-awareness`, `quality-gate`, `wasm-runtime`, `infra-provision`, `worker-queue`).
5. **Zero Vendor Lock-In:** Plug-and-play with OpenCode natively, and easily adapted for Claude Code, Cursor, Windsurf, Copilot, and Codex.

---

## 2. Installation & Scaffolding

To install Boost into your current Krewire project, run:

```bash
kiw boost install .
```

*Terminal output:*
```text
2026/09/29 21:00:00 INFO installing Krewire Boost guild into .
created AGENTS.md
created opencode.json
created .agents/agents/build.md
created .agents/agents/scout.md
created .agents/agents/reviewer.md
created .agents/agents/tester.md
created .agents/commands/kickoff.md
created .agents/commands/spec.md
created .agents/skills/quality-gate/SKILL.md
2026/09/29 21:00:00 INFO Boost installation complete: 42 files written
```

### CLI Flags:

| Flag | Description |
| :--- | :--- |
| `kiw boost install [path]` | Install template files into the target directory (defaults to current dir). |
| `--force` | Overwrite existing managed agent files without interactive confirmation. |
| `--dry-run` | Preview files that would be created without writing to disk. |
| `kiw guild install` | Canonical CLI alias for `kiw boost install`. |

---

## 3. Directory Layout of the Agent Guild

Once installed, your repository contains a standardized `.agents/` structure:

```text
my-project/
├── AGENTS.md                   # Unified Krewire constitution & rules
├── opencode.json               # OpenCode agent & plugin configuration
└── .agents/
    ├── agents/                 # Role-specific autonomous agents
    │   ├── build.md            # Primary implementation orchestrator
    │   ├── scout.md            # Codebase exploration & mapping
    │   ├── plan.md             # Architecture & system design
    │   ├── reviewer.md         # Quality, security & style review
    │   ├── tester.md           # Per-workload test automation
    │   ├── debugger.md         # Root-cause analysis & incident triage
    │   └── deploy.md           # CI/CD and production deployment
    │
    ├── commands/               # Slash commands for developers
    │   ├── kickoff.md          # /kickoff — maps stack and structure
    │   ├── spec.md             # /spec — creates verifiable design docs
    │   ├── test.md             # /test — runs workload test suite
    │   ├── review.md           # /review — triggers pre-commit audit
    │   └── commit.md           # /commit — creates conventional commits
    │
    └── skills/                 # Procedural skill packages
        ├── context-awareness/  # 4-layer situational intelligence
        ├── quality-gate/       # Lint, test, and SAST verification
        └── agent-workflow/     # Agent handoffs & lifecycle contracts
```

---

## 4. The 6-Step Agentic Workflow

When an AI agent operates within a Krewire Boost project, it executes a rigorous, deterministic six-step workflow:

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ 1. Understand│ ──> │    2. Map    │ ──> │   3. Plan    │
│  (Read & Ask)│     │  (/kickoff)  │     │ (/spec /plan)│
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
┌──────────────┐     ┌──────────────┐     ┌──────▼───────┐
│ 6. Summarize │ <── │  5. Verify   │ <── │ 4. Implement │
│ (Report Gate)│     │ (kiw test)   │     │ (build agent)│
└──────────────┘     └──────────────┘     └──────────────┘
```

### 1. Understand
The agent reads user instructions thoroughly. If any ambiguity or missing constraints are detected, the agent pauses and asks targeted questions rather than guessing or making unverified assumptions.

### 2. Map (`/kickoff`)
On the first session in a project, the developer or agent triggers `/kickoff`. The `scout` subagent inventories the project kind (`app`, `site`, `cli`, `worker`, etc.), maps `krewire.yaml` configurations, locates entry points, and records this context for future sessions.

### 3. Plan & Spec (`/spec`, `/plan`)
Non-trivial features or refactorings are never implemented blind. The agent drafts a lightweight engineering specification in `docs/specs/` defining:
- Goals and non-goals
- API contracts or schema changes
- Measurable acceptance criteria
- Test plan and roll-back strategy

### 4. Implement (`build` agent)
The `build` agent executes the minimal, convention-respecting code change. For specialized domain tasks, it delegates to domain subagents:

| Specialized Need | Subagent Invoked |
| :--- | :--- |
| Database migrations or ORM queries | `service` |
| Scoped `.kiw` component UI | `build` |
| WebAssembly client compilation | `runtime` |
| Cloud infrastructure definitions | `infra` |
| Background queues and cron schedules | `worker` |

### 5. Verify (Automated Quality Gate)
The agent executes the verification command suite:
```bash
kiw test && kiw vet
```
An agent is forbidden from reporting a task complete if tests fail or unformatted code exists.

### 6. Summarize
The agent delivers a concise, verifiable summary of changes, file citations (`file:///...`), and gate verification results.

---

## 5. Using Boost in Practice

### Kicking Off a Project

Open your project in OpenCode or your preferred AI-enabled environment and issue:

```text
/kickoff
```

The agent scans `krewire.yaml`, checks Go dependencies, detects that the workload is `site`, and responds:

```text
✓ Identified Krewire project: "my-first-app" (workload: site)
✓ Build command: kiw build
✓ Dev command: kiw dev (port 8080)
✓ Ready for tasks. Use /spec to design or /test to verify.
```

### Spec-Driven Feature Development

To design a new user registration feature:

```text
/spec user-registration
```

The agent generates a structured specification document under `docs/specs/SPEC-AUTH-user-registration.md` outlining the request schema, database constraints, error handling, and test cases before touching production code.

---

## 6. Supported AI Assistants

| Assistant | Integration Tier | How It Works |
| :--- | :--- | :--- |
| **OpenCode** | **Tier 1 (Native Plug-and-Play)** | Automatically reads `opencode.json`, loads `.agents/skills`, and provides native `/` commands. |
| **Claude Code** | **Tier 2 (Supported via Reference)** | Reads `AGENTS.md` and executes CLI commands via terminal bash sessions. |
| **Cursor / Windsurf** | **Tier 2 (Rule Import)** | Reference `AGENTS.md` from `.cursorrules` or `.windsurfrules`. |
| **GitHub Copilot** | **Tier 2 (Workspace Instructions)** | Reads `AGENTS.md` from the workspace root automatically. |

---

## Next Steps

Now that you know how to supercharge development with AI agents, explore the component templating language:

Proceed to [**2.5 DSL (.kiw) →**](/getting-started/dsl) to learn the component architecture of Krewire.
