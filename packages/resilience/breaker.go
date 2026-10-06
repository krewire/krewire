package resilience

import (
	"context"
	"sync"
	"time"
)

// BreakerState is the breaker's current disposition.
type BreakerState uint8

const (
	// Closed lets every call through and counts failures.
	Closed BreakerState = iota
	// Open refuses every call until the cooldown elapses.
	Open
	// HalfOpen admits a single probe call to test whether the dependency
	// recovered; a success closes the breaker, a failure reopens it.
	HalfOpen
)

// String names the state for diagnostics.
func (s BreakerState) String() string {
	switch s {
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// CircuitBreaker stops calling an operation that is failing, so a caller does
// not spend its own timeout budget waiting on a dependency that is already
// down. It opens after failureThreshold consecutive failures, stays open for
// cooldown, then admits one probe.
//
// A CircuitBreaker is safe for concurrent use; the zero value is not usable,
// call NewCircuitBreaker.
type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	cooldown         time.Duration
	failures         int
	state            BreakerState
	openedAt         time.Time
	probeInFlight    bool
}

// NewCircuitBreaker returns a breaker that opens after failureThreshold
// consecutive failures and stays open for cooldown. A threshold below 1 is
// raised to 1 and a negative cooldown is treated as 0, so a misconfigured
// breaker still gates traffic rather than never opening.
func NewCircuitBreaker(failureThreshold int, cooldown time.Duration) *CircuitBreaker {
	if failureThreshold < minFailureThreshold {
		failureThreshold = minFailureThreshold
	}
	if cooldown < 0 {
		cooldown = 0
	}
	return &CircuitBreaker{failureThreshold: failureThreshold, cooldown: cooldown}
}

// State returns the breaker's current state, applying a pending open-to-half-open
// transition if the cooldown has elapsed.
func (b *CircuitBreaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked(time.Now())
	return b.state
}

// Do executes operation when the circuit allows it and updates breaker state from its result.
func (b *CircuitBreaker) Do(ctx context.Context, operation func(context.Context) error) error {
	if b == nil {
		return ErrCircuitOpen
	}
	if operation == nil {
		return ErrNilOperation
	}
	if ctx == nil {
		ctx = context.Background()
	}
	b.mu.Lock()
	b.transitionLocked(time.Now())
	if b.state == Open || (b.state == HalfOpen && b.probeInFlight) {
		b.mu.Unlock()
		return ErrCircuitOpen
	}
	if b.state == HalfOpen {
		b.probeInFlight = true
	}
	b.mu.Unlock()
	err := operation(ctx)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.probeInFlight = false
	if err == nil {
		b.failures = 0
		b.state = Closed
		return nil
	}
	b.failures++
	if b.failures >= b.failureThreshold {
		b.state = Open
		b.openedAt = time.Now()
	}
	return err
}

func (b *CircuitBreaker) transitionLocked(now time.Time) {
	if b.state == Open && now.Sub(b.openedAt) >= b.cooldown {
		b.state = HalfOpen
		b.failures = 0
	}
}
