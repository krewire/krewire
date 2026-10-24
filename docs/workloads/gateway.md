# Workload: API Gateway (`gateway`)

The `gateway` kind is a standalone reverse proxy that fronts one or more
upstream services. It owns a declarative route table, applies middleware
(CORS, rate limiting, tracing, timeouts, circuit breaking), and resolves
upstream addresses through the service registry.

---

## 1. Key Capabilities

- **Declarative Route Table (`pkg/web/gateway`)**: exact and prefix routes,
  per-route method matching, upstream discovery or a fixed `target_url`.
- **Middleware Chain**: CORS, rate limiting, tracing, timeout, and circuit
  breaker middleware composed per route.
- **Service Discovery (`pkg/cloud/service/registry`)**: round-robin load
  balancing over registered endpoints; missing upstreams return `502` with a
  structured Problem JSON body.
- **Same Route Table in the CLI**: `kiw gateway` reads the identical
  `gateway:` block, so the gateway can run from the project binary or from
  `kiw` during development.

---

## 2. Directory Layout

The `kiw init --gateway` scaffold writes:

```text
my-gateway/
├── krewire.yaml
├── main.go
├── internal/
│   └── config/          # typed loader for the gateway: block
└── README.md
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
project:
  name: my-gateway
  kind: gateway
  version: 0.1.0

gateway:
  host: ""
  port: 8080
  routes:
    - path: /api
      prefix: true
      strip_prefix: true
      target_url: http://localhost:8081
      timeout: 10s
    - path: /users
      service: user-service
      middleware: [cors, tracing]
      rate_limit: 100
```

Route fields mirror `gateway.Route`: `path`, `method`, `service`,
`target_url`, `prefix`, `strip_prefix`, `middleware`, `rate_limit`, and
`timeout`. `GATEWAY_ADDR` overrides `gateway.host` + `gateway.port`.

---

## 4. Detection

`shape.Detect` reports `gateway` when:

- `project.kind: gateway` is declared; or
- `krewire.yaml` contains a top-level `gateway:` key.

An explicit `project.kind` always wins over marker inference.

---

## 5. Commands

```bash
# Run the gateway with hot reload
kiw dev

# Compile the gateway binary into .krewire/build/bin/<name>
kiw build

# Stage deployable artifacts in .krewire/dist/
kiw deploy

# Run the shared gateway route table straight from the CLI
kiw gateway --port 8080 --routes "/api=user-service,/pay=payment-service"
```
