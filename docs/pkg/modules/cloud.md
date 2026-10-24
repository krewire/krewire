# `cloud`

Import: `github.com/krewire/krewire/pkg/cloud`

## Purpose

Cloud orchestration suite providing container runners, microservice discovery, asynchronous queue workers, cloud object storage, and declarative infrastructure blueprints.

## Subpackages

- **`cloud/runner`**: Container lifecycle and workload task runner.
- **`cloud/service`**: Microservice coordination, registry (`registry.MemoryRegistry`), distributed tracing, and messaging.
- **`cloud/worker`**: Distributed background job queues and dead-letter queue (DLQ) supervision.
- **`cloud/storage`**: Cloud-specific object store drivers.
- **`cloud/infra`**: Declarative schema definitions and state management for infrastructure blueprints.
- **`cloud/deploy`**: Canary and blue-green deployment pipelines.

## Example

```go
package main

import (
	"context"
	"github.com/krewire/krewire/pkg/cloud/worker"
)

func main() {
	w := worker.New(worker.Config{
		Concurrency: 4,
		Queues:      []string{"default", "critical"},
	})
	_ = w.Start(context.Background())
}
```

## Design boundary

Implements `worker`, `service`, and `infra` workload kinds. See the detailed [Cloud Platform Reference](../platform/cloud.md).
