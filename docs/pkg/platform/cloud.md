# Package: `cloud` (Cloud Runtime & Infra Blueprint Engine)

The `packages/cloud` suite provides the cloud execution plane, task runners, service discovery, background queues, and infrastructure state management.

---

## 1. Cloud Subsystems

- **`cloud/runner`**: Task executor managing containerized workload processes.
- **`cloud/service`**: Microservice coordination, distributed resilience policies, and tracing.
- **`cloud/worker`**: Distributed asynchronous queue workers.
- **`cloud/storage`**: Cloud object store driver implementations.
- **`cloud/infra`**: Schema, provider definitions, and state locks for infrastructure blueprints.
- **`cloud/deploy`**: Rollout orchestration, canary deployment stages, and blue-green switching.
