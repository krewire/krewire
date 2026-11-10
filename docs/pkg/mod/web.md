# `web`

Import: `github.com/krewire/krewire/pkg/web`

## Purpose

High-performance HTTP routing, distributed tracing, observability middleware, Static Site Generation (SSG), Server-Side Rendering (SSR), and API Gateway reverse proxy.

## Subpackages & Core Primitives

- **`web`**:
  - `NewRouter()` — High-performance HTTP router with route groups, URL parameters, and context wrappers.
  - `NewApp()` — Monolith application container pre-loaded with the default UI component registry and baseline security middleware stack (`RecoverMiddleware`, `AccessLogMiddleware`, and `SecurityHeaders` enforcing `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, and `Referrer-Policy: strict-origin-when-cross-origin` per Pillar 1).
  - Middleware: `RequestID()`, `Logging()`, `Recover()`, `SecurityHeaders()`, `Gzip()`, `Compress()`.
- **`web/ssg`**: Static site generator, HTML minification, clean URLs, and Markdown pipeline.
- **`web/ssg/hmr`**: WebSocket live-reload and Hot Module Replacement for `kiw serve`.
- **`web/gateway`**: Reverse proxy with rate limiting, timeouts, CORS, and upstream service discovery.

## Example

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

	r.Get("/hello", func(w http.ResponseWriter, req *http.Request) {
		web.JSON(w, http.StatusOK, map[string]string{"message": "Hello from Krewire"})
	})

	http.ListenAndServe(":8080", r)
}
```

## Design boundary

Core HTTP transport layer. See the full [Web Reference](../web-runtime/web.md).
