---
title: "Krewire Framework"
description: "Comprehensive architectural deep dive into github.com/krewire/framework, the ecosystem modules, package layout, and modular monolith design."
date: "2026-09-29"
---

# Krewire Framework

The **Krewire Framework** (`github.com/krewire/framework`) is the foundational web and application engine powering the Krewire ecosystem. It provides an expressive, idiomatic Go toolset for building everything from lightweight static sites and command-line interfaces to high-throughput fullstack monoliths and distributed microservices.

---

## 1. Ecosystem Architecture

The Krewire project is divided into focused, decoupled repositories that compose cleanly:

```
krewire/
├── framework/    # Core runtime & web engine (web, ui, tui, dsl, app, worker, service, infra)
├── libs/         # Low-level standard-library utilities (core, kern, term, config, validation)
├── kiw/          # Unified developer CLI & task runner (kiw new, dev, build, run, test)
└── mdbind/       # Framework-free markdown manuscript & documentation book compiler
```

- **`github.com/krewire/libs`**: The standard-library-only foundation. Contains `libs/core` (declarative workload registry and core domain models), `libs/kern` (the imperative execution kernel), `libs/term` (terminal rendering utilities), `libs/config` (typed configuration loaders), and `libs/validation` (struct tag validator).
- **`github.com/krewire/framework`**: The high-level developer framework. Combines HTTP routing, the `.kiw` component engine, scoped styling, background worker harnesses, service discovery, and cloud IaC abstractions.
- **`github.com/krewire/kiw`**: The single developer tool binary. Drives scaffolding, compilation, hot reloading, test execution, and deployment pipelines.
- **`github.com/krewire/mdbind`**: The independent markdown book engine powering `krewire.com/docs/` and standalone technical manuscripts.

---

## 2. Package Organization

In accordance with specification `KWF-ARCH-M8K2Q`, the framework follows a flat, opt-in package architecture. Workload capabilities are activated solely through imports—unused features add zero bytes to compiled binaries:

| Package | Purpose & Functionality |
| :--- | :--- |
| `framework/web` | Expressive HTTP routing, request/response lifecycle, middleware pipeline, CORS, CSRF, security headers, sessions, cookies, and `html/template` rendering. |
| `framework/web/ssg` | File-based static site generator compiling `.kiw` files into optimized HTML/CSS assets. |
| `framework/ui` | Theme engine, Light/Dark palettes, CSS design tokens, and scoped class scoping (`data-kiw-component`, `data-kiw-layout`). |
| `framework/tui` | Command-line app harness, reactive terminal UI engine, POSIX flags, and `log/slog` structured logging. |
| `framework/dsl` | Compiler for the unified `.kiw` component format (HTML, CSS, Go, and Markdown in one file). |
| `framework/assets` | Static asset manager with multi-source store (`dir` / `embed.FS`), ETag generation, Cache-Control headers, and fingerprint manifests. |
| `framework/storage` | Key-value store abstraction with Memory and File backends, context cancellation, and DI provider binding. |
| `framework/app` | Fullstack application assembly, Dependency Injection (DI) container, and modular monolith wiring (`KWF-5ZHQV`). |
| `framework/worker` | Background jobs, asynchronous queues, cron schedules, exponential retry backoff, and dead-letter queues (DLQ). |
| `framework/service` | Microservice contracts: service discovery (Consul, etcd, DNS), API gateway routing, rate limiting, circuit breaker, and OpenTelemetry (OTel) tracing. |
| `framework/infra` | Provider-agnostic cloud infrastructure as code (IaC) declaring compute, databases, storage, and networking in pure Go. |
| `framework/runtime` | WebAssembly client runtime (`GOOS=js GOARCH=wasm`) with Virtual DOM diffing and interactive island hydration. |

---

## 3. The Modular Monolith Pattern (`KWF-5ZHQV`)

Modern software projects often suffer from the false dichotomy between a tightly coupled "spaghetti monolith" and an prematurely distributed "microservice hell".

Krewire implements the **Modular Monolith** architecture:

