package apphost

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRestartOnFailureRecovers(t *testing.T) {
	r := helper(KindCache, "crashonce", map[string]string{"HELPER_MARKER": filepath.Join(t.TempDir(), "marker")})
	r.Endpoints = tcpEndpoint()
	r.Restart = Restart{Policy: RestartOnFailure, MaxRestarts: 2, Backoff: Duration(10 * time.Millisecond)}
	top := topology(map[string]*Resource{"cache": r})
	top.Normalize()
	fastProbe(r, 5*time.Second)
	o, _ := New(top, Options{})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { s, _ := o.Status().Resource("cache"); return s.Restarts == 1 && s.Phase == PhaseReady })
	if !logContains(o, "cache", "crashing after ready") {
		t.Fatalf("logs: %+v", mustLogs(t, o, "cache"))
	}
	_ = o.Stop(context.Background())

	events := &eventLog{}
	job := helper(KindWorker, "exit0", nil)
	job.Restart = Restart{Policy: RestartAlways, MaxRestarts: 2, Backoff: Duration(5 * time.Millisecond)}
	o, _ = New(topology(map[string]*Resource{"job": job}), Options{Observer: events})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { s, _ := o.Status().Resource("job"); return s.Phase == PhaseExited })
	if s, _ := o.Status().Resource("job"); s.Restarts != 2 {
		t.Fatalf("restarts = %d, want 2", s.Restarts)
	}
	if got := len(events.phases(PhaseRestarting)); got != 2 {
		t.Fatalf("restarting events = %d", got)
	}
	_ = o.Stop(context.Background())

	crash := helper(KindWorker, "crashlater", nil)
	crash.Restart = Restart{Policy: RestartOnFailure, MaxRestarts: 1, Backoff: Duration(5 * time.Millisecond)}
	o, _ = New(topology(map[string]*Resource{"c": crash}), Options{})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { s, _ := o.Status().Resource("c"); return s.Phase == PhaseFailed })
	if s, _ := o.Status().Resource("c"); s.Restarts != 1 || !strings.Contains(s.LastError, "exit status 4") {
		t.Fatalf("status: %+v", s)
	}
	_ = o.Stop(context.Background())
}

func TestApprovalGateAndJournal(t *testing.T) {
	var mu sync.Mutex
	var recorded []OperationRecord
	rec := recorderFunc(func(r OperationRecord) { mu.Lock(); recorded = append(recorded, r); mu.Unlock() })
	var seen Operation
	deny := ApproverFunc(func(_ context.Context, op Operation) (Decision, error) {
		seen = op
		return Decision{Approved: false, By: "reviewer", Reason: "not today"}, nil
	})
	top := topology(map[string]*Resource{"w": helper(KindWorker, "silent", nil)})
	o, _ := New(top, Options{Approver: deny, Recorder: rec, TaskID: "task-7", Actor: "agent-1"})
	err := o.Start(context.Background())
	var se *StartError
	if !errors.As(err, &se) || se.Stage != "approval" {
		t.Fatalf("want approval StartError, got %v", err)
	}
	if s, _ := o.Status().Resource("w"); s.Phase != PhasePending || s.PID != 0 {
		t.Fatalf("denied start must launch nothing: %+v", s)
	}
	if seen.Actor != "agent-1" || !reflect.DeepEqual(seen.Resources, []string{"w"}) {
		t.Fatalf("operation: %+v", seen)
	}
	intent := seen.Intent()
	if intent.ToolName != "apphost.start" || intent.Target != "w" || intent.Autonomy == "" {
		t.Fatalf("intent: %+v", intent)
	}
	if len(recorded) != 1 || recorded[0].Outcome != OutcomeDenied || recorded[0].TaskID != "task-7" || recorded[0].Decision.By != "reviewer" {
		t.Fatalf("journal: %+v", recorded)
	}

	allow := ApproverFunc(func(context.Context, Operation) (Decision, error) { return Decision{Approved: true, By: "human"}, nil })
	o, _ = New(top, Options{Approver: allow, TaskID: "task-8"})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := o.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	var outcomes []string
	for _, r := range o.Journal() {
		if r.TaskID != "task-8" {
			t.Fatalf("record not tied to task: %+v", r)
		}
		outcomes = append(outcomes, r.Action+":"+r.Outcome)
	}
	if strings.Join(outcomes, ",") != "start:approved,start:ok,stop:ok" {
		t.Fatalf("journal = %v", outcomes)
	}
	failing := ApproverFunc(func(context.Context, Operation) (Decision, error) { return Decision{}, errors.New("policy engine offline") })
	o, _ = New(top, Options{Approver: failing})
	if err := o.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "policy engine offline") {
		t.Fatalf("approver error must block start: %v", err)
	}
}

type recorderFunc func(OperationRecord)

func (f recorderFunc) Record(r OperationRecord) { f(r) }

