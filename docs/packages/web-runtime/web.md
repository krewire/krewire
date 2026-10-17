# Web Framework — `web` Package Reference

## Description
The `packages/web` library provides high-performance HTTP routing, observability middleware, distributed tracing propagation, Static Site Generation (SSG), Server-Side Rendering (SSR), and WebSocket-driven Hot Module Replacement (HMR).

---

## Subsystems Overview

- **`web`**: Core HTTP router, route group matching, context wrappers, JSON serialization, and distributed request tracing.
- **`web/ssg`**: Static asset builder, image optimizer, HTML minifier, and clean URL generator.
- **`web/ssg/hmr`**: Local WebSocket server broadcasting file change events to the browser client during `kiw serve`.
- **`web/gateway`**: Reverse proxy, rate limiting, and route-based authorization filters.

---

## Observability & Distributed Tracing

`packages/web` includes production-grade observability middleware out-of-the-box:

```go
package main

import (
    "log/slog"
    "net/http"
    "github.com/krewire/krewire/packages/web"
)

func main() {
    r := web.NewRouter()

    // 1. Cross-Runtime Distributed Tracing & Request ID (KWN-NDV-013)
    // Extracts W3C "traceparent" header from Node.js/Laravel, or "X-Request-ID",
    // or generates a cryptographically secure random ID.
    r.Use(web.RequestID())

    // 2. Structured Access Logging
    r.Use(web.Logging(slog.Default()))

    // 3. Panic Recovery with JSON 500 fallback
    r.Use(web.Recover(slog.Default()))

    r.Get("/api/v1/users/{id}", func(w http.ResponseWriter, req *http.Request) {
        id := web.URLParam(req, "id")
        traceID := web.GetRequestID(req)

        web.JSON(w, http.StatusOK, map[string]any{
            "user_id":  id,
            "trace_id": traceID,
        })
    })

    http.ListenAndServe(":8080", r)
}
```

### Trace Headers Supported
- **`traceparent`**: Standard W3C distributed tracing header (`00-<trace_id>-<span_id>-<flags>`), propagated across microservices.
- **`X-Request-ID`**: Propagated into response headers and application context.
- **`X-Trace-ID`**: Supported as an alternative distributed trace ID.

---

## Static Site Generation (`web/ssg`)

Used by the `site` workload kind:
- Parses `.kiw` component files and Markdown manuscripts.
- Automatically compiles Scoped CSS (`<style>`), extracts client scripts (`<script>`), and minifies HTML.
- In development (`kiw serve`), connects to `web/ssg/hmr` via WebSocket for live-reload without page refresh.

---

## Quick References

- [Multi-Runtime Coexistence Guide](../../guides/multi-runtime-coexistence.md) — Node.js and Laravel trace propagation.
- [`krewire.yaml` Configuration Reference](../../tools/kiw/configuration.md) — HTTP server and SSG options.
- [Kernel & Control Plane Reference](../core/kern.md) — Process lifecycle and logging.
