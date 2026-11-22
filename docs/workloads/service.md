# Workload: Microservice (`service`)

The `service` workload targets headless backend microservices, RPC endpoints, and RESTful APIs optimized for high throughput, low latency, and zero-downtime operations.

---

## 1. Key Capabilities

- **High-Throughput HTTP Routing (`pkg/web`)**: Lightweight standard-library-aligned router with route groups and JSON serialization.
- **Observability & Tracing (`pkg/web`, `pkg/log`)**: Automatic propagation of W3C `traceparent` headers, request IDs, and structured logging.
- **Service Resilience (`pkg/resilience`)**: Integrated rate limiting, bulkheads, and circuit breakers preventing cascading service failures.
- **Health Check Endpoints**: Standardized liveness (`/healthz`) and readiness (`/readyz`) probe endpoints for container orchestrators.

---

## 2. Directory Layout

The `krew new my-service --service` or `krew init --service` scaffold writes:

```text
my-service/
├── krewire.yaml
├── main.go
├── internal/
│   └── handler/          # HTTP request handlers & controllers
│       ├── health.go
│       └── users.go
└── README.md
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-service

server:
  host: "0.0.0.0"
  port: 9000

service:
  name: "user-service"
  timeout: 10s
  cors:
    origins:
      - "https://app.krewire.com"
```

---

## 4. Implementation Example

### `internal/handler/health.go`
```go
package handler

import (
	"net/http"
	"github.com/krewire/krewire/pkg/web"
)

func HealthCheck(c *web.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
		"system": "krewire-service",
	})
}
```

### `main.go`
```go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krewire/krewire/internal/handler"
	"github.com/krewire/krewire/pkg/env"
	"github.com/krewire/krewire/pkg/log"
	"github.com/krewire/krewire/pkg/web"
)

func main() {
	logger := log.Install(env.EnvProduction, false)

	router := web.NewRouter()
	router.Get("/healthz", handler.HealthCheck)
	router.Get("/readyz", handler.HealthCheck)

	srv := &http.Server{
		Addr:         ":9000",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("microservice listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failure", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down microservice...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
```

---

## 5. Development & Production Commands

```bash
# Start service with hot reload on code changes
krew dev

# Run service once directly
krew run

# Build production binary into .krewire/build/bin/my-service
krew build

# Run security posture check and tests
krew vet --sec
krew test
```
