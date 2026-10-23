# Workload: Background Worker (`worker`)

The `worker` workload powers autonomous background processes, asynchronous queue consumers, scheduled crons, and event-driven data ingestion.

---

## 1. Key Capabilities

- **Supervised Lifecycle (`packages/lifecycle`)**: Integrates with the kernel supervisor for automatic restart, health heartbeats, and graceful cancellation upon SIGTERM/SIGINT.
- **Queue & Consumer Primitives (`packages/cloud/worker`)**: Pluggable queue abstractions (Redis, SQS, RabbitMQ, and in-memory).
- **Concurrency & Resilience (`packages/resilience`)**: Configurable worker pool concurrency, exponential backoff retries, and circuit breakers.

---

## 2. Directory Layout

```text
my-worker/
├── krewire.yaml          # Project configuration (kind: worker)
├── main.go               # Worker boot & supervisor integration
└── jobs/                 # Queue task handlers & jobs
    ├── email_job.go
    └── cleanup_job.go
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-worker
  kind: worker

worker:
  concurrency: 8
  queues:
    - name: "high-priority"
      rate_limit: 100
    - name: "default"
      rate_limit: 50
```

---

## 4. Commands

```bash
# Run worker in local development mode
kiw worker

# Build binary
kiw build

# Run test suite
kiw test
```
