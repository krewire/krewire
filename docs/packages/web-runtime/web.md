# Package: `web` (HTTP Router, SSG, SSR & Gateway)

The `packages/web` library provides high-performance web routing, middleware pipelines, Static Site Generation (SSG), Server-Side Rendering (SSR), and WebSocket-driven Hot Module Replacement (HMR).

---

## 1. Subsystems

- **`web`**: Core HTTP multiplexer, route group matching, context wrappers, and JSON streaming.
- **`web/ssg`**: Static asset builder, image optimizer (WebP), HTML minifier, and clean URL generation.
- **`web/ssg/hmr`**: Local WebSocket server broadcasting file change events to the browser client during `kiw serve`.
- **`web/gateway`**: Reverse proxy, rate limiting, and route-based authorization filters.

---

## 2. Router Usage Example

```go
package main

import (
	"net/http"
	"github.com/krewire/krewire/packages/web"
)

func main() {
	r := web.NewRouter()

	// Global middleware
	r.Use(web.RecoveryMiddleware())
	r.Use(web.LoggerMiddleware())

	// Route definitions
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello from Krewire Web!"))
	})

	r.Get("/api/v1/health", func(w http.ResponseWriter, req *http.Request) {
		web.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	http.ListenAndServe(":8080", r)
}
```
