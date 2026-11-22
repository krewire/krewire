package lifecycle

import (
	"context"
	"sync"

	"github.com/krewire/krewire/pkg/errs"
)

// Handler is an executable project function.
type Handler func(context.Context) errs.ExitCode

// Executor dispatches project execution.
type Executor interface {
	Execute(ctx context.Context) errs.ExitCode
}

type executor struct {
	mu      sync.RWMutex
	handler Handler
}

func newExecutor() *executor {
	return &executor{}
}

func (e *executor) Register(fn Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handler = fn
}

func (e *executor) Execute(ctx context.Context) errs.ExitCode {
	e.mu.RLock()
	fn := e.handler
	e.mu.RUnlock()
	if fn != nil {
		return fn(ctx)
	}
	return errs.ExitCodeSuccess
}
