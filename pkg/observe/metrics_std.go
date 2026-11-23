package observe

import (
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// Standard metric names. They are a stable contract: dashboards, alerts, and
// AI diagnostics key on them, so renaming one is a breaking change.
const (
	MetricHTTPRequests    = "krewire_http_server_requests_total"
	MetricHTTPDuration    = "krewire_http_server_request_duration_seconds"
	MetricHTTPInFlight    = "krewire_http_server_in_flight_requests"
	MetricJobs            = "krewire_worker_jobs_total"
	MetricJobDuration     = "krewire_worker_job_duration_seconds"
	MetricQueueDepth      = "krewire_worker_queue_depth"
	MetricRetries         = "krewire_resilience_retries_total"
	MetricBreakerState    = "krewire_resilience_breaker_state"
	MetricHealthStatus    = "krewire_health_check_status"
	MetricCommands        = "krewire_toolchain_commands_total"
	MetricCommandDuration = "krewire_toolchain_command_duration_seconds"
	MetricProcessStart    = "krewire_process_start_time_seconds"
	MetricGoroutines      = "krewire_go_goroutines"
	MetricHeapAlloc       = "krewire_go_heap_alloc_bytes"
)

// Job and command outcomes: the only values the "outcome" label may take.
const (
	OutcomeSuccess    = "success"
	OutcomeFailure    = "failure"
	OutcomeRetry      = "retry"
	OutcomeDeadLetter = "dead_letter"
)

// StandardMetrics is the single table of standard metric descriptions. Every
// label listed here has a bounded value set; docs/pkg/observe.md mirrors it.
var StandardMetrics = []Desc{
	{Name: MetricHTTPRequests, Kind: KindCounter, Unit: UnitCount, Labels: []string{"method", "route", "status_class"},
		Help: "HTTP requests served, by method, route pattern, and status class (2xx..5xx)."},
	{Name: MetricHTTPDuration, Kind: KindHistogram, Unit: UnitSeconds, Labels: []string{"method", "route"},
		Help: "HTTP request latency in seconds."},
	{Name: MetricHTTPInFlight, Kind: KindGauge, Unit: UnitCount,
		Help: "HTTP requests currently being served."},
	{Name: MetricJobs, Kind: KindCounter, Unit: UnitCount, Labels: []string{"queue", "outcome"},
		Help: "Worker job attempts by queue and outcome (success, failure, retry, dead_letter)."},
	{Name: MetricJobDuration, Kind: KindHistogram, Unit: UnitSeconds, Labels: []string{"queue"},
		Help: "Worker job attempt duration in seconds."},
	{Name: MetricQueueDepth, Kind: KindGauge, Unit: UnitCount, Labels: []string{"queue"},
		Help: "Jobs waiting in a queue."},
	{Name: MetricRetries, Kind: KindCounter, Unit: UnitCount, Labels: []string{"operation"},
		Help: "Retry attempts by logical operation name."},
	{Name: MetricBreakerState, Kind: KindGauge, Unit: UnitCount, Labels: []string{"breaker"},
		Help: "Circuit breaker state: 0 closed, 1 half-open, 2 open."},
	{Name: MetricHealthStatus, Kind: KindGauge, Unit: UnitCount, Labels: []string{"probe", "check"},
		Help: "Last health check result: 1 pass, 0.5 warn, 0 fail."},
	{Name: MetricCommands, Kind: KindCounter, Unit: UnitCount, Labels: []string{"command", "outcome", "origin"},
		Help: "Toolchain commands by command, outcome, and origin (command, agent, provider, runtime)."},
	{Name: MetricCommandDuration, Kind: KindHistogram, Unit: UnitSeconds, Labels: []string{"command"},
		Help: "Toolchain command duration in seconds."},
	{Name: MetricProcessStart, Kind: KindGauge, Unit: UnitSeconds,
		Help: "Process start time as Unix seconds."},
	{Name: MetricGoroutines, Kind: KindGauge, Unit: UnitCount,
		Help: "Live goroutines."},
	{Name: MetricHeapAlloc, Kind: KindGauge, Unit: UnitBytes,
		Help: "Bytes of allocated heap objects."},
}

// StandardDesc returns the standard description for name.
func StandardDesc(name string) (Desc, bool) {
	for _, d := range StandardMetrics {
		if d.Name == name {
			return d, true
		}
	}
	return Desc{}, false
}

// Std returns the registered standard instrument for name, registering it
// on first use. It panics for an unknown name, which is a programming error.
func (r *Registry) Std(name string) any {
	d, ok := StandardDesc(name)
	if !ok {
		panic("observe: unknown standard metric " + name)
	}
	return r.Register(d)
}

func stdCounter(r *Registry, name string) *Counter {
	c, _ := r.Std(name).(*Counter)
	return c
}

func stdGauge(r *Registry, name string) *Gauge {
	g, _ := r.Std(name).(*Gauge)
	return g
}

func stdHistogram(r *Registry, name string) *Histogram {
	h, _ := r.Std(name).(*Histogram)
	return h
}

// CollectRuntime registers the process gauges and refreshes them on every
// collection. It is cheap: one runtime.ReadMemStats per scrape.
func (r *Registry) CollectRuntime() {
	if r == nil {
		return
	}
	start := float64(time.Now().Unix())
	stdGauge(r, MetricProcessStart).Set(start)
	goroutines, heap := stdGauge(r, MetricGoroutines), stdGauge(r, MetricHeapAlloc)
	r.OnCollect(func() {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		goroutines.Set(float64(runtime.NumGoroutine()))
		heap.Set(float64(ms.HeapAlloc))
	})
}

// InstrumentHTTP records the standard HTTP server metrics. The route label is
// the ServeMux pattern that matched (bounded by construction), or "unmatched";
// raw paths are never used as labels. A nil registry returns next unchanged.
func InstrumentHTTP(r *Registry, next http.Handler) http.Handler {
	if r == nil {
		return next
	}
	reqs, dur, inflight := stdCounter(r, MetricHTTPRequests), stdHistogram(r, MetricHTTPDuration), stdGauge(r, MetricHTTPInFlight)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		inflight.Add(1)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			inflight.Add(-1)
			route := req.Pattern
			if route == "" {
				route = "unmatched"
			}
			method := boundedMethod(req.Method)
			reqs.Inc(method, route, strconv.Itoa(rec.status/100)+"xx")
			dur.Observe(time.Since(start).Seconds(), method, route)
		}()
		next.ServeHTTP(rec, req)
	})
}

func boundedMethod(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return m
	}
	return "OTHER"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status, s.wrote = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// ObserveJob records one worker job attempt with the standard metrics.
func ObserveJob(r *Registry, queue, outcome string, d time.Duration) {
	if r == nil {
		return
	}
	stdCounter(r, MetricJobs).Inc(queue, outcome)
	stdHistogram(r, MetricJobDuration).Observe(d.Seconds(), queue)
}

// ObserveCommand records one toolchain command execution.
func ObserveCommand(r *Registry, command, outcome string, origin Origin, d time.Duration) {
	if r == nil {
		return
	}
	stdCounter(r, MetricCommands).Inc(command, outcome, string(origin))
	stdHistogram(r, MetricCommandDuration).Observe(d.Seconds(), command)
}
