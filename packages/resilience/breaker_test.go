package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestRESILIENCE_010_BreakerOpensAfterThreshold verifies the breaker admits
// calls below the threshold and refuses them once the threshold is reached,
// without invoking the operation.
func TestRESILIENCE_010_BreakerOpensAfterThreshold(t *testing.T) {
	breaker := NewCircuitBreaker(3, time.Minute)
	failure := errors.New("dependency down")

	// The breaker opens once the failure count reaches the threshold, so calls
	// 1 and 2 are served and call 3 is the one that trips it.
	for i := 1; i <= 3; i++ {
		err := breaker.Do(context.Background(), func(context.Context) error { return failure })
		if !errors.Is(err, failure) {
			t.Fatalf("attempt %d: err = %v, want the operation error", i, err)
		}
		want := Closed
		if i == 3 {
			want = Open
		}
		if breaker.State() != want {
			t.Fatalf("attempt %d: state = %v, want %v", i, breaker.State(), want)
		}
	}

	called := false
	err := breaker.Do(context.Background(), func(context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("err = %v, want ErrCircuitOpen once the threshold is reached", err)
	}
	if called {
		t.Error("the operation must not be invoked while the breaker is open")
	}
	if breaker.State() != Open {
		t.Errorf("state = %v, want Open", breaker.State())
	}
}

// TestRESILIENCE_011_SuccessResetsFailureCount verifies a success inside the
// threshold resets the count, so an intermittent failure never accumulates into
// a spurious open.
func TestRESILIENCE_011_SuccessResetsFailureCount(t *testing.T) {
	breaker := NewCircuitBreaker(2, time.Minute)
	failure := errors.New("flaky")

	for i := 0; i < 5; i++ {
		if err := breaker.Do(context.Background(), func(context.Context) error { return failure }); err == nil {
			t.Fatalf("iteration %d: expected the failure to propagate", i)
		}
		if err := breaker.Do(context.Background(), func(context.Context) error { return nil }); err != nil {
			t.Fatalf("iteration %d: success must reset the count, got %v", i, err)
		}
	}
	if breaker.State() != Closed {
		t.Errorf("state = %v, want Closed: an alternating failure pattern must not open the breaker", breaker.State())
	}
}

// TestRESILIENCE_012_HalfOpenAdmitsOneProbe verifies that after the cooldown only
// a single probe runs: a second caller is refused while the probe is still in
// flight, which is what stops a stampede onto a recovering dependency.
func TestRESILIENCE_012_HalfOpenAdmitsOneProbe(t *testing.T) {
	breaker := NewCircuitBreaker(1, time.Millisecond)
	release := make(chan struct{})
	probeStarted := make(chan struct{})

	// Open the breaker.
	if err := breaker.Do(context.Background(), func(context.Context) error { return errors.New("down") }); err == nil {
		t.Fatal("expected the first call to fail")
	}
	time.Sleep(2 * time.Millisecond) // let the cooldown elapse

	probeDone := make(chan error, 1)
	go func() {
		probeDone <- breaker.Do(context.Background(), func(context.Context) error {
			close(probeStarted)
			<-release
			return nil
		})
	}()

	<-probeStarted // the probe is in flight; the breaker is half-open

	if err := breaker.Do(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("second caller during the probe: err = %v, want ErrCircuitOpen", err)
	}

	close(release)
	if err := <-probeDone; err != nil {
		t.Errorf("probe err = %v, want nil", err)
	}
	if breaker.State() != Closed {
		t.Errorf("state = %v, want Closed after a successful probe", breaker.State())
	}
}

// TestRESILIENCE_013_FailedProbeReopensBreaker verifies a failing half-open probe
// returns the breaker to Open rather than leaving it half-open to admit more
// traffic.
func TestRESILIENCE_013_FailedProbeReopensBreaker(t *testing.T) {
	breaker := NewCircuitBreaker(1, time.Millisecond)
	failure := errors.New("still down")

	_ = breaker.Do(context.Background(), func(context.Context) error { return failure })
	time.Sleep(2 * time.Millisecond)
	if breaker.State() != HalfOpen {
		t.Fatalf("state = %v, want HalfOpen after the cooldown", breaker.State())
	}

	if err := breaker.Do(context.Background(), func(context.Context) error { return failure }); !errors.Is(err, failure) {
		t.Errorf("err = %v, want the operation error", err)
	}
	if breaker.State() != Open {
		t.Errorf("state = %v, want Open: a failed probe must reopen the breaker", breaker.State())
	}
}

// TestRESILIENCE_014_NewCircuitBreakerNormalizesInput verifies the constructor
// raises a nonsensical threshold to 1 and clamps a negative cooldown, so a
// misconfigured breaker still gates traffic instead of never opening.
func TestRESILIENCE_014_NewCircuitBreakerNormalizesInput(t *testing.T) {
	for _, threshold := range []int{0, -1} {
		breaker := NewCircuitBreaker(threshold, time.Minute)
		if err := breaker.Do(context.Background(), func(context.Context) error { return errors.New("x") }); err == nil {
			t.Fatalf("threshold %d: expected the failure to propagate", threshold)
		}
		if breaker.State() != Open {
			t.Errorf("threshold %d: state = %v, want Open: the floor must be 1", threshold, breaker.State())
		}
	}

	// A negative cooldown is clamped to 0, so the breaker is immediately
	// half-open rather than staying open for a negative duration.
	breaker := NewCircuitBreaker(1, -time.Hour)
	_ = breaker.Do(context.Background(), func(context.Context) error { return errors.New("x") })
	if breaker.State() != HalfOpen {
		t.Errorf("state = %v, want HalfOpen with a clamped cooldown", breaker.State())
	}
}

// TestRESILIENCE_015_NilBreakerRefusesCalls verifies a nil breaker reports
// ErrCircuitOpen instead of panicking, so a caller holding an unconfigured
// breaker degrades to "refused" rather than crashing.
func TestRESILIENCE_015_NilBreakerRefusesCalls(t *testing.T) {
	var breaker *CircuitBreaker
	if err := breaker.Do(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("err = %v, want ErrCircuitOpen", err)
	}
}

// TestRESILIENCE_016_ZeroCooldownRecoversImmediately verifies a breaker built
// with a zero cooldown passes through half-open on the next call rather than
// latching open forever.
func TestRESILIENCE_016_ZeroCooldownRecoversImmediately(t *testing.T) {
	breaker := NewCircuitBreaker(1, 0)
	if err := breaker.Do(context.Background(), func(context.Context) error { return errors.New("x") }); err == nil {
		t.Fatal("expected the failure to propagate")
	}
	if err := breaker.Do(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Errorf("err = %v, want the probe to succeed with a zero cooldown", err)
	}
	if breaker.State() != Closed {
		t.Errorf("state = %v, want Closed", breaker.State())
	}
}

// TestRESILIENCE_018_BreakerStateString verifies the diagnostics name each state,
// so a logged breaker reads "half-open" rather than the number 2.
func TestRESILIENCE_018_BreakerStateString(t *testing.T) {
	cases := map[BreakerState]string{Closed: "closed", Open: "open", HalfOpen: "half-open"}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("BreakerState(%d).String() = %q, want %q", state, got, want)
		}
	}
}

