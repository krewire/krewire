// Package lifecycle provides the imperative control plane that boots,
// supervises, and executes workloads: Kernel, Registry, Executor, and
// Supervisor. It is the mechanics half of the Krewire Kernel, paired with the
// declarative rules in kern/model.
package lifecycle

import (
	"context"

	"github.com/krewire/krewire/packages/kern/errs"
	"github.com/krewire/krewire/packages/kern/model"
)

// Kernel is the central executor for the Krewire ecosystem.
type Kernel struct {
	Project    model.Project
	Registry   *Registry
	Supervisor *Supervisor
	Executor   Executor
	exec       *executor
}

// New creates a Kernel for the given project, validating it.
func New(project model.Project) (*Kernel, error) {
	if err := project.Validate(); err != nil {
		return nil, err
	}
	reg := NewRegistry()
	sup := NewSupervisor()
	exec := newExecutor()
	return &Kernel{
		Project:    project,
		Registry:   reg,
		Supervisor: sup,
		Executor:   exec,
		exec:       exec,
	}, nil
}

// Use registers modules and returns the receiver for chaining.
func (k *Kernel) Use(modules ...Module) *Kernel {
	for _, m := range modules {
		_ = k.Registry.Register(m) // duplicate will be surfaced in Boot
		// If module is startable, add to supervisor
		if s, ok := m.(Startable); ok {
			k.Supervisor.Add(s)
		}
	}
	return k
}

// RegisterHandler registers a workload handler for the given kind.
func (k *Kernel) RegisterHandler(kind model.Kind, fn func(context.Context, model.Workload[string]) errs.ExitCode) {
	k.exec.Register(kind, fn)
}

// Boot initializes all registered modules in dependency order.
func (k *Kernel) Boot(ctx context.Context) error {
	for _, m := range k.Registry.Ordered() {
		if err := m.Init(k); err != nil {
			if _, ok := err.(*errs.Error); ok {
				return err
			}
			return errs.FailureError(err.Error())
		}
	}
	return nil
}

// Execute dispatches the workload via the registered handler.
func (k *Kernel) Execute(ctx context.Context, workload model.Workload[string]) errs.ExitCode {
	return k.Executor.Execute(ctx, workload)
}

// Shutdown gracefully stops supervised modules.
func (k *Kernel) Shutdown(ctx context.Context) error {
	return k.Supervisor.Stop(ctx)
}
