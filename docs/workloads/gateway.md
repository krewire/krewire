# Workload: API Gateway (`gateway`)

The `gateway` kind is a standalone reverse proxy and ingress controller that fronts one or more upstream microservices or lateral platforms. It owns a declarative route table, applies defensive middleware (rate limiting, CORS, distributed tracing, circuit breakers), and resolves upstream addresses through the service registry or static target URLs.

In accordance with [ADR-E8P9D](../../../internal/docs/adr/ADR-E8P9D-pyramid-architecture-and-bridge-gateway-decoupling.md), the API Gateway acts as the decoupled bridge between lateral Layer 4 platforms (`auth`, `cloud`, `dash`), enforcing Zero Peer Coupling.

---

## 1. Architectural Capabilities

- **Declarative Route Table (`pkg/web/gateway`)**: Prefix and exact path routing, method filtering, URL rewrites, and path prefix stripping (`strip_prefix: true`).
- **Lateral Peer Decoupling**: Sibling platforms never import peer code; all lateral HTTP communication is routed through gateway boundaries.
- **Defensive Middleware Chain**: Built-in per-route rate limiters (token bucket), CORS policies, timeout contexts, and circuit breakers.
- **Service Discovery & Load Balancing (`pkg/cloud/service/registry`)**: Resolves service names to healthy endpoints with round-robin balancing and automatic failover.
- **RFC 7807 Structured Errors**: If an upstream service is unreachable, the gateway responds with standardized Problem Details (`502 Bad Gateway` / `504 Gateway Timeout`) rather than leaking raw socket connection errors.
- **W3C Distributed Tracing**: Propagates `traceparent` headers across microservices to maintain end-to-end request visibility.

---

## 2. Standard Directory Layout

Scaffolded via `kiw new <gateway-name> --gateway` or `kiw init --gateway`:

```text
my-gateway/
├── krewire.yaml          # Declarative route table & proxy config
├── main.go               # Gateway daemon entrypoint
├── internal/
│   ├── config/           # Typed configuration loader
│   └── handlers/         # Custom authentication or gateway filters
└── README.md
```

---

## 3. Configuration Contract (`krewire.yaml`)

```yaml
project:
  name: "api-gateway"
  kind: "gateway"
  version: "1.0.0"

server:
  host: "0.0.0.0"
  port: 8080

gateway:
  timeout: "15s"
  rate_limit:
    enabled: true
    rate: 100             # Requests per minute per client IP
    burst: 20
  routes:
    # 1. Route to Auth Platform
    - path: "/api/auth"
      prefix: true
      strip_prefix: true
      target_url: "http://127.0.0.1:8081"
      timeout: "5s"

    # 2. Route to Cloud PaaS Platform
    - path: "/api/cloud"
      prefix: true
      strip_prefix: true
      target_url: "http://127.0.0.1:8082"
      middleware: ["tracing", "cors"]

    # 3. Dynamic Service Discovery via Registry
    - path: "/api/billing"
      service: "billing-service"
      timeout: "10s"
```

---

## 4. Complete Gateway Implementation Example

```go
package main

import (
    "log"
    "net/http"
    "net/url"
    "time"
    "github.com/krewire/krewire/pkg/web/gateway"
)

func main() {
    gw := gateway.NewGateway()

    authTarget, _ := url.Parse("http://127.0.0.1:8081")
    cloudTarget, _ := url.Parse("http://127.0.0.1:8082")

    // Configure routes
    gw.Route("/api/auth/*", authTarget, gateway.RouteOptions{
        StripPrefix: true,
        RateLimit:   120, // 120 req/min
    })

    gw.Route("/api/cloud/*", cloudTarget, gateway.RouteOptions{
        StripPrefix: true,
        Timeout:     30 * time.Second,
    })

    log.Println("API Gateway listening on :8080")
    if err := http.ListenAndServe(":8080", gw); err != nil {
        log.Fatal(err)
    }
}
```

---

## 5. Development & Deployment Pipelines

```bash
# 1. Start gateway with live watching
kiw dev

# 2. Run ad-hoc gateway directly from the CLI
kiw gateway --port 8080 --routes "/api/auth=http://localhost:8081,/api/cloud=http://localhost:8082"

# 3. Compile static pure-Go binary
kiw build --minify

# 4. Deploy gateway via systemd service
kiw deploy --target systemd
```

---

## 6. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Acts as an edge perimeter; terminates malicious HTTP traffic, enforces token-bucket rate limits, and redacts internal topology details in RFC 7807 responses.
- **Sustainable (S2)**: Lean memory footprint (<15 MB); replaces multi-gigabyte commercial API gateways (Kong, Envoy) with a single static binary.
- **Scalable (S3)**: Enables the Zero Peer Coupling rule; services scale independently behind gateway abstraction boundaries.
