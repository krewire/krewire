---
description: Krewire infra — provider abstraction, state/locking, plan/apply, AWS and Kubernetes, krewire deploy. Optionally: /infra [plan|deploy|preview|destroy].
agent: build
---

Execute a Krewire infra task. $ARGUMENTS is the operation (`plan`/`deploy`/`preview`/`destroy`); default is `plan`.

1. Load `vision` + `infra-provision` skills and `KWF-B7N3D` spec.
2. Delegate to the `infra` subagent: provider contract, schema/validation, state+locking, plan purity, preview lifecycle.
3. Always run `krewire deploy --plan` (or `krewire build --plan`) as dry-run before any mutating `deploy`; handle `ErrAlreadyLocked` and idempotence checks.

Never use live AWS/K8s accounts in gates — fakes/`envtest` only. Report plan output, state backend, and lock behavior.
