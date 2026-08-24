---
title: "Workloads"
description: "Eight kinds, one CLI"
date: "2026-08-24"
---

# Workloads

Krewire has 8 workload kinds. `krewire.yaml` pins `project.kind`; `kiw` dispatches.

| Kind | Example | Driven by |
|------|---------|-----------|
| `app` | Fullstack monolith | `kiw run`, `kiw dev` |
| `cli` | Command-line app | `kiw run <args>` |
| `site` | Static site (this site) | `kiw build`, `kiw serve` |
| `book` | Manuscript → site | `kiw build`, `kiw serve` |
| `worker` | Job queues, cron | `kiw run`, `kiw worker` |
| `service` | Microservice | `kiw run`, `kiw dev` |
| `infra` | Cloud IaC | `kiw deploy --plan` |
| `runtime` | Go→WASM, VDOM, mount points | `kiw build` |

All share `krewire.yaml` as the single config and `framework/*` as the unified library.

## File-based routing

`pages/*.kiw` mirrors the filesystem — no router to configure.
