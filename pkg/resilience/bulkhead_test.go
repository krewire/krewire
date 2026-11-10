package resilience_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/resilience"
)

func TestBulkhead_ConcurrencyLimit(t *testing.T) {
	limit := 3
	bh := resilience.NewBulkhead(limit)
	if bh.Limit() != limit {
		t.Errorf("Limit = %d, want %d", bh.Limit(), limit)
	}

	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	numGoroutines := 20
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bh.Execute(context.Background(), func(ctx context.Context) error {
				curr := atomic.AddInt32(&active, 1)
				for {
					old := atomic.LoadInt32(&maxActive)
					if curr <= old || atomic.CompareAndSwapInt32(&maxActive, old, curr) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				return nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	if maxActive > int32(limit) {
		t.Errorf("max active goroutines %d exceeded bulkhead limit %d", maxActive, limit)
	}
	if bh.Running() != 0 {
		t.Errorf("expected 0 running tasks, got %d", bh.Running())
	}
}

func TestBulkhead_TryAcquire(t *testing.T) {
	bh := resilience.NewBulkhead(1)

	rel1, ok := bh.TryAcquire()
	if !ok || rel1 == nil {
		t.Fatal("expected first TryAcquire to succeed")
	}

	// Second acquisition should fail without blocking
	rel2, ok := bh.TryAcquire()
	if ok || rel2 != nil {
		t.Fatal("expected second TryAcquire to fail")
	}

	rel1()

	// Third acquisition should succeed after release
	rel3, ok := bh.TryAcquire()
	if !ok || rel3 == nil {
		t.Fatal("expected third TryAcquire to succeed after release")
	}
	rel3()
}

func TestBulkhead_ContextCancellation(t *testing.T) {
	bh := resilience.NewBulkhead(1)

	rel, ok := bh.TryAcquire()
	if !ok {
		t.Fatal("failed to acquire initial slot")
	}
	defer rel()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := bh.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestBulkhead_Close(t *testing.T) {
	bh := resilience.NewBulkhead(2)
	bh.Close()

	_, ok := bh.TryAcquire()
	if ok {
		t.Error("TryAcquire on closed bulkhead should fail")
	}

	_, err := bh.Acquire(context.Background())
	if !errors.Is(err, resilience.ErrBulkheadClosed) {
		t.Errorf("expected ErrBulkheadClosed, got %v", err)
	}
}