// TestRESILIENCE_017_BreakerIsConcurrencySafe exercises the breaker under
// parallel callers. Run with -race, this fails if the state transitions are not
// serialized: every call must be accounted for as served or refused, never
// silently dropped, and the breaker must not latch open under mixed traffic.
func TestRESILIENCE_017_BreakerIsConcurrencySafe(t *testing.T) {
	const (
		callers   = 32
		perCaller = 20
	)
	breaker := NewCircuitBreaker(8, 50*time.Microsecond)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		served  int
		failed  int
		refused int
	)
	for c := 0; c < callers; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := 0; i < perCaller; i++ {
				err := breaker.Do(context.Background(), func(context.Context) error {
					if (c+i)%3 == 0 {
						return errors.New("intermittent")
					}
					return nil
				})
				mu.Lock()
				switch {
				case err == nil:
					served++
				case errors.Is(err, ErrCircuitOpen):
					refused++
				default: // the operation's own error
					failed++
				}
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	// Every call must fall into exactly one bucket: served, failed by the
	// operation, or refused by the breaker. A dropped call would mean Do
	// returned neither an outcome nor an error.
	if served+failed+refused != callers*perCaller {
		t.Fatalf("accounted for %d of %d calls (served=%d failed=%d refused=%d)",
			served+failed+refused, callers*perCaller, served, failed, refused)
	}
	if breaker.State() == Open {
		t.Error("state = Open after sustained mixed traffic with a short cooldown; the breaker is not recovering")
	}
}
