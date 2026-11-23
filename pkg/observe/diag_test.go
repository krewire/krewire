package observe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticsBundle(t *testing.T) {
	h := NewHealth()
	h.Add(Check{Name: "db", Func: func(context.Context) error { return errors.New("dial mysql://u:pw@h/x") }},
		Check{Name: "search", Optional: true, Func: func(context.Context) error { return errors.New("slow") }})
	reg := NewRegistry()
	reg.SetMaxSeries(1)
	ObserveJob(reg, "mail", OutcomeDeadLetter, time.Millisecond)
	ObserveJob(reg, "sms", OutcomeSuccess, time.Millisecond)
	stdCounter(reg, MetricRetries).Add(3, "charge")
	stdGauge(reg, MetricBreakerState).Set(2, "payments")
	d := &Diagnostics{Identity: Identity{Service: "shop"}, Workload: WorkloadWorker, Health: h, Metrics: reg,
		Config: func() map[string]string { return map[string]string{"DB_PASSWORD": "x", "QUEUE": "redis://u:p@q"} },
		Checks: []DoctorCheck{{Name: "nil"}, {Name: "agent", Run: func(context.Context) []Finding {
			return []Finding{{Code: "agent.claim", Severity: SeverityInfo, Origin: OriginAgent, Message: "tests pass token=abc"},
				{Code: "cfg", Severity: SeverityInfo, Message: "ok"}}
		}}}}
	ctx, sc := StartSpan(WithIDs(context.Background(), IDs{TaskID: "t-1"}))

	b := d.Collect(ctx)
	if b.Schema != BundleSchema || b.Status != StatusFail || b.Identity.Workload != "worker" || b.Profile == nil || b.Profile.Mode != ModeHeartbeat {
		t.Fatalf("bundle header %+v", b)
	}
	if b.Correlation.TaskID != "t-1" || b.TraceID != sc.TraceID.String() || b.Runtime.GoVersion == "" || len(b.Probes) != 3 {
		t.Fatalf("bundle body %+v", b)
	}
	codes := map[string]Finding{}
	for _, f := range b.Findings {
		codes[f.Code] = f
	}
	for _, c := range []string{"health.startup.incomplete", "health.readiness.db", "health.readiness.search",
		"metrics.cardinality_overflow", "worker.dead_letters", "resilience.retries", "resilience.breaker_open", "agent.claim", "cfg"} {
		if _, ok := codes[c]; !ok {
			t.Errorf("missing finding %s in %+v", c, b.Findings)
		}
	}
	if codes["agent.claim"].Origin != OriginAgent || codes["cfg"].Origin != OriginRuntime || codes["health.readiness.db"].TraceID != sc.TraceID.String() {
		t.Error("origin/trace attribution lost")
	}
	if b.ExitCode(false) != 1 {
		t.Error("errors must fail CI")
	}
	var buf bytes.Buffer
	if err := b.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"u:pw", "u:p@", `"x"`, "abc"} {
		if strings.Contains(buf.String(), leak) {
			t.Errorf("bundle leaked %q", leak)
		}
	}

	h2 := NewHealth()
	h2.MarkStarted()
	h2.MarkDraining()
	b2 := (&Diagnostics{Health: h2, StartedAt: time.Now()}).Collect(context.Background())
	if b2.Status != StatusWarn || b2.ExitCode(false) != 0 || b2.ExitCode(true) != 1 || b2.Findings[0].Code != "health.readiness.draining" {
		t.Fatalf("draining bundle %+v", b2)
	}
	rec := httptest.NewRecorder()
	(&Diagnostics{}).Handler().ServeHTTP(rec, httptest.NewRequest("GET", PathDiagnostics, nil))
	var got Bundle
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Status != StatusPass || got.Findings == nil {
		t.Fatalf("handler %v %+v", err, got)
	}
}

func TestProfiles(t *testing.T) {
	if len(Workloads) != len(profiles) {
		t.Fatal("Workloads and profiles diverge")
	}
	for _, w := range Workloads {
		p, ok := ProfileFor(w)
		if !ok || p.Workload != w || p.Mode == "" || p.Correlation == nil {
			t.Errorf("%s: %+v", w, p)
		}
		for _, m := range p.Metrics {
			if _, ok := StandardDesc(m); !ok {
				t.Errorf("%s lists non-standard metric %s", w, m)
			}
		}
		if p.Mode == ModeHTTP && (p.Probes != DefaultPaths || p.MetricsPath != PathMetrics) {
			t.Errorf("%s: HTTP workloads use the default paths", w)
		}
	}
	p, _ := ProfileFor(WorkloadService)
	p.Metrics[0] = "mutated"
	if q, _ := ProfileFor(WorkloadService); q.Metrics[0] == "mutated" {
		t.Error("ProfileFor must return a copy")
	}
	if _, ok := ProfileFor("unknown"); ok {
		t.Error("unknown workload")
	}
}
