# Package: `cloud` (Cloud Runtime & Infra Blueprint Engine)

The `pkg/cloud` suite provides the cloud execution plane, task runners, service discovery, background queues, cloud object storage abstractions, declarative infrastructure state management, and production deployment targets.

Held strictly to the **Three Pillars (3S)**, `pkg/cloud` provides cloud-native primitives in 100% pure Go without heavy Kubernetes or external agent runtime dependencies.

---

## 1. Architectural Overview & Subsystem Map

`pkg/cloud` is organized into six cohesive subsystems:

```text
pkg/cloud/
├── runner/     # Workload execution contract (Runner, Func) & lifecycle binding
├── service/    # Microservice coordination, registry, discovery, tracing & messaging
├── worker/     # Distributed job queue, cron scheduler, retry policies & dead-letter queue
├── storage/    # Cloud-agnostic object storage provider abstractions
├── infra/      # Declarative infrastructure blueprints, schema validation & state locks
└── deploy/     # Deployment orchestration (binary, systemd, GitHub Pages, infra blueprints)
```

---

## 2. Workload Runner (`pkg/cloud/runner`)

The `runner` package establishes the execution contract for all Krewire workload runtimes once dependencies and providers are registered.

### Core Contracts

```go
package runner

import (
    "context"
    "github.com/krewire/krewire/pkg/cloud/service"
)

// Runner executes one application runtime after services have been registered and started.
type Runner interface {
    Run(ctx context.Context, registry service.Registry) error
}

// Func adapts a standard function into a Runner.
type Func func(context.Context, service.Registry) error

func (f Func) Run(ctx context.Context, registry service.Registry) error {
    if f == nil {
        return nil
    }
    return f(ctx, registry)
}
```

### Usage Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/krewire/krewire/pkg/cloud/runner"
    "github.com/krewire/krewire/pkg/cloud/service"
)

func main() {
    customRunner := runner.Func(func(ctx context.Context, reg service.Registry) error {
        fmt.Println("Workload runner executing within registry context...")
        return nil
    })

    // Executed by the workload lifecycle supervisor
    _ = customRunner.Run(context.Background(), nil)
}
```

---

## 3. Microservice Coordination (`pkg/cloud/service`)

The `service` package defines the dependency boundary and lifecycle contracts for modular services.

### Core Interfaces

```go
package service

import "context"

// Registry provides service registration and dependency lookup.
type Registry interface {
    Set(name string, value any) error
    Get(name string) (any, bool)
}

// Provider contributes a named service to the application.
type Provider interface {
    Name() string
    Register(Registry) error
}

// Starter is implemented by providers that perform asynchronous initialization.
type Starter interface {
    Start(context.Context, Registry) error
}

// Stopper is implemented by providers that require graceful shutdown.
type Stopper interface {
    Stop(context.Context, Registry) error
}
```

### Subpackages

- **`service/registry`**: In-memory and distributed service discovery registries.
- **`service/config`**: Dynamic configuration center with real-time reload listeners.
- **`service/messaging`**: High-performance asynchronous pub/sub event bus.
- **`service/tracing`**: Context-propagated distributed tracing spans.
- **`service/resilience`**: Circuit breaker, bulkhead limits, and timeout protection for remote calls.

---

## 4. Distributed Job Worker & Queue (`pkg/cloud/worker`)

The `worker` package provides a storage-agnostic, concurrency-bounded background queue engine supporting one-shot tasks, delays, cron schedules, exponential backoff retries, and dead-letter queues.

### Core Interfaces & Types

```go
package worker

import (
    "context"
    "time"
)

type Job interface {
    Run(ctx context.Context) error
}

type JobID string

type Options struct {
    Priority int
    Delay    time.Duration
    Cron     string
    Retry    *RetryPolicy
}

type DeadLetter struct {
    ID       JobID
    Job      Job
    Err      error
    Attempts int
    At       time.Time
}

