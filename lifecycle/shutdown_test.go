package lifecycle_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krewire/krewire/lifecycle"
)

type orderTrackingModule struct {
	name      string
	stopLog   *[]string
	mu        *sync.Mutex
	stopDelay time.Duration
	stopErr   error
	drainChan chan struct{}
}

func (m *orderTrackingModule) Start(ctx context.Context) error {
	return nil
}

func (m *orderTrackingModule) Stop(ctx context.Context) error {
	if m.stopDelay > 0 {
		select {
		case <-time.After(m.stopDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.drainChan != nil {
		close(m.drainChan)
	}
	m.mu.Lock()
	*m.stopLog = append(*m.stopLog, m.name)
	m.mu.Unlock()
	return m.stopErr
}

// TestSupervisor_GracefulShutdownAndLIFOOrdering verifies that long-running workloads
// drain in-flight work and shut down in reverse dependency order (LIFO).
func TestSupervisor_GracefulShutdownAndLIFOOrdering(t *testing.T) {
	s := lifecycle.NewSupervisor().WithShutdownTimeout(2 * time.Second)

	var stopLog []string
	var mu sync.Mutex

	dbDrain := make(chan struct{})
	httpDrain := make(chan struct{})

	dbModule := &orderTrackingModule{
		name:      "database_pool",
		stopLog:   &stopLog,
		mu:        &mu,
		drainChan: dbDrain,
	}

	httpModule := &orderTrackingModule{
		name:      "http_server",
		stopLog:   &stopLog,
		mu:        &mu,
		drainChan: httpDrain,
	}

	// Module registration order: database first, then HTTP server
	s.Add(dbModule)
	s.Add(httpModule)

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Trigger graceful shutdown
	shutdownCtx := context.Background()
	if err := s.Stop(shutdownCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// In LIFO order, HTTP server must stop BEFORE database pool
	mu.Lock()
	defer mu.Unlock()
	if len(stopLog) != 2 {
		t.Fatalf("expected 2 stopped modules, got %d", len(stopLog))
	}
	if stopLog[0] != "http_server" || stopLog[1] != "database_pool" {
		t.Errorf("expected LIFO shutdown order [http_server, database_pool], got: %v", stopLog)
	}
}

// TestSupervisor_ShutdownTimeoutCancellation verifies that unresponsive or hanging modules
// are cancelled when the shutdown timeout elapses.
func TestSupervisor_ShutdownTimeoutCancellation(t *testing.T) {
	// Short 50ms shutdown timeout
	s := lifecycle.NewSupervisor().WithShutdownTimeout(50 * time.Millisecond)

	var stopLog []string
	var mu sync.Mutex

	hangingModule := &orderTrackingModule{
		name:      "hanging_worker",
		stopLog:   &stopLog,
		mu:        &mu,
		stopDelay: 500 * time.Millisecond, // Exceeds 50ms timeout
	}

	s.Add(hangingModule)

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	start := time.Now()
	err := s.Stop(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("expected timeout context error during Stop, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Must finish near 50ms, not wait the full 500ms
	if elapsed > 250*time.Millisecond {
		t.Errorf("shutdown took %v, timeout was not respected", elapsed)
	}
}
