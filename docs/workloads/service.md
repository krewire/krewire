# Workload: Microservice (`service`)

The `service` workload targets headless backend microservices, RPC nodes, and RESTful APIs optimized for low latency and high concurrency.

---

## 1. Key Capabilities

- **Lightweight API Gateway (`packages/web/gateway`)**: Fast HTTP path routing, rate limiting, and JWT/token verification.
- **Service Discovery & Registry (`packages/cloud/service/registry`)**: Built-in service registration and health checks (`/healthz`, `/readyz`).
- **Distributed Tracing & Metrics (`packages/cloud/service/tracing`)**: Context-propagated request tracing and OpenTelemetry-aligned metrics.

---

## 2. Directory Layout

```text
my-service/
├── krewire.yaml          # Project configuration (kind: service)
├── main.go               # Server initialization
├── api/                  # Route handlers & DTO definitions
│   └── v1/
└── internal/             # Domain logic & data layer
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
  mesh: false
  health_check_path: "/healthz"
```

---

## 4. Commands

```bash
# Run service in development mode
kiw dev

# Build binary
kiw build

# Run security audit & test
kiw vet --sec
kiw test
```
