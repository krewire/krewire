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

1. **Runtime Scaffolds (`tools/kiw/internal/scaffold`)**: The authoritative Go code executed by `kiw new <project>` and `kiw init`. Supports all eight native workloads (`app`, `site`, `book`, `cli`, `worker`, `service`, `gateway`, `infra`) and external remote templates (`--template git@...`).
2. **Reference Blueprints (`templates/init/`, `templates/new/`)**: In-tree canonical templates documenting project structures, `krewire.yaml` baselines, starter handlers, and component trees.
3. **AI-Agent Boost Workflows (`templates/boost/`)**: An embedded multi-agent pair-programming team and operational constitution installed via `kiw boost install`.

---

## 2. Scaffolding Workflows

### Creating a Fresh Project (`kiw new`)

```bash
# Fullstack Go web application with .kiw components
kiw new myapp --app

# Static documentation book
kiw new mydocs --book

# High-performance CLI tool
kiw new mytool --cli

# Background worker daemon
kiw new myworker --worker
```

### Initializing an Existing Directory (`kiw init`)

Transforms an existing directory into a Krewire-managed workload without destroying existing files:

```bash
cd existing-repo

# Initialize as a Static Site
kiw init --site

# Initialize as a Microservice
kiw init --service
```

### Equipping Repositories with AI Agents (`kiw boost install`)

Installs the 21 specialized autonomous AI agents, slash commands, domain skills, and constitutional safety rules under `.agents/` and `AGENTS.md`:

```bash
kiw boost install .
```

---

## 3. Detailed Documentation Sections

- **[Project Blueprints & Workload Layouts](./blueprints.md)**: Generated file structure for all 8 workloads and custom blueprint authoring.
- **[Krewire Boost AI System](./boost/index.md)**: Architecture of the embedded multi-agent pair-programming team.
- **[21 Autonomous Agent Roles](./boost/agents.md)**: Detailed responsibilities and capability boundaries for every subagent.
- **[Agentic SDLC & Workflows](./boost/workflow.md)**: 6-step execution lifecycle, quality gates, and checkpoint coordination.
- **[Agent Conventions & Safety Guardrails](./boost/conventions.md)**: Conventional commits, single responsibility principle, and defensive security rules.
