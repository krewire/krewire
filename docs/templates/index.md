# Templates & Scaffolding Blueprints

Krewire ships generated project scaffolds, reference blueprints, and an autonomous multi-agent AI engineering constitution under `templates/`.

---

## 1. Overview

Scaffolding has two layers:

1. **Runtime scaffolds (`tools/kiw/internal/scaffold`)** — the code that `kiw new` and `kiw init` actually execute. Variants: `app`, `static`, `book`, `cli`, `worker`, `service`, `gateway`, `infra`, and remote `template` (git clone).
2. **Reference blueprints (`templates/init/`, `templates/new/`)** — in-tree examples of the generated layout. They document the shape of each variant but are not read by the CLI at runtime.
3. **AI-Agent Boost Workflows (`templates/boost/`)** — an embedded AI team and development constitution installed via `kiw boost install`.

---

## 2. Directory Map

```text
templates/
├── init/                 # Reference blueprints (app, book, cli, static)
├── new/                  # Reference bootstrap blueprint
└── boost/                # Autonomous AI agent team, commands & skills (.agents/)
```

---

## 3. Sections

- **[Project Blueprints](./blueprints.md)**: Generated file layout per variant and custom template creation.
- **[Krewire Boost AI System](./boost/index.md)**: Multi-agent coordination, agent roles, SDLC phases, and conventions.
