package apphost

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/model"
)

// Phase is the lifecycle phase of one resource.
type Phase string

// Phases, in the order a healthy resource visits them.
const (
	PhasePending    Phase = "pending"
	PhaseStarting   Phase = "starting"
	PhaseReady      Phase = "ready"
	PhaseRestarting Phase = "restarting"
	PhaseStopping   Phase = "stopping"
	PhaseStopped    Phase = "stopped"
	PhaseExited     Phase = "exited" // completed with exit 0 and no restart
	PhaseFailed     Phase = "failed"
)

// ResourceStatus is the machine-readable state of one resource.
type ResourceStatus struct {
	Name      string     `json:"name"`
	Kind      Kind       `json:"kind"`
	Runner    string     `json:"runner"`
	Phase     Phase      `json:"phase"`
	Ready     bool       `json:"ready"`
	PID       int        `json:"pid,omitempty"`
	Restarts  int        `json:"restarts"`
	Endpoints []Endpoint `json:"endpoints,omitempty"`
	StartedAt time.Time  `json:"startedAt,omitempty"`
	ReadyAt   time.Time  `json:"readyAt,omitempty"`
	LastProbe string     `json:"lastProbeError,omitempty"`
	LastError string     `json:"lastError,omitempty"`
}

// State is the machine-readable snapshot of a running topology; it is what
// `krew app status --json` prints and what AI workflows receive as context.
type State struct {
	Topology  string           `json:"topology,omitempty"`
	Version   int              `json:"version"`
	Running   bool             `json:"running"`
	Resources []ResourceStatus `json:"resources"`
}

// Resource returns the status of name.
func (s State) Resource(name string) (ResourceStatus, bool) {
	for _, r := range s.Resources {
		if r.Name == name {
			return r, true
		}
	}
	return ResourceStatus{}, false
}

// Event types emitted to the Observer.
const (
	EventPhase       = "phase"
	EventProbeFailed = "probe-failed"
	EventOperation   = "operation"
)

// Event is a structured lifecycle notification. It is deliberately small and
// flat so observability (#139) can map it onto its standard health/status
// model without importing orchestrator internals.
type Event struct {
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
	Resource string    `json:"resource"`
	Phase    Phase     `json:"phase,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// Observer receives events synchronously; it must not block.
type Observer interface {
	Observe(Event)
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(Event)

// Observe implements Observer.
func (f ObserverFunc) Observe(e Event) { f(e) }

// Actions an Operation can carry.
const (
	ActionStart = "start"
	ActionStop  = "stop"
)

// Operation is a request to act on resources, presented to the Approver.
type Operation struct {
	Action    string   `json:"action"`
	Topology  string   `json:"topology,omitempty"`
	Resources []string `json:"resources"`
	Actor     string   `json:"actor,omitempty"`
}

// Intent maps the operation onto the shared AI action-intent contract, so the
// same policy engine that gates agent tool calls can gate orchestration.
func (op Operation) Intent() model.ActionIntent {
	return model.ActionIntent{
		ID:          op.Action + ":" + op.Topology,
		Category:    model.ActionCommand,
		ToolName:    "apphost." + op.Action,
		Target:      strings.Join(op.Resources, ","),
		Description: fmt.Sprintf("%s %d resource(s) of topology %q", op.Action, len(op.Resources), op.Topology),
		Autonomy:    model.AutonomyApprovalRequired,
	}
}

// Decision is an Approver's verdict.
type Decision struct {
	Approved bool   `json:"approved"`
	By       string `json:"by,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// Approver gates operations with external effects (starting processes). Stop
// and rollback are never gated: cleanup guarantees must hold regardless.
type Approver interface {
	Approve(ctx context.Context, op Operation) (Decision, error)
}

// ApproverFunc adapts a function to Approver.
type ApproverFunc func(context.Context, Operation) (Decision, error)

// Approve implements Approver.
func (f ApproverFunc) Approve(ctx context.Context, op Operation) (Decision, error) { return f(ctx, op) }

// OperationRecord is one journal entry: an approval decision or an executed
// resource operation, keyed to the shared task record by TaskID.
type OperationRecord struct {
	Time     time.Time `json:"time"`
	TaskID   string    `json:"taskId,omitempty"`
	Action   string    `json:"action"`
	Resource string    `json:"resource,omitempty"`
	Decision *Decision `json:"decision,omitempty"`
	Outcome  string    `json:"outcome"`
	Error    string    `json:"error,omitempty"`
}

// Outcomes of an OperationRecord.
const (
	OutcomeApproved = "approved"
	OutcomeDenied   = "denied"
	OutcomeOK       = "ok"
	OutcomeFailed   = "failed"
)

// Recorder persists journal entries (for example into the task evidence
// ledger). It must not block.
type Recorder interface {
	Record(OperationRecord)
}

// StartError is returned when Start fails. Everything that had started was
// stopped in reverse order before it is returned.
type StartError struct {
	Resource   string    `json:"resource,omitempty"`
	Stage      string    `json:"stage"` // approval, preflight, launch, readiness
	Cause      error     `json:"-"`
	Message    string    `json:"message"`
	Hint       string    `json:"hint,omitempty"`
	LogTail    []LogLine `json:"logTail,omitempty"`
	RolledBack []string  `json:"rolledBack,omitempty"`
}

func (e *StartError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "apphost: start failed at %s", e.Stage)
	if e.Resource != "" {
		fmt.Fprintf(&b, " of %q", e.Resource)
	}
	fmt.Fprintf(&b, ": %s", e.Message)
	if len(e.RolledBack) > 0 {
		fmt.Fprintf(&b, "\n  rolled back (stopped in order): %s", strings.Join(e.RolledBack, ", "))
	}
	if e.Hint != "" {
		fmt.Fprintf(&b, "\n  hint: %s", e.Hint)
	}
	if len(e.LogTail) > 0 {
		b.WriteString("\n  last output:")
		for _, l := range e.LogTail {
			fmt.Fprintf(&b, "\n    %s", l.Text)
		}
	}
	return b.String()
}

// Unwrap exposes the cause.
func (e *StartError) Unwrap() error { return e.Cause }
