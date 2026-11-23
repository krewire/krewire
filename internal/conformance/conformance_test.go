package conformance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWorkloadConformance is the end-to-end qualification gate. It scaffolds,
// builds, runs, probes and shuts down every workload fixture from a clean
// temp dir and fails on any regression against baseline.json.
//
//	KREWIRE_CONFORMANCE_WORKLOADS=site,book  restrict the run
//	KREWIRE_CONFORMANCE_OUT=dir              write report.json + summary.md
//	KREWIRE_CONFORMANCE_UPDATE_BASELINE=1    rewrite baseline.json from this run
func TestWorkloadConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end workload conformance skipped in -short mode")
	}
	env, err := Discover(".")
	if err != nil {
		t.Fatal(err)
	}
	opts := Options{Timeout: 5 * time.Minute, TempDir: t.TempDir()}
	if v := os.Getenv("KREWIRE_CONFORMANCE_WORKLOADS"); v != "" {
		opts.Workloads = strings.Split(v, ",")
	}
	rep, err := Run(context.Background(), env, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range rep.Workloads {
		checkInvariants(t, w)
	}
	if os.Getenv("KREWIRE_CONFORMANCE_UPDATE_BASELINE") == "1" {
		if len(opts.Workloads) != 0 {
			t.Fatal("refusing to update baseline from a filtered run")
		}
		if err := WriteBaseline("baseline.json", BaselineFrom(rep)); err != nil {
			t.Fatal(err)
		}
	}
	base, err := ReadBaseline("baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	regressions, improvements := Compare(rep, base)
	summary := Summary(rep, regressions, improvements)
	if out := os.Getenv("KREWIRE_CONFORMANCE_OUT"); out != "" {
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := rep.WriteJSON(filepath.Join(out, "report.json")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "summary.md"), []byte(summary), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("\n" + summary)
	for _, r := range regressions {
		t.Errorf("regression: %s", r)
	}
}

// checkInvariants guards against fake passes in the evidence itself.
func checkInvariants(t *testing.T, w WorkloadResult) {
	t.Helper()
	if len(w.Gates) == 0 {
		t.Errorf("%s: no gates recorded", w.Workload)
	}
	if w.Status == StatusPassed && (len(w.Exclusions) > 0 || !w.EndToEnd) {
		t.Errorf("%s: passed with exclusions", w.Workload)
	}
	if w.DeclaredMaturity == "" {
		t.Errorf("%s: missing from docs/workloads/maturity-matrix.md", w.Workload)
	}
	for _, g := range w.Gates {
		if g.EvidenceHash == "" || !strings.HasPrefix(g.GateID, w.Workload+".") {
			t.Errorf("%s: malformed gate %+v", w.Workload, g.GateID)
		}
	}
}
