package apphost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// eventLog collects observer events safely.
type eventLog struct {
	mu     sync.Mutex
	events []Event
}

func (l *eventLog) Observe(e Event) {
	l.mu.Lock()
	l.events = append(l.events, e)
	l.mu.Unlock()
}

// phases returns, in order, the resources that reached phase p.
func (l *eventLog) phases(p Phase) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range l.events {
		if e.Type == EventPhase && e.Phase == p {
			out = append(out, e.Resource)
		}
	}
	return out
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func topology(res map[string]*Resource) *Topology {
	return &Topology{Version: SchemaVersion, Name: "test", Resources: res}
}

func fastProbe(r *Resource, timeout time.Duration) *Resource {
	r.Readiness.Interval = Duration(20 * time.Millisecond)
	r.Readiness.Timeout = Duration(timeout)
	return r
}

func tcpEndpoint() []Endpoint { return []Endpoint{{Name: "tcp"}} }

// sampleApp is the app + database + cache + worker topology of #136.
func sampleApp() *Topology {
	db := helper(KindDatabase, "listen", map[string]string{"HELPER_DELAY": "300ms"})
	db.Endpoints = []Endpoint{{Name: "tcp", Scheme: "postgres"}}
	cache := helper(KindCache, "listen", nil)
	cache.Endpoints = []Endpoint{{Name: "tcp", Scheme: "redis"}}
	api := helper(KindService, "api", map[string]string{
		"PORT":       "${self.port}",
		"DB_ADDR":    "${db.host}:${db.port}",
		"CACHE_ADDR": "${cache.tcp.host}:${cache.tcp.port}",
	})
	api.Endpoints = []Endpoint{{Name: "http", Scheme: "http"}}
	api.References = []string{"db", "cache"}
	api.Readiness = &Probe{HTTP: &HTTPProbe{Endpoint: "http", Path: "/healthz"}}
	worker := helper(KindWorker, "worker", map[string]string{"CACHE_ADDR": "${cache.host}:${cache.port}"})
	worker.DependsOn = []string{"api"}
	t := topology(map[string]*Resource{"db": db, "cache": cache, "api": api, "worker": worker})
	t.Normalize()
	for _, r := range []*Resource{db, cache, api} {
		fastProbe(r, 10*time.Second)
	}
	return t
}

func TestSampleTopologyLifecycle(t *testing.T) {
	events := &eventLog{}
	sink := &syncBuffer{}
	o, err := New(sampleApp(), Options{Observer: events, LogSink: sink})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := o.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	st := o.Status()
	if !st.Running || len(st.Resources) != 4 {
		t.Fatalf("status: %+v", st)
	}
	for _, r := range st.Resources {
		if r.Phase != PhaseReady || !r.Ready || r.PID == 0 {
			t.Fatalf("%s not ready: %+v", r.Name, r)
		}
	}
	// Dependents start only after their dependencies are ready.
	db, _ := st.Resource("db")
	api, _ := st.Resource("api")
	worker, _ := st.Resource("worker")
	if api.StartedAt.Before(db.ReadyAt) || worker.StartedAt.Before(api.ReadyAt) {
		t.Fatalf("api started %v before db ready %v (or worker before api)", api.StartedAt, db.ReadyAt)
	}
	if got := events.phases(PhaseReady); strings.Join(got, ",") != "cache,db,api,worker" {
		t.Fatalf("ready order = %v", got)
	}
	// Service discovery: the API serves the injected URLs on its allocated port.
	resp, err := http.Get(endpointURL(api.Endpoints[0]) + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "db=postgres://127.0.0.1:") || !strings.Contains(string(body), "cache=redis://127.0.0.1:") {
		t.Fatalf("discovery env not injected: %s", body)
	}
	if b, err := json.Marshal(st); err != nil || !bytes.Contains(b, []byte(`"phase":"ready"`)) {
		t.Fatalf("state json: %s %v", b, err)
	}
	waitFor(t, func() bool { return logContains(o, "worker", "worker connected to cache") })
	if !logContains(o, "api", "api reached DB_ADDR: ok") {
		t.Fatalf("api logs: %+v", mustLogs(t, o, "api"))
	}

	if err := o.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if got := events.phases(PhaseStopped); strings.Join(got, ",") != "worker,api,db,cache" {
		t.Fatalf("stop order = %v", got)
	}
	for _, name := range []string{"db", "cache"} {
		if !logContains(o, name, "graceful shutdown") {
			t.Fatalf("%s did not shut down gracefully: %+v", name, mustLogs(t, o, name))
		}
	}
	if !strings.Contains(sink.String(), "worker | worker draining") {
		t.Fatalf("aggregated log sink missing worker output:\n%s", sink.String())
	}
	for _, r := range o.Status().Resources {
		if r.Phase != PhaseStopped || r.PID != 0 {
			t.Fatalf("%s after stop: %+v", r.Name, r)
		}
	}
	for _, r := range st.Resources {
		waitFor(t, func() bool { return !processAlive(r.PID) })
	}
	if err := o.Stop(ctx); err != nil {
		t.Fatalf("second Stop must be a no-op: %v", err)
	}
	if err := o.Start(ctx); err == nil {
		t.Fatal("a stopped orchestrator must not restart")
	}
}