type Queue interface {
    Enqueue(ctx context.Context, job Job, opts Options) (JobID, error)
    Dequeue(ctx context.Context) (*Task, error)
    Ack(ctx context.Context, task *Task) error
    Nack(ctx context.Context, task *Task) error
    DLQ() []DeadLetter
}
```

### Complete Worker Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/krewire/krewire/pkg/cloud/worker"
)

type EmailJob struct {
    To      string
    Subject string
}

func (j *EmailJob) Run(ctx context.Context) error {
    fmt.Printf("Sending email to %s: %s\n", j.To, j.Subject)
    return nil
}

func main() {
    q := worker.NewMemoryQueue()
    ctx := context.Background()

    // 1. Enqueue job with retry and delay
    id, err := q.Enqueue(ctx, &EmailJob{
        To:      "dev@example.com",
        Subject: "Deploy Succeeded",
    }, worker.Options{
        Priority: 10,
        Delay:    100 * time.Millisecond,
        Retry: &worker.RetryPolicy{
            MaxAttempts: 3,
            Backoff:     worker.BackoffExponential,
            BaseDelay:   time.Second,
        },
    })
    if err != nil {
        panic(err)
    }

    // 2. Start worker runner
    runner := worker.NewRunner(q, worker.RunnerConfig{
        Concurrency: 4,
        PollInterval: 50 * time.Millisecond,
    })
    go runner.Start(ctx)

    time.Sleep(500 * time.Millisecond)
    runner.Stop()
    fmt.Printf("Processed task: %s\n", id)
}
```

---

## 5. Declarative Infrastructure Blueprints (`pkg/cloud/infra`)

The `infra` package defines infrastructure as code primitives entirely in Go. The lifecycle contract (`infra/contract`), state store (`infra/store`), canonical engine (`infra/engine`), conformance suite, and reference providers are documented in the [infra workload guide](../../workloads/infra.md). The snippet below shows the older `infra/provider` surface kept for compatibility.

### Contracts & Plan Engine

```go
type Op string

const (
    OpCreate Op = "create"
    OpUpdate Op = "update"
    OpDelete Op = "delete"
    OpNoOp   Op = "noop"
)

type Resource struct {
    Kind       string            `json:"kind"`
    ID         string            `json:"id"`
    Properties map[string]any    `json:"properties,omitempty"`
    DependsOn  []string          `json:"dependsOn,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

type Action struct {
    Op       Op       `json:"op"`
    Resource Resource `json:"resource"`
    Reason   string   `json:"reason,omitempty"`
}

type Plan struct {
    Actions []Action `json:"actions"`
}
```

- **Topological Sorting**: Resources are provisioned strictly in dependency order (`DependsOn`).
- **State Locking (`infra/state`)**: Prevents race conditions during concurrent provisioning runs.
- **Pluggable Providers (`infra/provider`)**: Clean driver boundary supporting Hetzner, AWS, DigitalOcean, and local mock environments.

---

## 6. Deployment Orchestration (`pkg/cloud/deploy`)

The `deploy` package provides unified deployment engines for Krewire artifacts:

- **`TargetBinary`**: Packages standalone static Go binaries and copies them to remote servers.
- **`TargetSystemd`**: Scaffolds and installs production systemd unit files (`ExecStart`, restart policies, Sandboxed user/group).
- **`TargetGhPages`**: Automated deployment of static SSG artifacts to GitHub Pages.
- **`TargetInfra`**: Orchestrates declarative blueprint provisioning across target environments.

### Systemd Deployment Configuration

```go
deployer := deploy.Registry[deploy.TargetSystemd]
result, err := deployer.Deploy(ctx, deploy.Options{
    ServiceName: "krewire-app",
    BinaryPath:  "/opt/krewire/bin/app",
    User:        "krewire",
    Group:       "krewire",
    WorkingDir:  "/opt/krewire",
    Restart:     "always",
    EnvFile:     "/opt/krewire/.env",
})
```

---

## 7. Compliance with the Three Pillars (3S)

- **Secure (S1)**:
  - Secrets are injected strictly through environment variables or secure key vaults; no credentials or state tokens are committed.
  - Workload runners operate with least-privilege sandboxing and systemd process confinement.
- **Sustainable (S2)**:
  - Worker runners enforce strictly bounded concurrency pools (default 4–8 goroutines), keeping memory under 15 MB.
  - Zero CGO dependencies (`CGO_ENABLED=0`); builds compile into single static binaries.
- **Scalable (S3)**:
  - In-memory worker queues swap transparently to Redis or PostgreSQL backends without modifying business logic.
  - Decoupled service registries and event buses enable progressive scaling from single-node monoliths to distributed microservices.
