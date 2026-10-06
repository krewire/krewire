package lifecycle

import (
	"context"
	"sync"

	"github.com/krewire/krewire/packages/kern/errs"
	"github.com/krewire/krewire/packages/kern/model"
)

// Executor dispatches a workload.
type Executor interface {
	Execute(ctx context.Context, workload model.Workload[string]) errs.ExitCode
}

// executor dispatches to the module that handles the workload's Kind.
type executor struct {
	mu       sync.RWMutex
	handlers map[model.Kind]func(context.Context, model.Workload[string]) errs.ExitCode
}

func newExecutor() *executor {
	return &executor{handlers: make(map[model.Kind]func(context.Context, model.Workload[string]) errs.ExitCode)}
}

func (e *executor) Register(kind model.Kind, fn func(context.Context, model.Workload[string]) errs.ExitCode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[kind] = fn
}

func (e *executor) Execute(ctx context.Context, workload model.Workload[string]) errs.ExitCode {
	e.mu.RLock()
	fn, ok := e.handlers[workload.Kind]
	e.mu.RUnlock()
	if ok {
		return fn(ctx, workload)
	}
	return errs.ExitCodeUsage
}