func logContains(o *Orchestrator, res, text string) bool {
	lines, _ := o.Logs(res, 0)
	for _, l := range lines {
		if strings.Contains(l.Text, text) {
			return true
		}
	}
	return false
}

func mustLogs(t *testing.T, o *Orchestrator, res string) []LogLine {
	t.Helper()
	l, err := o.Logs(res, 0)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 10s")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestPartialStartupFailureRollsBack(t *testing.T) {
	db := helper(KindDatabase, "listen", nil)
	db.Endpoints = tcpEndpoint()
	api := helper(KindService, "crash", nil)
	api.Endpoints = []Endpoint{{Name: "http"}}
	api.References = []string{"db"}
	never := helper(KindWorker, "silent", nil)
	never.DependsOn = []string{"api"}
	top := topology(map[string]*Resource{"db": db, "api": api, "worker": never})
	top.Normalize()
	fastProbe(db, 5*time.Second)
	fastProbe(api, 5*time.Second)
	o, err := New(top, Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = o.Start(context.Background())
	var se *StartError
	if !errors.As(err, &se) {
		t.Fatalf("want *StartError, got %v", err)
	}
	if se.Resource != "api" || se.Stage != "readiness" || strings.Join(se.RolledBack, ",") != "db" {
		t.Fatalf("diagnostic: %+v", se)
	}
	if !strings.Contains(se.Error(), "boom: cannot open database") || !strings.Contains(se.Error(), "hint:") {
		t.Fatalf("diagnostic lacks log tail or hint:\n%s", se.Error())
	}
	st := o.Status()
	want := map[string]Phase{"db": PhaseStopped, "api": PhaseFailed, "worker": PhasePending}
	for name, p := range want {
		if r, _ := st.Resource(name); r.Phase != p {
			t.Fatalf("%s phase = %s, want %s", name, r.Phase, p)
		}
	}
	if st.Running {
		t.Fatal("rolled back topology must not be running")
	}
	if err := o.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessTimeout(t *testing.T) {
	r := helper(KindService, "silent", nil)
	r.Endpoints = tcpEndpoint()
	top := topology(map[string]*Resource{"svc": r})
	top.Normalize()
	fastProbe(r, 300*time.Millisecond)
	o, _ := New(top, Options{})
	err := o.Start(context.Background())
	var se *StartError
	if !errors.As(err, &se) || se.Stage != "readiness" || !strings.Contains(se.Message, "did not pass within 300ms") {
		t.Fatalf("got %v", err)
	}
}

func TestShutdownTimeoutKillsStubbornProcess(t *testing.T) {
	r := helper(KindWorker, "stubborn", nil)
	r.ShutdownTimeout = Duration(300 * time.Millisecond)
	o, _ := New(topology(map[string]*Resource{"w": r}), Options{})
	if err := o.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return logContains(o, "w", "ignoring SIGTERM") })
	pid := o.Status().Resources[0].PID
	began := time.Now()
	err := o.Stop(context.Background())
	if !errors.Is(err, ErrShutdownTimeout) {
		t.Fatalf("want ErrShutdownTimeout, got %v", err)
	}
	if d := time.Since(began); d < 300*time.Millisecond || d > 5*time.Second {
		t.Fatalf("stop took %v", d)
	}
	waitFor(t, func() bool { return !processAlive(pid) })
	if j := o.Journal(); j[len(j)-1].Outcome != OutcomeFailed {
		t.Fatalf("journal: %+v", j)
	}
}
