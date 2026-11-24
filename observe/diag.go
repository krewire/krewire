package observe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

// BundleSchema versions the diagnostics bundle JSON. Consumers (krew doctor,
// CI, AI workflows) must check it before reading other fields.
const BundleSchema = "krewire.diagnostics/v1"

// Severity grades a finding.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Finding is one diagnosis. Origin says who asserted it (measured runtime or
// command vs. agent-reported); Class says which layer owns the failure.
// TraceID links the finding to underlying logs instead of a narrative.
type Finding struct {
	Code     string       `json:"code"`
	Severity Severity     `json:"severity"`
	Origin   Origin       `json:"origin"`
	Class    FailureClass `json:"class,omitempty"`
	Message  string       `json:"message"`
	Hint     string       `json:"hint,omitempty"`
	TraceID  string       `json:"trace_id,omitempty"`
}

// RuntimeInfo describes the process. It carries no hostnames or user names.
type RuntimeInfo struct {
	GoVersion      string    `json:"go_version"`
	OS             string    `json:"os"`
	Arch           string    `json:"arch"`
	NumCPU         int       `json:"num_cpu"`
	Goroutines     int       `json:"goroutines"`
	HeapAllocBytes uint64    `json:"heap_alloc_bytes"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"started_at"`
	UptimeSeconds  float64   `json:"uptime_seconds"`
}

// BuildInfo is read from the binary's embedded module data.
type BuildInfo struct {
	Module      string `json:"module,omitempty"`
	Version     string `json:"version,omitempty"`
	VCSRevision string `json:"vcs_revision,omitempty"`
	VCSTime     string `json:"vcs_time,omitempty"`
	VCSModified bool   `json:"vcs_modified,omitempty"`
}

// Bundle is the diagnostics report: one JSON document a human can read, a CI
// job can gate on (ExitCode), and an AI workflow can cite by trace ID.
type Bundle struct {
	Schema      string            `json:"schema"`
	Status      Status            `json:"status"`
	GeneratedAt time.Time         `json:"generated_at"`
	Identity    Identity          `json:"identity"`
	Profile     *Profile          `json:"profile,omitempty"`
	Correlation IDs               `json:"correlation"`
	TraceID     string            `json:"trace_id,omitempty"`
	Runtime     RuntimeInfo       `json:"runtime"`
	Build       BuildInfo         `json:"build"`
	Probes      []Report          `json:"probes,omitempty"`
	Metrics     []Family          `json:"metrics,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
	Findings    []Finding         `json:"findings"`
}

// ExitCode maps the bundle to a process exit status: 1 when any finding is
// an error (or, with strict, a warning), else 0.
func (b Bundle) ExitCode(strict bool) int {
	for _, f := range b.Findings {
		if f.Severity == SeverityError || (strict && f.Severity == SeverityWarn) {
			return 1
		}
	}
	return 0
}

// WriteJSON writes b as indented JSON.
func (b Bundle) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(b)
}

// DoctorCheck contributes findings to a bundle, e.g. "config file present"
// or "migration level current". Run must honor ctx and never return secrets.
type DoctorCheck struct {
	Name string
	Run  func(ctx context.Context) []Finding
}

// Diagnostics assembles bundles for one workload. Every field is optional.
type Diagnostics struct {
	Identity Identity
	Workload Workload
	Health   *Health
	Metrics  *Registry
	// Config returns effective settings to include; values are redacted.
	Config func() map[string]string
	Checks []DoctorCheck
	// StartedAt defaults to the time the package was initialized.
	StartedAt time.Time
}

var processStart = time.Now()

// Collect evaluates every probe, snapshots metrics, runs doctor checks, and
// derives findings. The result is redacted and safe to hand to an AI agent.
func (d *Diagnostics) Collect(ctx context.Context) Bundle {
	started := d.StartedAt
	if started.IsZero() {
		started = processStart
	}
	b := Bundle{Schema: BundleSchema, Status: StatusPass, GeneratedAt: time.Now().UTC(), Identity: d.Identity,
		Correlation: IDsFrom(ctx), TraceID: SpanFromContext(ctx).TraceID.String(), Build: readBuildInfo(),
		Findings: []Finding{}}
	if b.Identity.Workload == "" {
		b.Identity.Workload = string(d.Workload)
	}
	if p, ok := ProfileFor(d.Workload); ok {
		b.Profile = &p
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	b.Runtime = RuntimeInfo{GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH,
		NumCPU: runtime.NumCPU(), Goroutines: runtime.NumGoroutine(), HeapAllocBytes: ms.HeapAlloc,
		PID: os.Getpid(), StartedAt: started.UTC(), UptimeSeconds: time.Since(started).Seconds()}

	if d.Health != nil {
		for _, p := range []Probe{ProbeStartup, ProbeLiveness, ProbeReadiness} {
			rep := d.Health.Evaluate(ctx, p)
			b.Probes = append(b.Probes, rep)
			b.Findings = append(b.Findings, probeFindings(rep, b.TraceID)...)
		}
	}
	b.Metrics = d.Metrics.Snapshot()
	b.Findings = append(b.Findings, metricFindings(b.Metrics)...)
	if d.Config != nil {
		b.Config = RedactMap(d.Config())
	}
	for _, c := range d.Checks {
		if c.Run == nil {
			continue
		}
		for _, f := range c.Run(ctx) {
			f.Message, f.Hint = RedactString(f.Message), RedactString(f.Hint)
			if f.Origin == "" {
				f.Origin = OriginRuntime
			}
			b.Findings = append(b.Findings, f)
		}
	}
	for _, f := range b.Findings {
		switch {
		case f.Severity == SeverityError:
			b.Status = StatusFail
		case f.Severity == SeverityWarn && b.Status == StatusPass:
			b.Status = StatusWarn
		}
	}
	return b
}

// Handler serves a fresh bundle as JSON. Mount it on an admin or loopback
// listener (PathDiagnostics): it is redacted but still operational detail.
func (d *Diagnostics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = d.Collect(r.Context()).WriteJSON(w)
	})
}

func readBuildInfo() BuildInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return BuildInfo{}
	}
	out := BuildInfo{Module: bi.Main.Path, Version: bi.Main.Version}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			out.VCSRevision = s.Value
		case "vcs.time":
			out.VCSTime = s.Value
		case "vcs.modified":
			out.VCSModified = s.Value == "true"
		}
	}
	return out
}
