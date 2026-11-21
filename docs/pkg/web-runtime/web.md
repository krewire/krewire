# Web Framework — `web` Package Reference

The `pkg/web` library provides a high-performance HTTP router, observability middleware, distributed tracing propagation, RFC 7807 problem details, Static Site Generation (SSG), Server-Side Rendering (SSR), Edge API Gateway routing, and WebSocket-driven Hot Module Replacement (HMR).

---

## 1. Architectural Overview & Subsystem Map

```text
pkg/web/
├── router.go       # High-performance radix-tree HTTP router (Get, Post, Put, Delete, Group)
├── route.go        # Parameter extraction, wildcards & route handler matching
├── request.go      # Request decoding, query binding & validation hooks
├── response.go     # JSON, HTML, Text, Redirect & Streaming response writers
├── problem.go      # RFC 7807 Problem Details structured error serialization
├── security.go     # Baseline defense headers (CSP, HSTS, XFO, nosniff)
├── cookie.go       # Secure cookie creation, SameSite & encryption
├── session.go      # In-memory and signed session management
├── app.go          # High-level web application container (web.NewApp)
├── ssg/            # Static Site Generation pipeline, asset optimizer & HTML minifier
│   ├── render.go   # SSR/SSG template rendering engine
│   ├── mount.go    # Client-side island hydration mounting bridge
│   ├── hmr/        # WebSocket live-reload dev server for krew serve
│   └── assets.go   # Scoped CSS compilation & cache-busting fingerprinting
└── gateway/        # Decoupled Edge API Gateway & Reverse Proxy
    ├── gateway.go  # Proxy director, path rewrite & upstream connection pooling
    ├── ratelimit.go# In-memory token-bucket rate limiting per client IP
    └── middleware.go# Gateway security headers & authorization token verification
```

---

## 2. HTTP Routing & Parameter Extraction (`pkg/web.Router`)

The router supports fast URL parameter extraction, route groups, and modular middleware pipelines:

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/web"
)

func main() {
    r := web.NewRouter()

    // 1. Global Middleware
    r.Use(web.RequestID())
    r.Use(web.Logging(nil))
    r.Use(web.Recover(nil))

    // 2. Public Static & Health Routes
    r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
        web.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
    })

    // 3. Nested API Route Group
    api := r.Group("/api/v1")
    {
        api.Get("/users", ListUsers)
        api.Post("/users", CreateUser)
        api.Get("/users/{id}", GetUserByID)
    }

    http.ListenAndServe(":8080", r)
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
    id := web.URLParam(r, "id")
    web.JSON(w, http.StatusOK, map[string]string{"user_id": id})
}
```

---

## 3. RFC 7807 Problem Details (`pkg/web.Problem`)

Standardized HTTP error representations prevent inconsistent client-side error handling:

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/web"
)

func HandlePayment(w http.ResponseWriter, r *http.Request) {
    // Return RFC 7807 Problem Details
    web.WriteProblem(w, web.Problem{
        Type:     "https://krewire.com/errors/insufficient-balance",
        Title:    "Insufficient Account Balance",
        Status:   http.StatusPaymentRequired,
        Detail:   "Your wallet balance of $12.50 is insufficient for this $50.00 transaction.",
        Instance: "/accounts/acc_123/transactions",
        InvalidParams: []web.InvalidParam{
            {Name: "amount", Reason: "Exceeds available spendable funds"},
        },
    })
}
```

Emitted HTTP response (`Content-Type: application/problem+json`):
```json
{
  "type": "https://krewire.com/errors/insufficient-balance",
  "title": "Insufficient Account Balance",
  "status": 402,
  "detail": "Your wallet balance of $12.50 is insufficient for this $50.00 transaction.",
  "instance": "/accounts/acc_123/transactions",
  "invalid_params": [
    {"name": "amount", "reason": "Exceeds available spendable funds"}
  ]
}
```

---

## 4. Observability & Distributed Tracing

`pkg/web` supports seamless W3C distributed tracing across hybrid workloads:

- **`traceparent`**: Standard W3C distributed tracing header (`00-<trace_id>-<span_id>-<flags>`), propagated across microservices and external runtimes (Node.js/Laravel).
- **`X-Request-ID`**: Cryptographically random unique request ID attached to logger context and response headers.
- **`X-Trace-ID`**: Ecosystem trace ID binding.

---

## 5. Edge API Gateway & Reverse Proxy (`pkg/web/gateway`)

The decoupled API Gateway provides reverse proxying, path rewriting, and rate limiting:

```go
package main

import (
    "net/http"
    "net/url"
    "time"
    "github.com/krewire/krewire/pkg/web/gateway"
)

func main() {
    gw := gateway.NewGateway()

    // Upstream service targets
    authTarget, _ := url.Parse("http://127.0.0.1:8081")
    paasTarget, _ := url.Parse("http://127.0.0.1:8082")

    // Route traffic by prefix with rate limiting
    gw.Route("/api/auth/*", authTarget, gateway.RouteOptions{
        StripPrefix: true,
        RateLimit:   100, // requests per minute
    })

    gw.Route("/api/cloud/*", paasTarget, gateway.RouteOptions{
        StripPrefix: true,
        Timeout:     30 * time.Second,
    })

    http.ListenAndServe(":8080", gw)
}
```

---

## 6. Static Site Generation & Hot Module Replacement (`pkg/web/ssg`)

The `web/ssg` engine powers Krewire's `site` and `book` workloads:

- **Template Compilation**: Parses `.kiw` component files and Markdown content files.
- **Scoped CSS Isolation**: Generates deterministic scoped class identifiers (`data-kiw-*`).
- **Asset Optimization**: Inlines critical CSS, minifies HTML, and fingerprints asset URLs with version tags (`?v=0.2.3`).
- **Hot Module Replacement (`web/ssg/hmr`)**: Embeds a zero-dependency WebSocket client in development (`krew serve`) to hot-reload templates and styles without full page refreshes.

---

## 7. Security-First Monolith Assembly (`web.NewApp`)

`web.NewApp()` bundles all defense-in-depth middleware into a single entry point:

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/web"
)

func main() {
    app := web.NewApp()
    
    // Automatic middleware: Panic Recover + Structured slog + Security Headers
    app.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("<h1>Welcome to Krewire</h1>"))
    })

    http.ListenAndServe(":8080", app.Router())
}
```

---

## 8. Alignment with the Three Pillars (3S)

- **Secure (S1)**: OWASP defense headers enabled by default; RFC 7807 prevents leaking internal database stack traces to clients.
- **Sustainable (S2)**: Memory footprint bounded under 15 MB; zero npm or external reverse-proxy dependencies (Nginx/Traefik) needed for development.
- **Scalable (S3)**: Seamlessly transitions from local monolithic router to distributed reverse-proxy gateway (`pkg/web/gateway`) without rewriting route handlers.
