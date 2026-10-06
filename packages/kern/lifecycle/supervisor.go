package lifecycle

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

// DefaultShutdownTimeout is the default context timeout used by Stop.
const DefaultShutdownTimeout = 10 * time.Second

// Startable is implemented by long-running modules.
type Startable interface {
	Start(context.Context) error
	Stop(context.Context) error
}

// Supervisor manages long-running modules.
type Supervisor struct {
	mu              sync.Mutex
	modules         []Startable
	health          map[string]error
	reloadFn        func()
	shutdownTimeout time.Duration
}

// NewSupervisor creates a Supervisor with a 10s shutdown timeout.
func NewSupervisor() *Supervisor {
	return &Supervisor{
		health:          make(map[string]error),
		shutdownTimeout: DefaultShutdownTimeout,
	}
}

// WithShutdownTimeout sets the timeout used by Stop.
func (s *Supervisor) WithShutdownTimeout(d time.Duration) *Supervisor {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownTimeout = d
	return s
}

// OnReload sets the hot-reload hook.
func (s *Supervisor) OnReload(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadFn = fn
}

// Reload triggers the hot-reload hook if set.
func (s *Supervisor) Reload() {
	s.mu.Lock()
	fn := s.reloadFn
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Add registers a startable module for supervision.
func (s *Supervisor) Add(m Startable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modules = append(s.modules, m)
}

// Start starts all registered modules.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	mods := append([]Startable(nil), s.modules...)
	s.mu.Unlock()
	for i, m := range mods {
		if err := m.Start(ctx); err != nil {
			s.mu.Lock()
			s.health[indexName(i)] = err
			s.mu.Unlock()
			return err
		}
		s.mu.Lock()
		s.health[indexName(i)] = nil
		s.mu.Unlock()
	}
	return nil
}

// Stop stops all modules with the configured timeout.
// It attempts to stop all modules in reverse order and combines any errors.
func (s *Supervisor) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()
	s.mu.Lock()
	mods := append([]Startable(nil), s.modules...)
	s.mu.Unlock()

	var stopErr error
	// Stop in reverse order
	for i := len(mods) - 1; i >= 0; i-- {
		if err := mods[i].Stop(ctx); err != nil {
			s.mu.Lock()
			s.health[indexName(i)] = err
			s.mu.Unlock()
			stopErr = errors.Join(stopErr, err)
		} else {
			s.mu.Lock()
			s.health[indexName(i)] = nil
			s.mu.Unlock()
		}
	}
	return stopErr
}

// Health returns a snapshot of module health.
func (s *Supervisor) Health() map[string]error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]error, len(s.health))
	for k, v := range s.health {
		out[k] = v
	}
	return out
}

func indexName(i int) string {
	return strconv.Itoa(i)
}
