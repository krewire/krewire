// Package resilience provides failure-handling utilities for closure-based
// workloads: a retry policy with exponential backoff (retry.go), and a circuit
// breaker that stops calling an operation which is already failing (breaker.go).
package resilience

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

var (
	// ErrNilOperation reports that Retry or CircuitBreaker.Do was handed no
	// function to call.
	ErrNilOperation = errors.New("resilience: nil operation")
	// ErrCircuitOpen reports that the breaker refused to call the operation
	// because it is open (or a half-open probe is already in flight).
	ErrCircuitOpen = errors.New("resilience: circuit breaker is open")
)

// Backoff defaults, named because they are policy rather than arithmetic: a
// caller who sets InitialDelay but no Multiplier gets doubling.
const (
	// DefaultBackoffMultiplier is the growth factor applied per attempt when
	// RetryOptions.Multiplier is unset or below 1.
	DefaultBackoffMultiplier = 2
	// defaultMinAttempts is the attempt count used when MaxAttempts is unset
	// or below 1: one try, no retries.
	defaultMinAttempts = 1
	// minFailureThreshold is the breaker threshold used when the requested
	// one is below 1, so a breaker can never open on its first failure
	// unless that is what the caller asked for.
	minFailureThreshold = 1
)

// RetryOptions configures Retry. The zero value is valid and performs a single
// attempt with no delay.
type RetryOptions struct {
	// MaxAttempts is the total number of calls, including the first. Values
	// below 1 are treated as 1.
	MaxAttempts int
	// InitialDelay is the wait before the second attempt. Zero disables
	// waiting entirely.
	InitialDelay time.Duration
	// MaxDelay caps a single backoff interval. Zero means uncapped.
	MaxDelay time.Duration
	// Multiplier is the growth factor per attempt. Values below 1 fall back
	// to DefaultBackoffMultiplier.
	Multiplier float64
	// Jitter is the proportional randomization applied to each delay, in
	// (0, 1]. At 0.2 a delay varies by plus or minus 20 percent, which keeps
	// concurrent callers from retrying in lockstep. Zero disables jitter.
	Jitter float64
	// ShouldRetry, when set, decides per error whether another attempt is
	// worth making. Returning false returns that error immediately. A nil
	// function retries every error.
	ShouldRetry func(error) bool
}

// delay returns the wait before the given 1-based attempt number. It is pure
// except for the jitter draw, so a policy can be asserted directly in tests.
//
// Jitter is applied before the MaxDelay cap, not after. Capping first and then
// jittering lets the randomization push an interval past the ceiling the caller
// set, which would make MaxDelay unbounded in effect.
func (o RetryOptions) delay(attempt int) time.Duration {
	if o.InitialDelay <= 0 {
		return 0
	}
	multiplier := o.Multiplier
	if multiplier < 1 {
		multiplier = DefaultBackoffMultiplier
	}
	delay := float64(o.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= multiplier
	}
	if o.Jitter > 0 {
		delay *= 1 + (rand.Float64()*2-1)*o.Jitter
	}
	if delay < 0 {
		return 0
	}
	if o.MaxDelay > 0 && time.Duration(delay) > o.MaxDelay {
		return o.MaxDelay
	}
	return time.Duration(delay)
}

// Retry executes operation until it succeeds, the policy is exhausted, or ctx
// is cancelled. It returns the last error from operation, or the context error
// when ctx ends first.
func Retry(ctx context.Context, options RetryOptions, operation func(context.Context) error) error {
	if operation == nil {
		return ErrNilOperation
	}
	if ctx == nil {
		ctx = context.Background()
	}
	attempts := options.MaxAttempts
	if attempts < 1 {
		attempts = defaultMinAttempts
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := operation(ctx)
		if err == nil {
			return nil
		}
		last = err
		if options.ShouldRetry != nil && !options.ShouldRetry(err) {
			return err
		}
		if attempt == attempts {
			break
		}
		timer := time.NewTimer(options.delay(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}
