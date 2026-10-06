---
name: service-mesh
description: Use for Krewire microservice patterns — registry, config center, gateway, resilience, tracing, messaging. Triggers: "microservice", "service mesh", "gateway", "registry", "circuit breaker", "tracing", "OpenTelemetry", "NATS".
---

# Service Mesh — Microservice Patterns

Opt-in distributed patterns for the modular-monolith extraction path. This skill covers `framework/service` (spec `KWF-L5H2F`).

## 1. Detect the context

- Read `docs/project-vision.md`, `docs/specs/framework/KWF-L5H2F-microservice-patterns.md`, and the extraction checklist `KWF-5ZHQV`.
- Check if `project.kind: service` or `framework/service` is imported; if not, patterns are opt-in and monolith cost must remain zero.

## 2. Registry & config

- `Registry`: `Register(Service{ID,Name,Addr,Meta,HealthCheckURL})`, `Deregister`, `Discover(ServiceName) ([]Endpoint)`, `Watch(...) (<-chan []Endpoint)`. Backends: Consul/etcd/NATS/DNS via `service.registry.backend` in `krewire.yaml`.
- `ConfigCenter`: `Get`/`Set`/`Watch(prefix)`; backends etcd/Consul/S3/Git + local/file fallback; hot reload via push watchers, no restart.

## 3. Gateway

- `web/gateway.Gateway` with `Route{Path,Method,Service,Middleware[],RateLimit,Auth}`; proxies via `Discover`.
- Middleware: `Logger`/`Trace`/`CORS`/`Auth`/`RateLimit`/`CircuitBreaker`; routes reload atomically; missing upstream → `502` `Problem` JSON, not HTML.

## 4. Resilience

- Circuit breaker `closed→open→half-open` with thresholds and `OnStateChange`.
- Retry: exponential backoff + jitter, `RetryIf(error) bool`, respects `context` deadline.
- Timeout via `context.WithTimeout`; bulkhead/semaphore → `ErrBulkheadFull`.

## 5. Tracing & messaging

- `service/tracing` configures OTel from `service.tracing {exporter,endpoint,sampler}`; W3C `traceparent` propagation via middleware; exporters OTLP/Jaeger/Zipkin/stdout (config-selected).
- `service/messaging`: `Publisher`/`Subscriber`/`Stream` with consumer groups, at-least-once; NATS JetStream primary, Kafka secondary; poison → DLQ via `worker` queue.

## 6. Gates

- In-memory fakes for all interfaces so `go test ./...` needs no containers; live tests behind build tags.
- Fixture: 3 services (gateway + 2 domains) → gateway 200 + OTel trace covering gateway→downstream; downstream kill trips breaker.

## Rules

Read `rules/key-rules.md` before acting.
