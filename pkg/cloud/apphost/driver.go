package apphost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// Runner names used as keys of Options.Drivers.
const (
	RunnerProcess   = "process"
	RunnerContainer = "container"
	RunnerExternal  = "external"
)

// ErrShutdownTimeout reports that a resource ignored its graceful stop signal
// for the whole shutdown timeout and was killed.
var ErrShutdownTimeout = errors.New("apphost: shutdown timeout exceeded; process group killed")

// LaunchSpec is everything a Driver needs to start one resource. Endpoint
// ports are already allocated and Env is the complete environment.
type LaunchSpec struct {
	Resource *Resource
	Env      []string
	Stdout   io.Writer
	Stderr   io.Writer
}

// Process is a running resource instance owned by the orchestrator.
type Process interface {
	// PID is the operating-system process id (0 when not applicable).
	PID() int
	// Done is closed once the process has exited and been reaped.
	Done() <-chan struct{}
	// Err is the exit error, valid after Done is closed (nil for exit 0).
	Err() error
	// Stop asks the process (and its children) to exit, waits up to grace,
	// then kills them. It returns ErrShutdownTimeout when it had to kill, and
	// always returns only after the process has been reaped.
	Stop(grace time.Duration) error
}

// Driver launches resources of one runner type. Drivers are the extension
// point for system services or other adapters; the orchestrator never
// hard-codes a resource kind.
type Driver interface {
	Start(ctx context.Context, spec LaunchSpec) (Process, error)
}

// ProcessDriver runs RunSpec resources as local child processes in their own
// process group, so stopping a resource also stops anything it spawned.
type ProcessDriver struct {
	// WaitDelay bounds how long output copying may outlive the process
	// (default 2s); see exec.Cmd.WaitDelay.
	WaitDelay time.Duration
}

// Start implements Driver.
func (d ProcessDriver) Start(_ context.Context, spec LaunchSpec) (Process, error) {
	run := spec.Resource.Run
	if run == nil {
		return nil, fmt.Errorf("apphost: resource %q has no run block", spec.Resource.Name)
	}
	cmd := exec.Command(run.Command, run.Args...)
	cmd.Dir = run.Dir
	return startCmd(cmd, spec, d.WaitDelay)
}

func startCmd(cmd *exec.Cmd, spec LaunchSpec, waitDelay time.Duration) (*osProcess, error) {
	if waitDelay <= 0 {
		waitDelay = 2 * time.Second
	}
	cmd.Env = spec.Env
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	cmd.WaitDelay = waitDelay
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("apphost: launch %q: %w", spec.Resource.Name, err)
	}
	p := &osProcess{cmd: cmd, done: make(chan struct{})}
	go func() {
		p.err = cmd.Wait()
		// Reap anything the process left behind in its group.
		killGroup(cmd)
		close(p.done)
	}()
	return p, nil
}

// osProcess is a Process backed by exec.Cmd.
type osProcess struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
	// stop overrides the graceful signal (containers use `docker stop`).
	stop func(grace time.Duration)
}

func (p *osProcess) PID() int              { return p.cmd.Process.Pid }
func (p *osProcess) Done() <-chan struct{} { return p.done }
func (p *osProcess) Err() error            { return p.err }

func (p *osProcess) Stop(grace time.Duration) error {
	select {
	case <-p.done:
		return nil
	default:
	}
	if p.stop != nil {
		go p.stop(grace)
	} else {
		terminateGroup(p.cmd)
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-p.done:
		return nil
	case <-timer.C:
		killGroup(p.cmd)
		<-p.done
		return ErrShutdownTimeout
	}
}
