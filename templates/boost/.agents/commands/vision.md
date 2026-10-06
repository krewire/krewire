---
description: Load the unified Krewire vision — workload matrix, roadmap, and spec index — and route to the right slice.
agent: build
---

Load the unified vision and orient before any workload work.

1. Read `docs/project-vision.md` and summarize the 9-row workload matrix + roadmap in 5 lines.
2. Read `docs/specs/index.md` to see which specs are `✅ Shipped` vs `🔜 Planned` for the requested slice.
3. Route: `runtime` for frontend/WASM, `infra` for deploy/cloud, `service` for microservice/gateway, `worker` for queues — or `build` for cross-cutting.
4. Report the compact vision and the recommended subagent/skill for the task.

Use this as the first step in any new session or when the task spans multiple workloads.
