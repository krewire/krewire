# Krewire Boost — AI Agent Engineering System

An autonomous, multi-agent AI pair-programming team and operational engineering constitution embedded directly into the developer workflow.

---

## 1. Overview & Core Philosophy

Krewire Boost equips repositories with a complete team of 21 specialized autonomous AI agents, predefined slash commands, and domain skills stored under `.agents/` and governed by the repository `AGENTS.md`.

Rather than treating AI as a generic code autocomplete, Boost models software engineering as a coordinated, role-specialized team executing deterministic development phases with strict quality gates:

```text
Understand ──► Map (Scout) ──► Plan & Spec ──► Implement ──► Verify (Gates) ──► Summarize
```

---

## 2. Installing Boost in Any Project

Boost is installed into any existing repository using the `kiw` CLI tool:

```bash
# Install Boost into current repository
kiw boost install .

# Preview files to be created without writing to disk
kiw boost install . --dry-run

# Force update / re-align Boost templates
kiw boost install . --force
```

### Managed Files & Directory Layout

```text
your-project/
├── AGENTS.md                  # Operational constitution and repository rule SSOT
├── opencode.json              # Tool configuration & agent routing parameters
└── .agents/                   # Specialized agent configurations
    ├── agents/                # 21 subagent personas and tool permission boundaries
    ├── commands/              # Slash commands (/plan, /spec, /refactor, /test, /audit, /deploy)
    ├── skills/                # Domain skill definitions with procedural execution guides
    └── rules/                 # Invariant safety, quality, and architectural rules
```

---

## 3. Boost System Guides

- **[AI-Native Engineering Model](../../guides/ai-engineering-model.md)** — Governed interoperability across Krew, Boost, and `kiw`.
- **[21 Autonomous Agent Roles](./agents.md)** — Specialized subagents across planning, development, testing, and operations.
- **[Autonomous SDLC & Multi-Agent Workflow](./workflow.md)** — The 7-phase governed development lifecycle and slash command mappings.
- **[Agent Conventions & Safety Guardrails](./conventions.md)** — Coding style, pure-Go invariant, and zero-secrets security rules.

- **[Autonomous Agent Roster & Specializations](./agents.md)**: Directory of all 21 specialized agents covering planning, architecture, coding, testing, refactoring, security auditing, and operations.
- **[Agentic SDLC & Execution Workflows](./workflow.md)**: 6-phase development lifecycle, slash commands, role delegation, and gate enforcement.
- **[Agent Conventions & Safety Boundaries](./conventions.md)**: Code style invariants, Conventional Commits, destructive command prevention, and secret masking.
