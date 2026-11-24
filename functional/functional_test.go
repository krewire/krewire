package functional

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoizeCachesSuccessfulResults(t *testing.T) {
	var calls atomic.Int32
	cached := Memoize(func(key string) (string, error) { calls.Add(1); return key + "!", nil })
	for i := 0; i < 2; i++ {
		got, err := cached("go")
		if err != nil || got != "go!" {
			t.Fatalf("got %q, %v", got, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestMemoizeDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	cached := Memoize(func(int) (string, error) { calls.Add(1); return "", errors.New("fail") })
	_, _ = cached(1)
	_, _ = cached(1)
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestAdvancedUtilities(t *testing.T) {
	var calls atomic.Int32
	cached := MemoizeWithTTL(time.Minute, func(key int) (int, error) { calls.Add(1); return key * 2, nil })
	if value, _ := cached(2); value != 4 {
		t.Fatal("MemoizeWithTTL returned wrong value")
	}
	if value, _ := cached(2); value != 4 || calls.Load() != 1 {
		t.Fatal("MemoizeWithTTL did not cache")
	}
	if got := Map([]int{1, 2, 3}, func(v int) string { return fmt.Sprint(v) }); len(got) != 3 {
		t.Fatal("Map returned wrong length")
	}
	if got := Filter([]int{1, 2, 3}, func(v int) bool { return v%2 == 1 }); len(got) != 2 {
		t.Fatal("Filter returned wrong length")
	}
	if got := Reduce([]int{1, 2, 3}, 0, func(sum, v int) int { return sum + v }); got != 6 {
		t.Fatal("Reduce returned wrong result")
	}
	if got := Pipe(func(v int) int { return v + 1 }, func(v int) int { return v * 2 })(3); got != 8 {
		t.Fatal("Pipe returned wrong result")
	}
}

func TestBindComposeAndCounter(t *testing.T) {
	add := Bind(func(a, b int) int { return a + b }, 5)
	if add(3) != 8 {
		t.Fatal("Bind returned wrong result")
	}
	pipeline := Compose(func(v int) int { return v * 2 }, func(v int) string { return string(rune('0' + v)) })
	if pipeline(2) != "4" {
		t.Fatal("Compose returned wrong result")
	}
	next := Counter(0)
	if next() != 1 || next() != 2 {
		t.Fatal("Counter did not retain state")
	}
}
