# Package: `kern` (Kernel Executor & Supervision)

The `packages/kern` library provides the generic lifecycle executor, module registration, process supervision, and boot orchestration for Krewire applications (`KWL-KERN-X8P3L`).

---

## 1. Responsibilities

- **Zero-Dependency Floor (`KWL-LAYER-002`)**: Sits at the bottom of the dependency graph; requires no other ecosystem packages.
- **Module Lifecycle Management**: Orchestrates modular boot sequences across three distinct phases: `Init`, `Start`, and `Stop`.
- **Supervision & Recovery**: Supervises long-running background routines, monitors heartbeats, and guarantees graceful termination upon system interruption signals (`SIGINT`, `SIGTERM`).

---

## 2. Core Types & Interface

```go
package kern

import "context"

// Module represents a self-contained functional subsystem.
type Module interface {
	Name() string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Supervisor manages concurrent background modules.
type Supervisor struct {
	// ...
}

func NewSupervisor() *Supervisor
func (s *Supervisor) Register(m Module)
func (s *Supervisor) Run(ctx context.Context) error
```

---

## 3. Usage Example

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/packages/kern"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sup := kern.NewSupervisor()
	// Register modules...
	if err := sup.Run(ctx); err != nil {
		os.Exit(1)
	}
}
```
