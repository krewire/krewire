package apphost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Options configures an Orchestrator. The zero value runs process resources
// with the host environment and keeps 1000 log lines per resource.
type Options struct {
	// Drivers overrides or adds launch drivers keyed by runner
	// (RunnerProcess, RunnerContainer). The process driver is always
	// available; the container driver is detected on PATH when needed.
	Drivers map[string]Driver
	// BaseEnv is the environment every resource inherits (nil = os.Environ()).
	BaseEnv []string
	// LogSink receives aggregated "resource | line" output.
	LogSink io.Writer
	// LogHistory bounds the per-resource log buffer (default 1000 lines).
	LogHistory int
	// Observer receives lifecycle events.
	Observer Observer
	// Approver gates Start; nil approves everything.
	Approver Approver
	// Recorder receives every journal entry.
	Recorder Recorder
	// TaskID ties journal entries to the shared task/evidence record.
	TaskID string
	// Actor names who requested operations (a user or an agent id).
	Actor string
}

// Orchestrator runs one topology locally. Create it with New; it is safe for
// concurrent use. Start and Stop may each be called once per instance.
type Orchestrator struct {
	opts Options
	top  *Topology // private copy; endpoint ports are allocated in place
	logs *logHub

	lifecycle sync.Mutex // serializes Start and Stop

	mu       sync.Mutex
	units    map[string]*unit
	started  bool
	running  bool
	stopping bool
	stopCh   chan struct{}
	journal  []OperationRecord
}

type unit struct {
	res     *Resource
	env     []string
	status  ResourceStatus
	proc    Process
	out     *lineWriter
	err     *lineWriter
	halt    chan struct{}
	halted  bool
	supDone chan struct{}
}

// New validates t and prepares an orchestrator for it. t itself is never
// modified.
func New(t *Topology, opts Options) (*Orchestrator, error) {
	top := t.Clone()
	top.Normalize()
	if err := top.Validate(); err != nil {
		return nil, err
	}
	if opts.BaseEnv == nil {
		opts.BaseEnv = os.Environ()
	}
	drivers := map[string]Driver{RunnerProcess: ProcessDriver{}}
	for k, d := range opts.Drivers {
		drivers[k] = d
	}
	opts.Drivers = drivers
	o := &Orchestrator{opts: opts, top: top, units: map[string]*unit{}, stopCh: make(chan struct{})}
	o.logs = newLogHub(opts.LogHistory, opts.LogSink, nil)
	for _, name := range top.Names() {
		r := top.Resources[name]
		o.units[name] = &unit{
			res:    r,
			status: ResourceStatus{Name: name, Kind: r.Kind, Runner: r.Runner(), Phase: PhasePending},
			out:    o.logs.writer(name, StreamStdout),
			err:    o.logs.writer(name, StreamStderr),
			halt:   make(chan struct{}),
		}
	}
	return o, nil
}

// Topology returns a copy of the topology with allocated endpoint ports.
func (o *Orchestrator) Topology() *Topology {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.top.Clone()
}

// Start validates preconditions, asks the Approver, allocates ports and
// starts every resource in dependency order, waiting for each one's readiness
// probe before starting anything that depends on it. On any failure it stops
// what it started in reverse order and returns a *StartError.
func (o *Orchestrator) Start(ctx context.Context) error {
	o.lifecycle.Lock()
	defer o.lifecycle.Unlock()
	o.mu.Lock()
	if o.started || o.stopping {
		o.mu.Unlock()
		return errors.New("apphost: orchestrator already started or stopped; create a new one")
	}
	o.started = true
	o.mu.Unlock()

	order := o.top.Order()
	if err := o.approve(ctx, Operation{Action: ActionStart, Topology: o.top.Name, Resources: order, Actor: o.opts.Actor}); err != nil {
		return err
	}
	for _, name := range order {
		r := o.top.Resources[name]
		if r.Runner() == RunnerExternal {
			continue
		}
		if _, ok := o.opts.Drivers[r.Runner()]; ok {
			continue
		}
		if d, ok := DetectContainerDriver(); ok && r.Runner() == RunnerContainer {
			o.opts.Drivers[RunnerContainer] = d
			continue
		}
		return &StartError{Resource: name, Stage: "preflight", Message: fmt.Sprintf("no driver for runner %q", r.Runner()),
			Hint: "install docker/podman or pass Options.Drivers[\"" + r.Runner() + "\"]"}
	}
	o.mu.Lock()
	err := allocatePorts(o.top)
	o.mu.Unlock()
	if err != nil {
		return &StartError{Stage: "preflight", Cause: err, Message: err.Error(), Hint: "fix the port conflicts; nothing was started"}
	}
	for _, name := range order {
		u := o.units[name]
		env, err := buildEnv(o.top, u.res, o.opts.BaseEnv)
		if err != nil {
			return &StartError{Resource: name, Stage: "preflight", Cause: err, Message: err.Error()}
		}
		o.mu.Lock()
		u.env = env
		u.status.Endpoints = append([]Endpoint(nil), u.res.Endpoints...)
		o.running = true
		o.mu.Unlock()
	}

	var started []string
	for _, name := range order {
		u := o.units[name]
		stage, err := o.startUnit(ctx, u, o.stopCh)
		if err == nil {
			started = append(started, name)
			o.record(OperationRecord{Action: ActionStart, Resource: name, Outcome: OutcomeOK})
			continue
		}
		o.record(OperationRecord{Action: ActionStart, Resource: name, Outcome: OutcomeFailed, Error: err.Error()})
		o.setPhase(u, PhaseFailed, err.Error())
		// Clean up the failed unit first, then everything started, newest first.
		_ = o.stopUnit(ctx, u, PhaseFailed)
		rolled := make([]string, 0, len(started))
		for i := len(started) - 1; i >= 0; i-- {
			_ = o.stopUnit(ctx, o.units[started[i]], PhaseStopped)
			rolled = append(rolled, started[i])
		}
		o.mu.Lock()
		o.running = false
		o.mu.Unlock()
		return &StartError{Resource: name, Stage: stage, Cause: err, Message: err.Error(), Hint: hintFor(stage, u.res),
			LogTail: o.logs.tail(name, 20), RolledBack: rolled}
	}
	return nil
}

