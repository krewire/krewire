package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Probe names one of the three orchestration questions.
type Probe string

const (
	// ProbeLiveness: is the process able to make progress? Failing it means
	// "restart me". It must never depend on external dependencies.
	ProbeLiveness Probe = "liveness"
	// ProbeReadiness: should traffic or work be routed here now? Failing it
	// means "route around me"; the process is never restarted for it.
	ProbeReadiness Probe = "readiness"
	// ProbeStartup: has initialization finished? Until it passes, liveness
	// and readiness verdicts are not acted on by the orchestrator.
	ProbeStartup Probe = "startup"
)

// Status is a check or probe verdict (IETF health-check draft vocabulary).
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn" // degraded but serving; probe still succeeds
	StatusFail Status = "fail"
)

// Default probe paths. /healthz and /readyz match pkg/sec.Health.
const (
	PathLiveness  = "/healthz"
	PathReadiness = "/readyz"
	PathStartup   = "/startupz"
)

// DefaultCheckTimeout bounds a check that sets no Timeout.
const DefaultCheckTimeout = 2 * time.Second

// Check is one named health check. A check runs on the probes it lists
// (readiness when none are listed). An Optional check that fails yields warn
// instead of fail, for dependencies the workload can serve without.
type Check struct {
	Name     string
	Probes   []Probe
	Optional bool
	Timeout  time.Duration
	Func     func(ctx context.Context) error
}

func (c Check) on(p Probe) bool {
	if len(c.Probes) == 0 {
		return p == ProbeReadiness
	}
	for _, q := range c.Probes {
		if q == p {
			return true
		}
	}
	return false
}

// CheckResult is one check's outcome. Error is redacted.
type CheckResult struct {
	Name       string  `json:"name"`
	Status     Status  `json:"status"`
	Optional   bool    `json:"optional,omitempty"`
	Error      string  `json:"error,omitempty"`
	DurationMS float64 `json:"duration_ms"`
}

// Report is the JSON body every probe endpoint returns.
type Report struct {
	Probe  Probe         `json:"probe"`
	Status Status        `json:"status"`
	Reason string        `json:"reason,omitempty"`
	Checks []CheckResult `json:"checks,omitempty"`
	Time   time.Time     `json:"time"`
}

// Health is the probe registry of one workload. A nil *Health always passes.
type Health struct {
	mu       sync.RWMutex
	checks   []Check
	started  atomic.Bool
	draining atomic.Bool
	metrics  *Registry
}

// NewHealth returns an empty registry in the "starting" state.
func NewHealth() *Health { return &Health{} }

// Add registers checks. Names must be unique, low-cardinality identifiers.
func (h *Health) Add(checks ...Check) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, checks...)
}

// SetMetrics publishes every evaluation to the standard health gauge. Call
// it during setup, before probes are served.
func (h *Health) SetMetrics(r *Registry) {
	if h != nil {
		h.metrics = r
	}
}

// MarkStarted ends the startup phase; call it once initialization succeeds.
func (h *Health) MarkStarted() {
	if h != nil {
		h.started.Store(true)
	}
}

// MarkDraining flips readiness to fail so load balancers stop routing while
// in-flight work completes; liveness keeps passing. Call it first on shutdown.
func (h *Health) MarkDraining() {
	if h != nil {
		h.draining.Store(true)
	}
}

// Heartbeat adds a liveness check that fails when beat has not been called
// within maxAge — the watchdog for loops without an HTTP surface (workers).
func (h *Health) Heartbeat(name string, maxAge time.Duration) (beat func()) {
	var last atomic.Int64
	last.Store(time.Now().UnixNano())
	h.Add(Check{Name: name, Probes: []Probe{ProbeLiveness}, Func: func(context.Context) error {
		if age := time.Since(time.Unix(0, last.Load())); age > maxAge {
			return fmt.Errorf("no heartbeat for %s (max %s)", age.Round(time.Millisecond), maxAge)
		}
		return nil
	}})
	return func() { last.Store(time.Now().UnixNano()) }
}

// Evaluate runs the checks for p concurrently, each under its own timeout,
// and folds them into one verdict: any required failure fails the probe; an
// optional failure only warns.
func (h *Health) Evaluate(ctx context.Context, p Probe) Report {
	rep := Report{Probe: p, Status: StatusPass, Time: time.Now().UTC()}
	if h == nil {
		return rep
	}
	switch {
	case p != ProbeLiveness && !h.started.Load():
		rep.Status, rep.Reason = StatusFail, "starting"
	case p == ProbeReadiness && h.draining.Load():
		rep.Status, rep.Reason = StatusFail, "draining"
	}
	h.mu.RLock()
	var selected []Check
	for _, c := range h.checks {
		if c.on(p) && c.Func != nil {
			selected = append(selected, c)
		}
	}
	h.mu.RUnlock()

	rep.Checks = make([]CheckResult, len(selected))
	var wg sync.WaitGroup
	for i, c := range selected {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rep.Checks[i] = runCheck(ctx, c)
		}()
	}
	wg.Wait()
	for _, r := range rep.Checks {
		if r.Status == StatusFail {
			rep.Status = StatusFail
		} else if r.Status == StatusWarn && rep.Status == StatusPass {
			rep.Status = StatusWarn
		}
		if h.metrics != nil {
			stdGauge(h.metrics, MetricHealthStatus).Set(statusValue(r.Status), string(p), r.Name)
		}
	}
	return rep
}

func runCheck(ctx context.Context, c Check) CheckResult {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultCheckTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if v := recover(); v != nil {
				done <- fmt.Errorf("check panicked: %v", v)
			}
		}()
		done <- c.Func(cctx)
	}()
	var err error
	select {
	case err = <-done:
	case <-cctx.Done():
		err = fmt.Errorf("timed out after %s", timeout)
	}
	res := CheckResult{Name: c.Name, Status: StatusPass, Optional: c.Optional,
		DurationMS: float64(time.Since(start).Microseconds()) / 1000}
	if err != nil {
		res.Error = RedactString(err.Error())
		res.Status = StatusFail
		if c.Optional {
			res.Status = StatusWarn
		}
	}
	return res
}

func statusValue(s Status) float64 {
	switch s {
	case StatusPass:
		return 1
	case StatusWarn:
		return 0.5
	}
	return 0
}

// Handler serves probe p as JSON: 200 for pass/warn, 503 for fail. Bodies
// carry check names and redacted errors only, never configuration.
func (h *Health) Handler(p Probe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rep := h.Evaluate(r.Context(), p)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		code := http.StatusOK
		if rep.Status == StatusFail {
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(rep)
	})
}

// Paths names the probe endpoints; an empty field disables that endpoint.
type Paths struct {
	Liveness, Readiness, Startup string
}

// DefaultPaths are the standard probe paths.
var DefaultPaths = Paths{Liveness: PathLiveness, Readiness: PathReadiness, Startup: PathStartup}

// Mount registers the probe handlers on mux as GET routes.
func (h *Health) Mount(mux *http.ServeMux, paths Paths) {
	for p, path := range map[Probe]string{ProbeLiveness: paths.Liveness, ProbeReadiness: paths.Readiness, ProbeStartup: paths.Startup} {
		if path != "" {
			mux.Handle("GET "+path, h.Handler(p))
		}
	}
}
