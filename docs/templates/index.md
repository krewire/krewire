# Templates & Scaffolding Blueprints

Krewire ships generated project scaffolds, reference blueprints, and an autonomous multi-agent AI engineering constitution under `templates/`.

---

## 1. Architectural Overview & Scaffolding Pipeline

Scaffolding operates across three distinct tiers:

```text
templates/
├── init/                 # Reference in-place initialization blueprints (app, book, cli, site, worker, service)
├── new/                  # Reference fresh project scaffolding blueprints
└── boost/                # Autonomous AI agent team, commands, skills & constitution (.agents/)
```

1. **Runtime Scaffolds (`krew/internal/scaffold`)**: The authoritative Go code executed by `krew new <project>` and `krew init`. Supports all eight native workloads (`app`, `site`, `book`, `cli`, `worker`, `service`, `gateway`, `infra`) and external remote templates (`--template git@...`).
2. **Reference Blueprints (`templates/init/`, `templates/new/`)**: In-tree canonical templates documenting project structures, `krewire.yaml` baselines, starter handlers, and component trees.
3. **AI-Agent Boost Workflows (`templates/boost/`)**: An embedded multi-agent pair-programming team and operational constitution installed via `krew boost install`.

---

## 2. Scaffolding Workflows

### Creating a Fresh Project (`krew new`)

```bash
# Fullstack Go web application with .kiw components
krew new myapp --app

# Static documentation book
krew new mydocs --book

# High-performance CLI tool
krew new mytool --cli

# Background worker daemon
krew new myworker --worker
```

### Initializing an Existing Directory (`krew init`)

Transforms an existing directory into a Krewire-managed workload without destroying existing files:

```bash
cd existing-repo

# Initialize as a Static Site
krew init --site

# Initialize as a Microservice
krew init --service
```

### Equipping Repositories with AI Agents (`krew boost install`)

Installs the 21 specialized autonomous AI agents, slash commands, domain skills, and constitutional safety rules under `.agents/` and `AGENTS.md`:

```bash
krew boost install .
```

---

## 3. Detailed Documentation Sections

- **[Project Blueprints & Workload Layouts](./blueprints.md)**: Generated file structure for all 8 workloads and custom blueprint authoring.
- **[Krewire Boost AI System](./boost/index.md)**: Architecture of the embedded multi-agent pair-programming team.
- **[21 Autonomous Agent Roles](./boost/agents.md)**: Detailed responsibilities and capability boundaries for every subagent.
- **[Agentic SDLC & Workflows](./boost/workflow.md)**: 6-step execution lifecycle, quality gates, and checkpoint coordination.
- **[Agent Conventions & Safety Guardrails](./boost/conventions.md)**: Conventional commits, single responsibility principle, and defensive security rules.
