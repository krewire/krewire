---
description: Krewire microservice patterns — registry, gateway, resilience, tracing, messaging. Optionally: /service [registry|gateway|resilience|tracing].
agent: build
---

Execute a Krewire microservice task. $ARGUMENTS is the slice; if empty, map from the request.

1. Load `vision` + `service-mesh` skills and `KWF-L5H2F` spec (and `KWF-5ZHQV` extraction checklist).
2. Delegate to the `service` subagent: registry/config center, gateway (atomic reload, `Problem` JSON), resilience (breaker/retry/bulkhead), tracing (OTel/W3C), messaging (NATS JetStream).
3. Ensure opt-in cost — `framework/app` without `service` shows no size change; tests use fakes, not containers.

For full service projects, let `build` orchestrate `service` + `infra`; report registry backend, gateway routes, and trace evidence.
