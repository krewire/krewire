package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryStopsAfterSuccess(t *testing.T) {
	attempts := 0
	err := Retry(context.Background(), RetryOptions{MaxAttempts: 3}, func(context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary")
		}
		return nil
	})
	if err != nil || attempts != 2 {
		t.Fatalf("err=%v attempts=%d", err, attempts)
	}
}

func TestRetryHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Retry(ctx, RetryOptions{MaxAttempts: 3}, func(context.Context) error { return errors.New("fail") }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

// TestNilOperations verifies both policies reject a nil operation up front
// rather than panicking on the first call.
func TestNilOperations(t *testing.T) {
	if !errors.Is(Retry(context.Background(), RetryOptions{}, nil), ErrNilOperation) {
		t.Error("Retry did not reject nil operation")
	}
	if !errors.Is(NewCircuitBreaker(1, time.Second).Do(context.Background(), nil), ErrNilOperation) {
		t.Error("breaker did not reject nil operation")
	}
}

// TestRESILIENCE_001_RetryDefaultsToOneAttempt verifies a zero MaxAttempts still
// calls the operation exactly once, so the zero RetryOptions is a usable policy
// rather than a silent no-op.
func TestRESILIENCE_001_RetryDefaultsToOneAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), RetryOptions{}, func(context.Context) error {
		calls++
		return errors.New("boom")
	})
	if calls != 1 {
		t.Errorf("calls = %d, want 1 for the zero options", calls)
	}
	if err == nil {
		t.Error("err = nil, want the operation error")
	}
}

// TestRESILIENCE_002_RetryToleratesNilContext verifies a nil context is treated
// as context.Background rather than panicking on the first ctx.Err() call.
func TestRESILIENCE_002_RetryToleratesNilContext(t *testing.T) {
	//lint:ignore SA1012 a nil context is exactly the input under test.
	err := Retry(nil, RetryOptions{MaxAttempts: 1}, func(context.Context) error { return nil })
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// TestRESILIENCE_003_ShouldRetryShortCircuits verifies ShouldRetry can stop the
// loop early on a permanent error, and that the error is returned unwrapped.
func TestRESILIENCE_003_ShouldRetryShortCircuits(t *testing.T) {
	permanent := errors.New("unauthorized")
	calls := 0
	err := Retry(context.Background(), RetryOptions{
		MaxAttempts: 5,
		ShouldRetry: func(e error) bool { return !errors.Is(e, permanent) },
	}, func(context.Context) error {
		calls++
		return permanent
	})
	if calls != 1 {
		t.Errorf("calls = %d, want 1: ShouldRetry must stop the loop", calls)
	}
	if !errors.Is(err, permanent) {
		t.Errorf("err = %v, want %v", err, permanent)
	}
}

// TestRESILIENCE_004_ShouldRetryAllowsRecovery verifies ShouldRetry returning true
// does not prevent a later attempt from succeeding.
func TestRESILIENCE_004_ShouldRetryAllowsRecovery(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), RetryOptions{
		MaxAttempts: 3,
		ShouldRetry: func(error) bool { return true },
	}, func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("err = %v, calls = %d; want nil after 3 calls", err, calls)
	}
}

// TestRESILIENCE_005_DelayIsExponential covers the backoff schedule, which was
// the least-exercised logic in the package: an unset multiplier must fall back to
// the documented doubling default rather than to 1 or to no growth.
func TestRESILIENCE_005_DelayIsExponential(t *testing.T) {
	cases := []struct {
		name    string
		opts    RetryOptions
		attempt int
		want    time.Duration
	}{
		{"no delay configured", RetryOptions{}, 3, 0},
		{"negative delay is treated as none", RetryOptions{InitialDelay: -time.Second}, 2, 0},
		{"first retry uses the initial delay", RetryOptions{InitialDelay: 10 * time.Millisecond}, 1, 10 * time.Millisecond},
		{"unset multiplier doubles", RetryOptions{InitialDelay: 10 * time.Millisecond}, 3, 40 * time.Millisecond},
		{"multiplier below one doubles", RetryOptions{InitialDelay: time.Second, Multiplier: 0.5}, 2, 2 * time.Second},
		{"explicit multiplier is honoured", RetryOptions{InitialDelay: time.Second, Multiplier: 3}, 3, 9 * time.Second},
		{"max delay caps growth", RetryOptions{InitialDelay: time.Second, Multiplier: 10, MaxDelay: 5 * time.Second}, 3, 5 * time.Second},
		{"max delay above the schedule is inert", RetryOptions{InitialDelay: time.Second, MaxDelay: time.Minute}, 2, 2 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.opts.delay(c.attempt); got != c.want {
				t.Errorf("delay(%d) = %v, want %v", c.attempt, got, c.want)
			}
		})
	}
}

// TestRESILIENCE_006_DelayJitterStaysInBand verifies jitter randomizes the delay
// within the requested proportional band, never goes negative, and still
// respects a MaxDelay cap that sits below the base delay.
func TestRESILIENCE_006_DelayJitterStaysInBand(t *testing.T) {
	const (
		base    = 100 * time.Millisecond
		jitter  = 0.5
		samples = 200
	)

	// MaxDelay below the base: every draw must land on the cap.
	capped := RetryOptions{InitialDelay: base, Multiplier: 1, Jitter: jitter, MaxDelay: 10 * time.Millisecond}
	for i := 0; i < samples; i++ {
		if got := capped.delay(1); got != capped.MaxDelay {
			t.Fatalf("delay = %v, want the MaxDelay cap %v", got, capped.MaxDelay)
		}
	}

	banded := RetryOptions{InitialDelay: base, Multiplier: 1, Jitter: jitter}
	low := time.Duration(float64(base) * (1 - jitter))
	high := time.Duration(float64(base) * (1 + jitter))
	distinct := make(map[time.Duration]bool)
	for i := 0; i < samples; i++ {
		got := banded.delay(1)
		if got < 0 {
			t.Fatalf("jittered delay must never be negative, got %v", got)
		}
		if got < low || got > high {
			t.Fatalf("jittered delay %v outside [%v, %v]", got, low, high)
		}
		distinct[got] = true
	}
	if len(distinct) < 2 {
		t.Errorf("jitter produced %d distinct value(s); it must actually randomize", len(distinct))
	}
}

// TestRESILIENCE_007_RetryCancelsDuringBackoff verifies a context cancelled while
// the loop is about to wait returns the context error instead of burning the
// remaining attempts.
func TestRESILIENCE_007_RetryCancelsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Retry(ctx, RetryOptions{MaxAttempts: 10, InitialDelay: 50 * time.Millisecond}, func(context.Context) error {
		calls++
		cancel() // cancel while the loop is about to wait
		return errors.New("transient")
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1: cancellation must end the loop", calls)
	}
}
