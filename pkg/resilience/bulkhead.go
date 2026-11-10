package resilience

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrBulkheadFull is returned when all execution slots in the bulkhead are occupied.
	ErrBulkheadFull = errors.New("resilience: bulkhead is full")

	// ErrBulkheadClosed is returned when an operation is attempted on a closed bulkhead.
	ErrBulkheadClosed = errors.New("resilience: bulkhead is closed")
)

// Bulkhead bounds concurrent calls to an operation or dependency, preventing
// thread/goroutine exhaustion from cascading across services.
type Bulkhead struct {
	limit   int
	sem     chan struct{}
	closed  chan struct{}
	closeMu sync.RWMutex
	isClose bool
}

// NewBulkhead creates a new Bulkhead that permits at most limit concurrent executions.
// If limit <= 0, it defaults to 10.
func NewBulkhead(limit int) *Bulkhead {
	if limit <= 0 {
		limit = 10
	}
	return &Bulkhead{
		limit:  limit,
		sem:    make(chan struct{}, limit),
		closed: make(chan struct{}),
	}
}

// Limit returns the maximum concurrency limit of the bulkhead.
func (b *Bulkhead) Limit() int {
	return b.limit
}

// Running returns the number of active operations currently in flight.
func (b *Bulkhead) Running() int {
	return len(b.sem)
}

// Available returns the number of remaining execution slots.
func (b *Bulkhead) Available() int {
	return b.limit - len(b.sem)
}

// TryAcquire attempts to acquire an execution slot without blocking.
// Returns a release function and true if acquired, or nil and false if the bulkhead is full or closed.
func (b *Bulkhead) TryAcquire() (func(), bool) {
	b.closeMu.RLock()
	if b.isClose {
		b.closeMu.RUnlock()
		return nil, false
	}
	b.closeMu.RUnlock()

	select {
	case b.sem <- struct{}{}:
		var once sync.Once
		release := func() {
			once.Do(func() {
				<-b.sem
			})
		}
		return release, true
	default:
		return nil, false
	}
}

// Acquire waits for an execution slot up to the context deadline.
// Returns a release function to be called when the task completes.
func (b *Bulkhead) Acquire(ctx context.Context) (func(), error) {
	b.closeMu.RLock()
	if b.isClose {
		b.closeMu.RUnlock()
		return nil, ErrBulkheadClosed
	}
	b.closeMu.RUnlock()

	select {
	case b.sem <- struct{}{}:
		var once sync.Once
		release := func() {
			once.Do(func() {
				<-b.sem
			})
		}
		return release, nil
	case <-b.closed:
		return nil, ErrBulkheadClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Execute runs fn within the bulkhead boundary, blocking until a slot is available
// or ctx is canceled.
func (b *Bulkhead) Execute(ctx context.Context, fn func(context.Context) error) error {
	release, err := b.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return fn(ctx)
}

// Close gracefully terminates the bulkhead, refusing any new acquisitions.
func (b *Bulkhead) Close() {
	b.closeMu.Lock()
	defer b.closeMu.Unlock()
	if !b.isClose {
		b.isClose = true
		close(b.closed)
	}
}
