package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when an operation is rejected due to rate limiting.
	ErrRateLimitExceeded = errors.New("resilience: rate limit exceeded")
)

// RateLimiter controls execution frequency using the token bucket algorithm.
type RateLimiter struct {
	mu         sync.Mutex
	rate       float64 // tokens added per second
	capacity   float64 // maximum token capacity (burst)
	tokens     float64
	lastRefill time.Time
}

// NewRateLimiter creates a RateLimiter that produces rps tokens per second with a burst capacity.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if rps <= 0 {
		rps = 1.0
	}
	if burst <= 0 {
		burst = int(rps)
		if burst <= 0 {
			burst = 1
		}
	}
	return &RateLimiter{
		rate:       rps,
		capacity:   float64(burst),
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// Allow reports whether 1 token is available, consuming it immediately if so.
func (rl *RateLimiter) Allow() bool {
	return rl.AllowN(time.Now(), 1)
}

// AllowN reports whether n tokens are available at now, consuming them if so.
func (rl *RateLimiter) AllowN(now time.Time, n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill(now)
	if rl.tokens >= float64(n) {
		rl.tokens -= float64(n)
		return true
	}
	return false
}

// Wait blocks until 1 token is available or ctx is canceled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		now := time.Now()
		rl.mu.Lock()
		rl.refill(now)
		if rl.tokens >= 1.0 {
			rl.tokens -= 1.0
			rl.mu.Unlock()
			return nil
		}

		// Calculate sleep duration until next token arrives
		needed := 1.0 - rl.tokens
		sleepDur := time.Duration((needed / rl.rate) * float64(time.Second))
		rl.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDur):
		}
	}
}

func (rl *RateLimiter) refill(now time.Time) {
	elapsed := now.Sub(rl.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
	rl.lastRefill = now
}