func hintFor(stage string, r *Resource) string {
	switch stage {
	case "launch":
		return "check run.command/run.dir or the container image of " + r.Name
	case "readiness":
		return fmt.Sprintf("inspect `%s` logs; the probe (%s) must pass within readiness.timeout", r.Name, describeProbe(r.Readiness))
	}
	return ""
}

func (o *Orchestrator) approve(ctx context.Context, op Operation) error {
	if o.opts.Approver == nil {
		return nil
	}
	d, err := o.opts.Approver.Approve(ctx, op)
	if err == nil && !d.Approved {
		err = fmt.Errorf("operation %s denied: %s", op.Action, d.Reason)
	}
	if err != nil {
		o.record(OperationRecord{Action: op.Action, Decision: &d, Outcome: OutcomeDenied, Error: err.Error()})
		return &StartError{Stage: "approval", Cause: err, Message: err.Error(), Hint: "nothing was started; obtain approval and retry"}
	}
	o.record(OperationRecord{Action: op.Action, Decision: &d, Outcome: OutcomeApproved})
	return nil
}

// Stop stops every resource in reverse dependency order, honouring each
// resource's shutdown timeout (capped by ctx's deadline) before killing its
// process group. It interrupts an in-flight Start, is idempotent, and returns
// the joined per-resource errors (including ErrShutdownTimeout).
func (o *Orchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	if !o.stopping {
		o.stopping = true
		close(o.stopCh)
	}
	o.mu.Unlock()
	o.lifecycle.Lock()
	defer o.lifecycle.Unlock()
	o.mu.Lock()
	running := o.running
	o.running = false
	o.mu.Unlock()
	if !running {
		return nil
	}
	order := o.top.Order()
	var errs []error
	for i := len(order) - 1; i >= 0; i-- {
		if err := o.stopUnit(ctx, o.units[order[i]], PhaseStopped); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", order[i], err))
		}
	}
	return errors.Join(errs...)
}

// Run starts the topology, blocks until ctx is cancelled, then stops it with
// a fresh context so shutdown timeouts still apply.
func (o *Orchestrator) Run(ctx context.Context) error {
	if err := o.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return o.Stop(context.Background())
}

// Status returns the machine-readable state.
func (o *Orchestrator) Status() State {
	o.mu.Lock()
	defer o.mu.Unlock()
	s := State{Topology: o.top.Name, Version: o.top.Version, Running: o.running}
	for _, name := range o.top.Order() {
		st := o.units[name].status
		st.Endpoints = append([]Endpoint(nil), st.Endpoints...)
		s.Resources = append(s.Resources, st)
	}
	return s
}

// Logs returns up to n recent log lines of a resource (all when n <= 0).
func (o *Orchestrator) Logs(name string, n int) ([]LogLine, error) {
	if _, ok := o.units[name]; !ok {
		return nil, fmt.Errorf("apphost: unknown resource %q", name)
	}
	return o.logs.tail(name, n), nil
}

// Journal returns every approval decision and executed operation so far.
func (o *Orchestrator) Journal() []OperationRecord {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]OperationRecord(nil), o.journal...)
}

func (o *Orchestrator) record(r OperationRecord) {
	r.Time = time.Now()
	r.TaskID = o.opts.TaskID
	o.mu.Lock()
	o.journal = append(o.journal, r)
	o.mu.Unlock()
	if o.opts.Recorder != nil {
		o.opts.Recorder.Record(r)
	}
	o.emit(Event{Type: EventOperation, Resource: r.Resource, Message: r.Action + " " + r.Outcome})
}

func (o *Orchestrator) emit(e Event) {
	if o.opts.Observer == nil {
		return
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	o.opts.Observer.Observe(e)
}

func (o *Orchestrator) setPhase(u *unit, p Phase, msg string) {
	o.mu.Lock()
	u.status.Phase = p
	u.status.Ready = p == PhaseReady
	if p == PhaseFailed && msg != "" {
		u.status.LastError = msg
	}
	o.mu.Unlock()
	o.logs.system(u.res.Name, "%s %s", p, msg)
	o.emit(Event{Type: EventPhase, Resource: u.res.Name, Phase: p, Message: msg})
}
