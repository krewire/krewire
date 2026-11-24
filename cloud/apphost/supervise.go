package apphost

import (
	"context"
	"errors"
	"time"
)

// maxBackoff caps the exponential restart backoff.
const maxBackoff = 30 * time.Second

// startUnit launches u and waits for readiness. It returns the failing stage.
func (o *Orchestrator) startUnit(ctx context.Context, u *unit, stop <-chan struct{}) (string, error) {
	o.setPhase(u, PhaseStarting, "")
	if u.res.External != nil {
		if err := waitReady(ctx, u.res, u.env, nil, stop, o.probeFailed(u)); err != nil {
			return "readiness", err
		}
		o.markReady(u)
		return "", nil
	}
	p, err := o.launch(ctx, u)
	if err != nil {
		return "launch", err
	}
	if err := waitReady(ctx, u.res, u.env, p.Done(), stop, o.probeFailed(u)); err != nil {
		if errors.Is(err, errExitedBeforeReady) && p.Err() != nil {
			err = errors.Join(err, p.Err())
		}
		return "readiness", err
	}
	o.markReady(u)
	o.mu.Lock()
	u.supDone = make(chan struct{})
	o.mu.Unlock()
	go o.supervise(u, p)
	return "", nil
}

func (o *Orchestrator) launch(ctx context.Context, u *unit) (Process, error) {
	d := o.opts.Drivers[u.res.Runner()]
	p, err := d.Start(ctx, LaunchSpec{Resource: u.res, Env: u.env, Stdout: u.out, Stderr: u.err})
	if err != nil {
		return nil, err
	}
	o.mu.Lock()
	u.proc = p
	u.status.PID = p.PID()
	u.status.StartedAt = time.Now()
	o.mu.Unlock()
	o.logs.system(u.res.Name, "launched pid %d", p.PID())
	return p, nil
}

func (o *Orchestrator) markReady(u *unit) {
	o.mu.Lock()
	u.status.ReadyAt = time.Now()
	u.status.LastProbe = ""
	o.mu.Unlock()
	o.setPhase(u, PhaseReady, "")
}

func (o *Orchestrator) probeFailed(u *unit) func(error) {
	return func(err error) {
		o.mu.Lock()
		u.status.LastProbe = err.Error()
		o.mu.Unlock()
		o.emit(Event{Type: EventProbeFailed, Resource: u.res.Name, Message: err.Error()})
	}
}

// supervise applies the restart policy after p became ready. It returns when
// the unit is halted or the policy gives up.
func (o *Orchestrator) supervise(u *unit, p Process) {
	defer close(u.supDone)
	for {
		select {
		case <-p.Done():
		case <-u.halt:
			return
		}
		cause := p.Err()
		for {
			if o.halted(u) {
				return
			}
			u.out.flush()
			u.err.flush()
			if !o.shouldRestart(u, cause) {
				if cause != nil {
					o.setPhase(u, PhaseFailed, cause.Error())
				} else {
					o.setPhase(u, PhaseExited, "exit 0")
				}
				return
			}
			o.mu.Lock()
			n := u.status.Restarts
			u.status.Restarts++
			o.mu.Unlock()
			msg := "exit 0"
			if cause != nil {
				msg = cause.Error()
			}
			o.setPhase(u, PhaseRestarting, msg)
			wait := u.res.Restart.Backoff.D() << n
			if wait > maxBackoff || wait <= 0 {
				wait = maxBackoff
			}
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-u.halt:
				timer.Stop()
				return
			}
			o.setPhase(u, PhaseStarting, "restart")
			np, err := o.launch(context.Background(), u)
			if err != nil {
				cause = err
				continue
			}
			if err := waitReady(context.Background(), u.res, u.env, np.Done(), u.halt, o.probeFailed(u)); err != nil {
				if errors.Is(err, errStopping) {
					return
				}
				_ = np.Stop(u.res.ShutdownTimeout.D())
				cause = err
				continue
			}
			o.markReady(u)
			p = np
			break
		}
	}
}

func (o *Orchestrator) halted(u *unit) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return u.halted
}

func (o *Orchestrator) shouldRestart(u *unit, cause error) bool {
	o.mu.Lock()
	n := u.status.Restarts
	o.mu.Unlock()
	if n >= u.res.Restart.MaxRestarts {
		return false
	}
	switch u.res.Restart.Policy {
	case RestartAlways:
		return true
	case RestartOnFailure:
		return cause != nil
	}
	return false
}

// stopUnit halts supervision, then stops the current process within the
// shutdown timeout (capped by ctx's deadline). final is the phase recorded
// afterwards.
func (o *Orchestrator) stopUnit(ctx context.Context, u *unit, final Phase) error {
	o.mu.Lock()
	if !u.halted {
		u.halted = true
		close(u.halt)
	}
	sup := u.supDone
	o.mu.Unlock()
	if sup != nil {
		<-sup
	}
	o.mu.Lock()
	p := u.proc
	u.proc = nil
	o.mu.Unlock()
	if p == nil {
		if final != PhaseFailed {
			o.setPhase(u, final, "")
		}
		return nil
	}
	grace := u.res.ShutdownTimeout.D()
	if dl, ok := ctx.Deadline(); ok {
		if left := time.Until(dl); left < grace {
			grace = max(left, 0)
		}
	}
	if final != PhaseFailed {
		o.setPhase(u, PhaseStopping, "")
	}
	err := p.Stop(grace)
	u.out.flush()
	u.err.flush()
	o.mu.Lock()
	u.status.PID = 0
	if err != nil {
		u.status.LastError = err.Error()
	}
	o.mu.Unlock()
	rec := OperationRecord{Action: ActionStop, Resource: u.res.Name, Outcome: OutcomeOK}
	if err != nil {
		rec.Outcome, rec.Error = OutcomeFailed, err.Error()
	}
	o.record(rec)
	if final != PhaseFailed {
		o.setPhase(u, final, "")
	}
	return err
}
