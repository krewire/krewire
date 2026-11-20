# Package Reference: `web`

Import: `github.com/krewire/krewire/pkg/web`

## 1. Purpose & Architecture

The `pkg/web` package provides high-performance HTTP routing, structured observability, distributed tracing propagation, RFC 7807 problem details, Static Site Generation (SSG), and Edge API Gateway routing.

For complete routing, middleware, and gateway specifications, see [Web Framework Reference Guide](./web-runtime/web.md).

---

## 2. Main API Reference

- **`web.NewRouter() *Router`**: Radix-tree HTTP router supporting parameterized paths (`/users/{id}`), wildcards, and route groups.
- **`web.NewApp() *App`**: Monolith application container with recovery, logging, and security headers active by default.
- **`web.JSON(w, status, data)`**: Encodes data to JSON with appropriate headers.
- **`web.WriteProblem(w, problem)`**: Serializes RFC 7807 Problem Details for standardized HTTP error responses.
- **`web.RequestID()` / `web.Logging(logger)` / `web.Recover(logger)`**: Standard observability middleware.
- **`web/gateway.NewGateway()`**: Decoupled reverse-proxy and rate-limiting Edge API Gateway.

---

## 3. Usage Example

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/web"
)

func main() {
    r := web.NewRouter()
    r.Use(web.RequestID())
    r.Use(web.Recover(nil))

    r.Get("/api/status", func(w http.ResponseWriter, req *http.Request) {
        web.JSON(w, http.StatusOK, map[string]string{
            "status":   "operational",
            "trace_id": web.GetRequestID(req),
        })
    })

    http.ListenAndServe(":8080", r)
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Baseline security headers (CSP, HSTS, X-Content-Type-Options: nosniff) enabled by default; RFC 7807 prevents leaking database credentials in error messages.
- **Sustainable (S2)**: Memory footprint bounded under 15 MB; cold starts sub-millisecond on a $5/month VPS.
- **Scalable (S3)**: Seamlessly transitions from local router to decoupled reverse-proxy gateway (`pkg/web/gateway`).
