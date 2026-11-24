package observe_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/krewire/krewire/observe"
)

// Example wires a service workload: identity-stamped redacting logs, trace
// propagation, standard HTTP metrics, probes, and diagnostics.
func Example() {
	id := observe.Identity{Service: "orders", Workload: string(observe.WorkloadService), Resource: "orders-api"}
	logger := observe.NewLogger(slog.Default(), id)
	profile, _ := observe.ProfileFor(observe.WorkloadService)

	health := observe.NewHealth()
	metrics := observe.NewRegistry()
	metrics.CollectRuntime()
	health.SetMetrics(metrics)
	health.Add(observe.Check{Name: "db", Func: func(context.Context) error { return nil }})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), "order read")
	})
	observe.Mount(mux, profile, health, metrics, &observe.Diagnostics{Identity: id, Workload: observe.WorkloadService, Health: health, Metrics: metrics})
	handler := observe.TraceHTTP(observe.InstrumentHTTP(metrics, mux))

	health.MarkStarted()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	fmt.Println(rec.Code, rec.Header().Get("traceparent") != "")
	// Output: 200 true
}

// ExampleDiagnostics_Collect shows the doctor flow a CLI runs: collect, print
// JSON, exit with the bundle's code.
func ExampleDiagnostics_Collect() {
	h := observe.NewHealth()
	d := &observe.Diagnostics{Workload: observe.WorkloadCLI, Health: h}
	b := d.Collect(context.Background())
	fmt.Println(b.Schema, b.Status, b.Findings[0].Code, b.ExitCode(false))
	// Output: krewire.diagnostics/v1 fail health.startup.incomplete 1
}
