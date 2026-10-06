---
description: Krewire worker — job queues, cron, retries, DLQ, krewire worker runner. Optionally: /worker [enqueue|retry|dlq|cron].
agent: build
---

Execute a Krewire worker task. $ARGUMENTS is the operation; default is to map from the request.

1. Load `vision` + `worker-queue` skills and `KWF-L5H2F` worker section.
2. Delegate to the `worker` subagent: `Job` contract, queue ops (`Enqueue`/`Dequeue`/`Ack`/`Nack` with `Priority`/`Delay`/`Cron`), backends (NATS/Redis/Postgres/in-memory), `krewire worker` runner, retries + DLQ.
3. Default to in-memory backend for local gates; verify enqueue→process→DLQ fixture and `krewire worker dlq list`.

Report backend used, jobs processed/retried/DLQ'd, and runner evidence.
