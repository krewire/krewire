package observe

import (
	"slices"
	"time"
)

// Workload is a canonical Krewire workload kind as declared in krewire.yaml.
type Workload string

const (
	WorkloadApp     Workload = "app"
	WorkloadService Workload = "service"
	WorkloadGateway Workload = "gateway"
	WorkloadWorker  Workload = "worker"
	WorkloadSite    Workload = "site"
	WorkloadBook    Workload = "book"
	WorkloadInfra   Workload = "infra"
	WorkloadCLI     Workload = "cli"
)

// ProbeMode is how a workload answers its probes.
type ProbeMode string

const (
	// ModeHTTP serves the probe endpoints from the workload's listener.
	ModeHTTP ProbeMode = "http"
	// ModeHeartbeat drives liveness from Health.Heartbeat; probes are served
	// only on an optional admin listener.
	ModeHeartbeat ProbeMode = "heartbeat"
	// ModeStatic means a static host answers; GET of the index is the probe.
	ModeStatic ProbeMode = "static"
	// ModeExitCode means a short-lived process; its exit status is the probe
	// and its diagnostics bundle is the readiness evidence.
	ModeExitCode ProbeMode = "exit_code"
)

// PathMetrics is the conventional Prometheus scrape path.
const PathMetrics = "/metrics"

// PathDiagnostics serves the diagnostics bundle; mount it on an admin or
// loopback listener only.
const PathDiagnostics = "/debug/diagnostics"

// Profile is the observability default for one workload kind. Applications
// may override any field; the defaults keep every workload consistent.
type Profile struct {
	Workload     Workload      `json:"workload"`
	Mode         ProbeMode     `json:"mode"`
	Probes       Paths         `json:"probes"`
	MetricsPath  string        `json:"metrics_path,omitempty"`
	DrainTimeout time.Duration `json:"drain_timeout"`
	// Metrics lists the standard metrics this workload is expected to emit.
	Metrics []string `json:"metrics,omitempty"`
	// Correlation lists the fields every log record of this workload carries
	// beyond the identity fields.
	Correlation []string `json:"correlation"`
}

var (
	runtimeMetrics = []string{MetricProcessStart, MetricGoroutines, MetricHeapAlloc}
	httpMetrics    = append([]string{MetricHTTPRequests, MetricHTTPDuration, MetricHTTPInFlight}, runtimeMetrics...)
	traceFields    = []string{FieldTraceID, FieldSpanID, FieldRequestID}
	taskFields     = []string{FieldTraceID, FieldSpanID, FieldSessionID, FieldTaskID, FieldAgentRunID, FieldToolCallID, FieldProcessID}
)

func with(base []string, extra ...string) []string {
	return append(append([]string(nil), base...), extra...)
}

// profiles is the single table of per-workload defaults.
var profiles = map[Workload]Profile{
	WorkloadApp: {Workload: WorkloadApp, Mode: ModeHTTP, Probes: DefaultPaths, MetricsPath: PathMetrics,
		DrainTimeout: 10 * time.Second, Metrics: httpMetrics, Correlation: traceFields},
	WorkloadService: {Workload: WorkloadService, Mode: ModeHTTP, Probes: DefaultPaths, MetricsPath: PathMetrics,
		DrainTimeout: 10 * time.Second, Metrics: with(httpMetrics, MetricRetries, MetricBreakerState), Correlation: traceFields},
	WorkloadGateway: {Workload: WorkloadGateway, Mode: ModeHTTP, Probes: DefaultPaths, MetricsPath: PathMetrics,
		DrainTimeout: 15 * time.Second, Metrics: with(httpMetrics, MetricRetries, MetricBreakerState), Correlation: traceFields},
	WorkloadWorker: {Workload: WorkloadWorker, Mode: ModeHeartbeat, Probes: DefaultPaths, MetricsPath: PathMetrics,
		DrainTimeout: 30 * time.Second, Metrics: with(runtimeMetrics, MetricJobs, MetricJobDuration, MetricQueueDepth, MetricRetries),
		Correlation: with(traceFields, FieldJobID, FieldQueue)},
	WorkloadSite: {Workload: WorkloadSite, Mode: ModeStatic, Probes: Paths{Liveness: "/", Readiness: "/"},
		Correlation: []string{}},
	WorkloadBook: {Workload: WorkloadBook, Mode: ModeStatic, Probes: Paths{Liveness: "/index.html", Readiness: "/index.html"},
		Correlation: []string{}},
	WorkloadInfra: {Workload: WorkloadInfra, Mode: ModeExitCode, DrainTimeout: 30 * time.Second,
		Metrics: []string{MetricCommands, MetricCommandDuration}, Correlation: taskFields},
	WorkloadCLI: {Workload: WorkloadCLI, Mode: ModeExitCode, DrainTimeout: 2 * time.Second,
		Metrics: []string{MetricCommands, MetricCommandDuration}, Correlation: taskFields},
}

// Workloads lists the canonical kinds in documentation order.
var Workloads = []Workload{WorkloadApp, WorkloadService, WorkloadGateway, WorkloadWorker,
	WorkloadSite, WorkloadBook, WorkloadInfra, WorkloadCLI}

// ProfileFor returns a copy of the defaults for w.
func ProfileFor(w Workload) (Profile, bool) {
	p, ok := profiles[w]
	if !ok {
		return Profile{}, false
	}
	p.Metrics = slices.Clone(p.Metrics)
	p.Correlation = slices.Clone(p.Correlation)
	return p, true
}
