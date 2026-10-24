# Workload: Background Worker (`worker`)

The `worker` workload powers autonomous background processes, asynchronous queue consumers, scheduled crons, and event-driven data ingestion.

---

## 1. Key Capabilities

- **Supervised Lifecycle (`pkg/lifecycle`)**: Integrates with the kernel supervisor for automatic restart, health heartbeats, and graceful cancellation upon SIGTERM/SIGINT.
- **Queue & Consumer Primitives (`pkg/cloud/worker`)**: Pluggable queue abstractions (Redis, SQS, RabbitMQ, and in-memory).
- **Concurrency & Resilience (`pkg/resilience`)**: Configurable worker pool concurrency, exponential backoff retries, and circuit breakers.

---

## 2. Directory Layout

The `kiw init --worker` scaffold writes:

```text
my-worker/
├── krewire.yaml
├── main.go
├── internal/
│   └── jobs/             # Queue task handlers
└── README.md
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
    - high-priority
    - default
```

---

## 4. Commands

```bash
# Run worker in local development mode
kiw worker

# Build binary into .krewire/build/bin/<name>
kiw build

# Run test suite
kiw test
```