```
app/
├── cmd/app/main.go            # Single binary entry point
├── internal/
│   ├── identity/              # Domain module: Auth, Users, Roles
│   │   ├── handler.go         # HTTP endpoints
│   │   ├── service.go         # Domain business logic
│   │   └── store.go           # Database operations
│   ├── billing/               # Domain module: Subscriptions, Invoices
│   └── notifications/         # Domain module: Email, Webhooks
└── web/
    ├── layouts/               # Presentation layouts
    └── pages/                 # File-based views
```

### Architectural Benefits

1. **High Cohesion, Low Coupling:** Each business domain resides in its own isolated package with clear public contracts and private implementations.
2. **Single Binary Deployment:** Develop, test, and deploy everything as one unified static binary with `kiw run` or Docker.
3. **Painless Extraction:** When a specific domain (such as `notifications` or `billing`) requires independent scaling or a dedicated queue, it can be extracted directly into a `worker` or `service` workload without rewriting business logic.

---

## 4. The `.kiw` Component DSL

Krewire introduces the `.kiw` file format—a modern, single-file component architecture combining HTML structure, scoped CSS, Go server scripts, and Markdown prose with zero external dependencies:

```kiw
---
title: "Product Overview"
layout: "Base"
---

<div class="product-card">
  <h1>{{.Title}}</h1>
  <p class="summary">{{.Description}}</p>
  <button class="btn-primary" onclick="alert('Added!')">Add to Cart</button>
</div>

<style>
.product-card {
  border: var(--pop-border);
  background: var(--base-1);
  padding: 1.5rem;
  border-radius: 12px;
  box-shadow: var(--pop-shadow);
}
.btn-primary {
  background: var(--primary);
  color: var(--primary-content);
  font-weight: 800;
  border: var(--pop-border-thin);
}
</style>

<script lang="go" server>
package product

import "context"

type Props struct {
    Title       string
    Description string
}

func Load(ctx context.Context, p Props) (Props, error) {
    p.Description = "Loaded dynamically from database or API"
    return p, nil
}
</script>

<markdown>
### Product Specifications
- Single binary runtime with embedded assets
- Zero client-side JavaScript required by default
- Automatic CSS scoping prevents style leakage
</markdown>
```

---

## 5. Production HTTP Engine

The `framework/web` package delivers a robust HTTP pipeline built directly on top of Go's `net/http` standard library:

### Expressive Routing & Middleware

```go
package main

import (
    "context"
    "net/http"
    "github.com/krewire/framework/web"
)

func main() {
    app := web.New()

    // Global Middleware Stack
    app.Use(web.Recovery())
    app.Use(web.Logger())
    app.Use(web.SecurityHeaders())
    app.Use(web.CORS(
        web.WithOrigins("https://app.krewire.com"),
        web.WithMethods("GET", "POST", "PUT", "DELETE"),
        web.WithCredentials(true),
    ))

    // Health and Readiness probes
    app.Use(web.Health(
        web.WithCheck("db", func(ctx context.Context) error { return db.Ping(ctx) }),
    ))

    // Route Grouping
    api := app.Group("/api/v1")
    {
        api.Get("/users", listUsers)
        api.Post("/users", createUser)
        api.Get("/users/:id", getUserByID)
    }

    app.Listen(":8080")
}
```

### Standardized Problem Details (RFC 7807)

Return consistent, machine-readable API error responses:

```go
func getUserByID(r *web.Request) *web.Response {
    user, err := store.FindUser(r.Param("id"))
    if err != nil {
        return web.ProblemResponse(http.StatusNotFound, "User Not Found", func(p *web.Problem) {
            p.Detail = "No user matches the provided identifier."
            p.Instance = r.URL.Path
        })
    }
    return web.JSON(http.StatusOK, user)
}
```

### Declarative Struct Validation

```go
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18"`
}

func registerHandler(r *web.Request) *web.Response {
    var req RegisterRequest
    if err := r.BindJSON(&req); err != nil {
        return web.Error(err) // Auto-generates structured HTTP 400 with field details
    }
    return web.Created(req)
}
```

---

## 6. Summary

The Krewire Framework bridges the gap between high-velocity developer ergonomics and robust Go system design. By standardizing routing, configuration, component compilation, and deployment into a unified ecosystem, Krewire allows teams to scale seamlessly from idea to high-volume production.

Proceed to [**1.2 Why Krewire? →**](/docs/why-krewire) to learn how Krewire compares to traditional multi-language architectures.
