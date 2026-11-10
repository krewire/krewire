# `resilience`

Import: `github.com/krewire/krewire/pkg/resilience`

## Purpose

The `pkg/resilience` package provides robust fault-tolerance primitives for closure-based workloads. Designed with zero third-party dependencies, it equips workloads with exponential retries, circuit breaking, concurrency bulkheading, and token-bucket rate limiting.

---

## Main API

### 1. Concurrency Bulkheading (`Bulkhead`)
- `NewBulkhead(limit int) *Bulkhead` — Creates a bulkhead that bounds concurrent operations to at most `limit` active slots (default: 10).
- `Bulkhead.Execute(ctx context.Context, fn func(ctx context.Context) error) error` — Executes `fn` within the bulkhead boundary, blocking until a slot is available or `ctx` is canceled.
- `Bulkhead.Acquire(ctx context.Context) (func(), error)` — Manually acquires an execution slot, returning a release callback.
- `Bulkhead.TryAcquire() (func(), bool)` — Non-blocking acquisition returning false when the bulkhead is saturated.
- `Bulkhead.Running() int`, `Bulkhead.Available() int`, `Bulkhead.Limit() int` — Observability metrics.
- `Bulkhead.Close()` — Gracefully closes the bulkhead, rejecting subsequent requests.
- `ErrBulkheadFull`, `ErrBulkheadClosed`.

### 2. Token-Bucket Rate Limiting (`RateLimiter`)
- `NewRateLimiter(rps float64, burst int) *RateLimiter` — Creates a token-bucket rate limiter that generates `rps` tokens per second with maximum `burst` capacity.
- `RateLimiter.Allow() bool` — Consumes 1 token immediately if available without blocking.
- `RateLimiter.AllowN(now time.Time, n int) bool` — Consumes `n` tokens at a specific timestamp if available.
- `RateLimiter.Wait(ctx context.Context) error` — Blocks until 1 token is available or `ctx` is canceled.
- `ErrRateLimitExceeded`.

### 3. Exponential Retry with Jitter (`Retry`)
- `Retry(ctx context.Context, opts RetryOptions, fn func(ctx context.Context) error) error` — Retries an operation with bounded attempts, exponential backoff, randomized jitter, and context cancellation.
- `RetryOptions`:
  - `MaxAttempts int`: Maximum execution attempts (default: 3).
  - `InitialDelay time.Duration`: Initial backoff period.
  - `MaxDelay time.Duration`: Upper ceiling for backoff delays.
  - `Jitter float64`: Randomization factor (0.0 to 1.0).
  - `ShouldRetry func(error) bool`: Predicate determining if an error is transient.

### 4. Circuit Breakers (`CircuitBreaker`)
- `NewCircuitBreaker(opts CircuitBreakerOptions) *CircuitBreaker` — Protects downstream dependencies by opening after consecutive failures.
- `CircuitBreaker.Do(ctx context.Context, fn func(ctx context.Context) error) error` — Runs `fn` when closed or half-open; fails fast when open.
- `CircuitBreaker.State()` — Reports `StateClosed`, `StateOpen`, or `StateHalfOpen`.

---

## Examples

### Concurrency Bulkhead & Rate Limiting
```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/krewire/krewire/pkg/resilience"
)

func main() {
	ctx := context.Background()

	// 1. Bound concurrent executions to 5 goroutines
	bulkhead := resilience.NewBulkhead(5)

	// 2. Limit request throughput to 100 RPS with burst of 20
	limiter := resilience.NewRateLimiter(100.0, 20)

	// Execute task safely through both guards
	if err := limiter.Wait(ctx); err != nil {
		panic(err)
	}

	err := bulkhead.Execute(ctx, func(innerCtx context.Context) error {
		// Business logic executed with guaranteed bounded concurrency
		fmt.Println("Executing task within bulkhead slot")
		return nil
	})
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
```

### Combined Retry & Circuit Breaker
```go
package main

import (
	"context"
	"time"

	"github.com/krewire/krewire/pkg/resilience"
)

func main() {
	ctx := context.Background()

	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerOptions{
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
	})

	err := cb.Do(ctx, func(cCtx context.Context) error {
		return resilience.Retry(cCtx, resilience.RetryOptions{
			MaxAttempts:  3,
			InitialDelay: 50 * time.Millisecond,
			MaxDelay:     500 * time.Millisecond,
			Jitter:       0.2,
		}, func(rCtx context.Context) error {
			// Call remote microservice
			return nil
		})
	})
	if err != nil {
		fmt.Printf("Service unavailable: %v\n", err)
	}
}
```
