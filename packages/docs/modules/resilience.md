# `resilience`

Import: `github.com/krewire/libs/resilience`

## Purpose

`resilience` provides failure-handling primitives for closure-based workloads. It does not perform I/O or assume HTTP, CLI, worker, or storage semantics.

## Main API

- `Retry` — retries an operation with bounded attempts, exponential delay, optional jitter, and context cancellation.
- `RetryOptions` — configures retry behavior and an optional retry predicate.
- `NewCircuitBreaker` — creates a breaker that opens after consecutive failures and probes again after a cooldown.
- `CircuitBreaker.Do` — executes an operation while the circuit is closed or half-open.
- `CircuitBreaker.State` — returns `Closed`, `Open`, or `HalfOpen`.

## Example

```go
err := resilience.Retry(ctx, resilience.RetryOptions{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     time.Second,
    Jitter:       0.2,
    ShouldRetry:  isTransient,
}, func(ctx context.Context) error {
    return client.Send(ctx)
})
```

## Design boundary

Retry policies should be used only for operations that are safe to repeat or provide idempotency. The package returns the last operation error after attempts are exhausted and returns context errors when cancellation or deadlines win.
