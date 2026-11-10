package resilience_test

import (
	"context"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/resilience"
)

func TestRateLimiter_BurstAndThrottle(t *testing.T) {
	// 10 tokens per second, burst 3
	rl := resilience.NewRateLimiter(10.0, 3)

	// Burst of 3 should all succeed
	if !rl.Allow() {
		t.Error("expected token 1 to be allowed")
	}
	if !rl.Allow() {
		t.Error("expected token 2 to be allowed")
	}
	if !rl.Allow() {
		t.Error("expected token 3 to be allowed")
	}

	// 4th token should be refused immediately
	if rl.Allow() {
		t.Error("expected token 4 to be rejected due to burst exhaustion")
	}

	// Wait 120ms (enough for at least 1 new token at 10 tokens/sec)
	time.Sleep(120 * time.Millisecond)

	if !rl.Allow() {
		t.Error("expected token to be refilled after sleep")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	// 50 tokens per sec, burst 1
	rl := resilience.NewRateLimiter(50.0, 1)

	// Consume initial token
	if !rl.Allow() {
		t.Fatal("expected initial token to be allowed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("expected Wait to succeed, got %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 15*time.Millisecond {
		t.Errorf("Wait should have waited for token generation, elapsed: %v", elapsed)
	}
}

func TestRateLimiter_WaitContextCanceled(t *testing.T) {
	rl := resilience.NewRateLimiter(1.0, 1)
	_ = rl.Allow() // exhaust

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Fatal("expected Wait to error on timeout")
	}
}
