---
description: Initialize a new Krewire project from a request or spec. Optionally give the name and variant: /new-project [name] [app|static|book|cli].
agent: build
---

Create a brand new Krewire project.

1. Load the `project-init` skill and follow it.
2. Scope: $ARGUMENTS if provided — project name and optional variant (`app` default fullstack, `static`, `book`, `cli`, `worker`, `service`, `infra`). Otherwise ask the user for the project name, purpose, and variant.
3. If the request is complex, write a spec first (use the `spec-writing` skill) and get approval before scaffolding.
4. Scaffold with `krewire new <name>` and equip the variant with `krewire init [--static|--book|--cli|--worker|--service|--infra] <name>` (or `--template <git-url>` for a starter). Then install the guild template with `krewire guild install <name>` and map the project with the `kickoff` workflow.

Report the created tree and the suggested first task.