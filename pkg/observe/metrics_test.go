package observe

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	fn()
}

func TestRegistryValidation(t *testing.T) {
	r := NewRegistry()
	mustPanic(t, "bad name", func() { r.Gauge("Bad-Name", "", UnitCount) })
	mustPanic(t, "counter suffix", func() { r.Counter("requests", "", UnitCount) })
	mustPanic(t, "bad label", func() { r.Gauge("g", "", UnitCount, "le") })
	mustPanic(t, "kind", func() { r.Register(Desc{Name: "x", Kind: "summary"}) })
	mustPanic(t, "std", func() { r.Std("nope") })
	r.Gauge("dup", "h", UnitCount, "a")
	r.Gauge("dup", "h", UnitCount, "a")
	mustPanic(t, "conflict", func() { r.Gauge("dup", "other", UnitCount, "a") })
}

func TestNilRegistryIsNoop(t *testing.T) {
	var r *Registry
	r.SetMaxSeries(3)
	r.OnCollect(func() {})
	r.CollectRuntime()
	r.Counter("a_total", "", UnitCount).Inc()
	r.Gauge("g", "", UnitCount).Set(1)
	r.Gauge("g", "", UnitCount).Add(1)
	r.Histogram("h", "", UnitSeconds, nil).Observe(1)
	ObserveJob(r, "q", OutcomeSuccess, time.Second)
	ObserveCommand(r, "test", OutcomeSuccess, OriginCommand, time.Second)
	if r.Snapshot() != nil || r.ExportTo(context.Background(), nil) != nil {
		t.Fatal("nil registry must be empty")
	}
	h := http.NotFoundHandler()
	if InstrumentHTTP(nil, h) == nil {
		t.Fatal("nil registry must pass handler through")
	}
	rec := httptest.NewRecorder()
	MetricsHandler(nil).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled metrics endpoint = %d", rec.Code)
	}
}

func TestCardinalityBound(t *testing.T) {
	r := NewRegistry()
	r.SetMaxSeries(2)
	c := r.Counter("hits_total", "", UnitCount, "user")
	for _, u := range []string{"a", "b", "c", "d"} {
		c.Inc(u)
	}
	c.Inc("too", "many")
	c.Add(-5, "a")
	s := r.Snapshot()[0].Samples
	if len(s) != 3 || s[0].Labels["user"] != OverflowValue || s[0].Value != 3 {
		t.Fatalf("samples %+v", s)
	}
}

func TestPrometheusExposition(t *testing.T) {
	r := NewRegistry()
	r.CollectRuntime()
	r.Counter("jobs_total", "Jobs\ndone", UnitCount, "queue").Add(2, `q"1\`)
	r.Histogram("lat_seconds", "Latency", UnitSeconds, []float64{1, 0.1}, "op").Observe(0.5, "x")
	g := r.Gauge("temp", "T", UnitCount)
	g.Set(1.5)
	g.Add(1)
	srv := InstrumentHTTP(r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("x"))
	}))
	mux := http.NewServeMux()
	mux.Handle("GET /items/{id}", srv)
	mux.Handle("/", srv)
	for _, p := range []string{"/items/1", "/items/2"} {
		mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", p, nil))
	}
	srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("BREW", "/raw?q=1", nil))
	ObserveJob(r, "mail", OutcomeRetry, 20*time.Millisecond)
	ObserveCommand(r, "test", OutcomeFailure, OriginCommand, time.Second)

	rec := httptest.NewRecorder()
	MetricsHandler(r).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()
	if rec.Header().Get("Content-Type") != PrometheusContentType {
		t.Fatal("content type")
	}
	for _, want := range []string{
		"# HELP jobs_total Jobs\\ndone\n# TYPE jobs_total counter\n",
		`jobs_total{queue="q\"1\\"} 2`,
		`lat_seconds_bucket{op="x",le="0.1"} 0`,
		`lat_seconds_bucket{op="x",le="1"} 1`,
		`lat_seconds_bucket{op="x",le="+Inf"} 1`,
		`lat_seconds_sum{op="x"} 0.5`,
		`lat_seconds_count{op="x"} 1`,
		"temp 2.5\n",
		`krewire_http_server_requests_total{method="GET",route="GET /items/{id}",status_class="4xx"} 2`,
		`krewire_http_server_requests_total{method="OTHER",route="unmatched",status_class="4xx"} 1`,
		`krewire_worker_jobs_total{queue="mail",outcome="retry"} 1`,
		`krewire_toolchain_commands_total{command="test",outcome="failure",origin="command"} 1`,
		"krewire_go_goroutines ",
		"krewire_http_server_in_flight_requests 0",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "/items/1") || strings.Contains(body, "q=1") {
		t.Error("raw paths leaked into labels")
	}
	for _, v := range []float64{posInf(), -posInf(), nan()} {
		if formatFloat(v) == "" {
			t.Fatal("format")
		}
	}
}

func posInf() float64 { return 1 / zero() }
func nan() float64    { z := zero(); return z / z }
func zero() float64   { return 0 }

func TestExporters(t *testing.T) {
	r := NewRegistry()
	r.Counter("a_total", "", UnitCount).Inc()
	var got []Family
	err := r.ExportTo(context.Background(), nil, ExporterFunc(func(_ context.Context, f []Family) error {
		got = f
		return nil
	}), ExporterFunc(func(context.Context, []Family) error { return errors.New("down") }))
	if len(got) != 1 || got[0].Name != "a_total" || err == nil {
		t.Fatalf("got %v err %v", got, err)
	}
	for _, d := range StandardMetrics {
		if _, ok := StandardDesc(d.Name); !ok || d.Help == "" || d.Unit == "" {
			t.Errorf("standard metric %s incomplete", d.Name)
		}
		NewRegistry().Register(d)
	}
}
