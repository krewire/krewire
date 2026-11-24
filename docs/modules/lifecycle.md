# Execution & Lifecycle Supervision (`lifecycle`)

Import: `github.com/krewire/krewire/lifecycle`

## Purpose

The `lifecycle` package provides the imperative control plane that boots, supervises, and executes Krewire workloads. It acts as the execution mechanics counterpart to the declarative models in `model`.

---

## Architecture & Subsystems

The package is composed of four coordinated subsystems:

```text
                  ┌──────────────────────┐
                  │        Kernel        │
                  │ (Top-level orchestr) │
                  └──────────┬───────────┘
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│    Registry     │ │   Supervisor    │ │    Executor     │
│ (Topo ordering) │ │ (Health & Stop) │ │(Task dispatch)  │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

1. **Kernel (`Kernel`)**: Central orchestrator initialized with a validated `model.Project`. Coordinates module initialization and runtime dispatch.
2. **Registry (`Registry`)**: Manages modular components (`Module`), resolving dependencies into an acyclic topological startup sequence.
3. **Supervisor (`Supervisor`)**: Manages long-running concurrent processes (`Startable`), handling graceful cancellation upon signals and health reporting.
4. **Executor (`Executor`)**: Dispatches project execution tasks and handlers.

---

## Core Interfaces & Types

### 1. `Module`
Implemented by components that require initialization against the kernel:

```go
type Module interface {
    Name() string
    Init(*Kernel) error
}

// Optional interface declaring dependency ordering:
type DependentModule interface {
    Module
    DependsOn() []string
}
```

### 2. `Startable`
Implemented by long-running background daemons, servers, or workers:

```go
type Startable interface {
    Start(context.Context) error
    Stop(context.Context) error
}
```

---

## Main API

- **`New(project model.Project) (*Kernel, error)`**: Initializes and validates a Kernel instance.
- **`(*Kernel).Register(mod Module) error`**: Registers a module into the kernel registry.
- **`(*Kernel).Init() error`**: Resolves topological module order and calls `Init` on each.
- **`(*Kernel).Execute(ctx context.Context, w model.Workload[string]) errs.ExitCode`**: Dispatches workload execution.
- **`(*Supervisor).Register(mod Startable)`**: Enrolls a startable service into supervisor management.
- **`(*Supervisor).Start(ctx context.Context) error`**: Boots all registered startables concurrently.
- **`(*Supervisor).Stop(ctx context.Context) error`**: Concurrently stops all services within `DefaultShutdownTimeout` (10 seconds).

---

## Example Usage

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/lifecycle"
	"github.com/krewire/krewire/model"
)

type WorkerService struct{}

func (w *WorkerService) Start(ctx context.Context) error {
	fmt.Println("Worker service started")
	<-ctx.Done()
	return nil
}

func (w *WorkerService) Stop(ctx context.Context) error {
	fmt.Println("Worker service stopping gracefully...")
	return nil
}

func main() {
	proj := model.Project{
		Name: "task-worker",
	}

	kernel, err := lifecycle.New(proj)
	if err != nil {
		panic(err)
	}

	worker := &WorkerService{}
	kernel.Supervisor.Register(worker)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := kernel.Supervisor.Start(ctx); err != nil {
		fmt.Printf("Startup error: %v\n", err)
	}
}
```

---

## Design Boundaries

- **Pure Mechanics**: `lifecycle` owns process execution mechanics, not application routing, business models, or config parsing.
- **Standard Library Primitives**: Built directly on `context.Context`, `sync.Mutex`, and standard channel traps.
