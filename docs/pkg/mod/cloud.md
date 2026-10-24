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
	"log"

	"github.com/krewire/krewire/pkg/cloud/worker"
)

type GreetJob struct{}

func (GreetJob) Run(ctx context.Context) error {
	log.Println("hello from a Krewire worker job")
	return nil
}

func main() {
	ctx := context.Background()

	q := worker.NewInMemoryQueue()
	if _, err := q.Enqueue(ctx, GreetJob{}, worker.Options{Priority: 1}); err != nil {
		log.Fatal(err)
	}

	task, err := q.Dequeue(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := task.Job.Run(ctx); err != nil {
		task.LastErr = err
		_ = q.Nack(ctx, task)
		log.Fatal(err)
	}
	if err := q.Ack(ctx, task); err != nil {
		log.Fatal(err)
	}
	log.Println("dead letters:", len(q.DLQ()))
}
```

For long-running consumption, wrap the same `Queue` in `worker.NewRunner(q, worker.WithConcurrency(n))` and call `Run(ctx)`; it dequeues with bounded concurrency, acks successes, and nacks failures so the queue applies retries and dead-lettering.

## Design boundary

Implements `worker`, `service`, and `infra` workload kinds. See the detailed [Cloud Platform Reference](../platform/cloud.md).
