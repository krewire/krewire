# Workload: Background Worker (`worker`)

The `worker` workload powers autonomous background processing daemons, asynchronous job consumers, scheduled cron routines, and event-driven data ingestion.

---

## 1. Key Capabilities

- **Supervised Lifecycle (`pkg/lifecycle`)**: Integrates with the kernel supervisor for automatic restart, health heartbeats, and graceful cancellation upon SIGTERM/SIGINT signals.
- **Queue & Consumer Primitives (`pkg/cloud/worker`)**: Pluggable queue abstractions supporting Redis, PostgreSQL, and in-memory test queues.
- **Concurrency & Resilience (`pkg/resilience`)**: Bounded concurrency pools (preventing thread/goroutine exhaustion), exponential backoff retries with jitter, and circuit breakers.
- **Zero-CGO & Minimal Memory Footprint**: Runs in a lean 10–30 MB RAM profile on modest VPS nodes ($5/mo).

---

## 2. Directory Layout

The `krew new my-worker --worker` or `krew init --worker` scaffold produces:

```text
my-worker/
├── krewire.yaml
├── main.go
├── internal/
│   └── jobs/             # Job definitions and consumer handlers
│       └── email.go
└── README.md
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-worker

worker:
  concurrency: 10
  poll_interval: 1s
  queues:
    - critical
    - default
    - low-priority

database:
  driver: redis
  dsn: "redis://127.0.0.1:6379/0"
```

---

## 4. Implementation Example

### `internal/jobs/email.go`
```go
package jobs

import (
	"context"
	"fmt"
	"time"
)

type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func HandleSendEmail(ctx context.Context, payload SendEmailPayload) error {
	select {
	case <-time.After(100 * time.Millisecond):
		fmt.Printf("Email sent to %s: %s\n", payload.To, payload.Subject)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

### `main.go`
```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/pkg/env"
	"github.com/krewire/krewire/pkg/lifecycle"
	"github.com/krewire/krewire/pkg/log"
	"github.com/krewire/krewire/pkg/model"
)

func main() {
	logger := log.Install(env.EnvProduction, false)
	logger.Info("starting background worker daemon...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	proj := model.Project{
		Name: "my-worker",
	}

	kernel, err := lifecycle.New(proj)
	if err != nil {
		logger.Error("kernel initialization failed", "error", err)
		os.Exit(1)
	}

	if err := kernel.Supervisor.Start(ctx); err != nil {
		logger.Error("worker supervisor stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("worker shut down gracefully")
}
```

---

## 5. Development & Production Commands

```bash
# Run worker in local development mode with live restart
krew dev

# Run worker once directly
kiw worker

# Build production binary into .krewire/build/bin/my-worker
krew build

# Run unit and integration tests
krew test
```
