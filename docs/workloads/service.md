# Workload: Microservice (`service`)

The `service` workload targets headless backend microservices, RPC nodes, and RESTful APIs optimized for low latency and high concurrency.

---

## 1. Key Capabilities

- **Lightweight API Gateway (`pkg/web/gateway`)**: Fast HTTP path routing, rate limiting, and token verification.
- **Service Discovery & Registry (`pkg/cloud/service/registry`)**: Built-in service registration and health checks (`/healthz`, `/readyz`).
- **Distributed Tracing & Metrics (`pkg/cloud/service/tracing`)**: Context-propagated request tracing and metrics.
- **Resilience (`pkg/cloud/service/resilience`)**: Retries, circuit breakers, and timeout policies for service-to-service calls.

---

## 2. Directory Layout

The `kiw init --service` scaffold writes:

```text
my-service/
├── krewire.yaml
├── main.go
├── internal/
│   └── handler/          # HTTP handlers
└── README.md
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-service
  kind: service

server:
  host: "0.0.0.0"
  port: 9000

service:
  name: "my-service"
  port: 9000
```

---

## 4. Commands

```bash
# Run service in development mode
kiw dev

# Build binary into .krewire/build/bin/<name>
kiw build

# Run security audit & test
kiw vet --sec
kiw test
```
