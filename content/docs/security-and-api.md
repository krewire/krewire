---
title: "Security & HTTP API"
description: "CORS, Problem Details, Health Probes, and Request Validation"
date: "2026-09-26"
---

# Security & HTTP API

Krewire provides a production-ready, zero-dependency suite of HTTP security, validation, and observability middlewares.

---

## 1. Cross-Origin Resource Sharing (CORS)

Enable secure cross-origin communication between your API backend and frontend SPAs or WASM applications.

```go
package main

import (
    "github.com/krewire/framework/web"
)

func main() {
    app := web.New()

    // Sane defaults: allows all origins for fast development
    app.Use(web.CORS())

    // Or configure specific origins, headers, credentials, and preflight max-age
    app.Use(web.CORS(
        web.WithOrigins("https://app.example.com", "http://localhost:3000"),
        web.WithMethods("GET", "POST", "PUT", "DELETE"),
        web.WithHeaders("Authorization", "Content-Type", "X-CSRF-Token"),
        web.WithCredentials(true),
        web.WithMaxAge(86400),
    ))
}
```

---

## 2. Health & Readiness Probes (`/healthz` & `/readyz`)

Mount standardized probe endpoints for Docker, Kubernetes, and uptime monitoring:

```go
app.Use(web.Health(
    web.WithCheck("database", func(ctx context.Context) error {
        return db.PingContext(ctx)
    }),
    web.WithCheck("cache", func(ctx context.Context) error {
        return redis.Ping(ctx).Err()
    }),
))
```

- **`GET /healthz`**: Returns `200 OK` `{"status":"ok"}` for container liveness.
- **`GET /readyz`**: Executes registered dependency checks concurrently. Returns `200 OK` when all healthy, or `503 Service Unavailable` with degraded status when a dependency fails.

---

## 3. RFC 7807 Problem Details

Format standardized machine-readable error responses (`application/problem+json`):

```go
// 1. Direct response writer
web.WriteProblem(w, http.StatusBadRequest, "Insufficient funds", func(p *web.Problem) {
    p.Instance = "/accounts/123/withdraw"
    p.Errors = map[string]string{"amount": "exceeds daily limit"}
})

// 2. Fluent response builder in handlers
return web.ProblemResponse(http.StatusNotFound, "User not found")
```

---

## 4. Request Validation & Field Errors

Validate JSON payloads declaratively using struct tags:

```go
type CreateUser struct {
    Email string `json:"email" validate:"required,email"`
    Name  string `json:"name"  validate:"required,min=3"`
}

func handleCreate(r *web.Request) *web.Response {
    var req CreateUser
    if err := r.BindJSON(&req); err != nil {
        // Automatically returns HTTP 400 with structured field errors:
        // {"code":"validation_error","message":"...","details":[{"field":"email","rule":"email"}]}
        return web.Error(err)
    }
    return web.Created(req)
}
```
