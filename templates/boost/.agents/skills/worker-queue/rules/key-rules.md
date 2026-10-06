# Worker Queue — Key Rules

- Queue contract is law: `Enqueue`/`Dequeue`/`Ack`/`Nack` + retries + DLQ are backend-agnostic.
- In-memory backend is default for local dev; `krewire worker` must run without external services.
- Cron is queue-native via delayed jobs; no separate scheduler.
- Handler errors auto-nack/redeliver; poison after retry exhaustion goes to DLQ.
- DLQ is CLI-inspectable (`krewire worker dlq list` / `replay`).
- Tests use in-memory backend by gate; NATS/Redis/Postgres only behind build tags.
