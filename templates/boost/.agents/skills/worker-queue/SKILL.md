---
name: worker-queue
description: Use for Krewire background workers — job queues, cron, retries, DLQ, krewire worker runner. Triggers: "worker", "job queue", "cron", "DLQ", "retry", "background job".
---

# Worker Queue — Background Jobs

Background work with durable queues. This skill covers `framework/worker` (spec `KWF-L5H2F` § worker).

## 1. Detect the context

- Read `docs/project-vision.md` and `docs/specs/framework/KWF-SVC-L5H2F-microservice-patterns.md` (§ worker).
- Check `project.kind: worker` or `worker:` in `krewire.yaml`; `framework/worker` may be used inside `app` as well.

## 2. Job & queue contract

- `Job`: `Run(ctx) error`; queue ops `Enqueue(Job, Options{Priority,Delay,Cron})`, `Dequeue`, `Ack`, `Nack`.
- Options: priority levels, delayed execution, cron expressions (queue-native, not separate scheduler).
- Backends: NATS, Redis, PostgreSQL (advisory locks), in-memory (default for dev — zero external deps).

## 3. Runner

- `krewire worker` reads `worker:` config or container-registered jobs; it is the native runner, not an external binary.
- Jobs are declared in `krewire.yaml` and/or registered in `framework/app` container.

## 4. Retries & DLQ

- Retry policy is part of the queue contract (configurable attempts, backoff).
- Poison messages after exhaustion go to DLQ; inspect via `krewire worker dlq list`, replay via `dlq replay`.
- Messaging DLQ (`service/messaging` poison) routes through the same worker DLQ.

## 5. Cron

- Cron jobs are enqueued as delayed jobs with cron schedule; no separate scheduler process in v1.

## 6. Gates

- `go test ./...` uses in-memory backend; NATS/Redis/Postgres behind build tags.
- Fixture: enqueue delayed job with retry policy through NATS → process → verify poison variant lands in DLQ inspectable via CLI.

## Rules

Read `rules/key-rules.md` before acting.
