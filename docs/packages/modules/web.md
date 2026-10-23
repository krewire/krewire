# `web`

Import: `github.com/krewire/krewire/packages/web`

## Purpose

High-performance HTTP routing, distributed tracing, observability middleware, Static Site Generation (SSG), Server-Side Rendering (SSR), and API Gateway reverse proxy.

## Subpackages

- **`web`**: HTTP router, route matching, URL parameters, JSON helpers, and middleware (Recover, Logging, RequestID).
- **`web/ssg`**: Static site generator, HTML minification, clean URLs, and Markdown pipeline.
- **`web/ssg/hmr`**: WebSocket live-reload and Hot Module Replacement for `kiw serve`.
- **`web/gateway`**: Reverse proxy with rate limiting, timeouts, CORS, and upstream service discovery.

## Example

```go
package main

import (
	"net/http"
	"github.com/krewire/krewire/packages/web"
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
