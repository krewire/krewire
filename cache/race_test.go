package cache_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krewire/krewire/cache"
)

// TestCache_ConcurrentRace verifies thread-safety and race-free concurrent execution
// of cache operations across goroutines under go test -race.
func TestCache_ConcurrentRace(t *testing.T) {
	c := cache.NewMemoryCache(10 * time.Millisecond)
	defer c.Close()
	ctx := context.Background()

	const workers = 20
	const iterations = 50
	var wg sync.WaitGroup

	wg.Add(workers)
	for w := 0; w < workers; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key_%d", (workerID+i)%10)
				val := fmt.Sprintf("val_%d_%d", workerID, i)

				// 1. Write
				if err := c.Set(ctx, key, []byte(val), 100*time.Millisecond); err != nil {
					t.Errorf("Set failed: %v", err)
				}

				// 2. Read
				_, _ = c.Get(ctx, key)

				// 3. Delete
				if i%5 == 0 {
					_ = c.Delete(ctx, key)
				}
			}
		}()
	}

	wg.Wait()
}
