---
description: Deploy the project to a target environment. Optionally give the stage: /deploy [staging|prod|...].
agent: build
---

Deploy the project to the requested environment.

1. Load the `deploy` agent and the `ci-cd`/`release` skills as relevant.
2. Identify the target environment ($ARGUMENTS or ask; default = the project's normal flow) and the project's deploy method. For Krewire projects use the unified flow: `krewire build` then `krewire deploy` (use `--plan` for infra, `--preview` for PR previews). For `site`/`book` the artifact is `site/`; for `app`/`cli`/`worker`/`service` it is the binary; for `infra` it is the plan. Docs-based repos (docs/, krewire.github.io) publish to `gh-pages`.
3. Enforce gates before shipping: tests green, build succeeds, changelog/version in place (see `quality-gate`). Stop if any gate fails.
4. Run the deploy, then verify the release is live (health check or smoke test — for static sites `curl` the published URL).

Report: environment targeted, artifacts, version/tag, and verification evidence.