func TestFixedPortInUseFailsBeforeStarting(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	first := helper(KindCache, "silent", nil)
	busy := helper(KindService, "silent", nil)
	busy.Endpoints = []Endpoint{{Name: "http", Port: port}}
	o, err := New(topology(map[string]*Resource{"a": first, "b": busy}), Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = o.Start(context.Background())
	var ve *ValidationError
	if !errors.As(err, &ve) || !ve.Has(CodePortInUse) {
		t.Fatalf("want port-in-use, got %v", err)
	}
	if s, _ := o.Status().Resource("a"); s.PID != 0 {
		t.Fatal("nothing may start when preflight fails")
	}
}

func TestStopDoesNotOrphanGrandchildren(t *testing.T) {
	r := helper(KindWorker, "spawn", nil)
	r.ShutdownTimeout = Duration(2 * time.Second)
	o, _ := New(topology(map[string]*Resource{"p": r}), Options{})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var child int
	waitFor(t, func() bool {
		for _, l := range mustLogs(t, o, "p") {
			if v, ok := strings.CutPrefix(l.Text, "child="); ok {
				child, _ = strconv.Atoi(v)
			}
		}
		return child > 0
	})
	if !processAlive(child) {
		t.Skip("cannot observe grandchild on this platform")
	}
	// The grandchild ignores SIGTERM; the parent exits gracefully, so the
	// orchestrator must reap the rest of the process group.
	if err := o.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitFor(t, func() bool { return !processAlive(child) })
}

func TestStopInterruptsStartAndCapsTimeout(t *testing.T) {
	r := helper(KindDatabase, "silent", nil)
	r.Endpoints = tcpEndpoint()
	top := topology(map[string]*Resource{"db": r})
	top.Normalize()
	fastProbe(r, time.Minute)
	o, _ := New(top, Options{})
	done := make(chan error, 1)
	go func() { done <- o.Start(context.Background()) }()
	waitFor(t, func() bool { s, _ := o.Status().Resource("db"); return s.PID != 0 })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("interrupted Start must report an error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start was not interrupted")
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	o, _ := New(topology(map[string]*Resource{"w": helper(KindWorker, "silent", nil)}), Options{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- o.Run(ctx) }()
	waitFor(t, func() bool { return o.Status().Running })
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if o.Status().Running {
		t.Fatal("Run must stop the topology")
	}
	if _, err := o.Logs("ghost", 1); err == nil {
		t.Fatal("unknown resource logs must fail")
	}
}

func TestExternalResourceIsProbedNotLaunched(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	ext := &Resource{Kind: KindExternal, External: &ExternalSpec{URL: "amqp://broker"}, Endpoints: []Endpoint{{Name: "amqp", Port: port}}}
	ext.Readiness = &Probe{TCP: "amqp"}
	w := helper(KindWorker, "silent", nil)
	w.References = []string{"mq"}
	top := topology(map[string]*Resource{"mq": ext, "w": w})
	o, err := New(top, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	s, _ := o.Status().Resource("mq")
	if s.Phase != PhaseReady || s.PID != 0 {
		t.Fatalf("external: %+v", s)
	}
	env := DiscoveryEnv(o.Topology(), o.Topology().Resources["w"])
	if env["KREWIRE_MQ_URL"] != "amqp://broker" || env["KREWIRE_MQ_PORT"] != strconv.Itoa(port) || env["KREWIRE_MQ_AMQP_URL"] == "" {
		t.Fatalf("discovery env: %v", env)
	}
	if err := o.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestContainerDriverIsPluggable(t *testing.T) {
	r := &Resource{Name: "redis", Kind: KindCache, Container: &ContainerSpec{Image: "redis:7", Args: []string{"--save", ""}},
		Endpoints: []Endpoint{{Name: "tcp", Host: "127.0.0.1", Port: 6380, TargetPort: 6379}}, Env: map[string]string{"MODE": "x"}}
	args := ContainerDriver{}.RunArgs(LaunchSpec{Resource: r, Env: []string{"HOME=/root", "MODE=x", "KREWIRE_RESOURCE=redis"}})
	want := "run --rm --name krewire-redis -p 127.0.0.1:6380:6379 -e MODE=x -e KREWIRE_RESOURCE=redis redis:7 --save "
	if strings.Join(args, " ") != want {
		t.Fatalf("args = %q", strings.Join(args, " "))
	}
	if _, err := (ContainerDriver{}).Start(context.Background(), LaunchSpec{Resource: &Resource{Name: "x"}}); err == nil {
		t.Fatal("non-container resource must be rejected")
	}
	if _, err := (ProcessDriver{}).Start(context.Background(), LaunchSpec{Resource: &Resource{Name: "x"}}); err == nil {
		t.Fatal("non-process resource must be rejected")
	}
	// Any Driver can back the container runner: here the "container" is a
	// local child process, so the test needs no Docker.
	c := &Resource{Kind: KindCache, Container: &ContainerSpec{Image: "fake"}}
	o, err := New(topology(map[string]*Resource{"c": c}), Options{Drivers: map[string]Driver{RunnerContainer: fakeContainer{}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s, _ := o.Status().Resource("c"); s.Runner != RunnerContainer || s.Phase != PhaseReady {
		t.Fatalf("status: %+v", s)
	}
	if err := o.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type fakeContainer struct{}

func (fakeContainer) Start(ctx context.Context, spec LaunchSpec) (Process, error) {
	r := *spec.Resource
	r.Run = &RunSpec{Command: helper(KindCache, "silent", nil).Run.Command}
	spec.Resource = &r
	spec.Env = append(spec.Env, helperEnv+"=silent")
	return ProcessDriver{}.Start(ctx, spec)
}

func TestLaunchFailureIsDiagnosed(t *testing.T) {
	bad := &Resource{Kind: KindService, Run: &RunSpec{Command: filepath.Join(t.TempDir(), "missing-binary")}}
	o, _ := New(topology(map[string]*Resource{"bad": bad}), Options{})
	err := o.Start(context.Background())
	var se *StartError
	if !errors.As(err, &se) || se.Stage != "launch" || !strings.Contains(se.Hint, "run.command") {
		t.Fatalf("got %v", err)
	}
	if _, err := New(topology(map[string]*Resource{"x": {Kind: "nope"}}), Options{}); err == nil {
		t.Fatal("New must validate")
	}
}
