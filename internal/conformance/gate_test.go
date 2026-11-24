package conformance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/model"
)

func newRun(name string) *run {
	return &run{env: &Env{Commit: "abc"}, res: &WorkloadResult{Workload: name, DeclaredMaturity: "Stable"}}
}

func ok() (string, error)   { return "fine", nil }
func fail() (string, error) { return "boom", errors.New("exit status 1") }

func TestGateBlocksAfterRequiredFailure(t *testing.T) {
	r := newRun("site")
	r.gate("build", gateBuild, "", "go build", fail)
	ran := false
	r.gate("start", gateTest, "", "serve", func() (string, error) { ran = true; return "", nil })
	if ran {
		t.Fatal("gate after a failed required gate must not execute")
	}
	if got := r.res.Gates[1].Status; got != model.GateBlocked {
		t.Fatalf("status = %s, want BLOCKED", got)
	}
	if r.res.Gates[0].EvidenceHash != r.res.Gates[0].ComputeHash("abc") {
		t.Fatal("evidence hash must bind gate output to the commit")
	}
}

func TestAssertDoesNotBlock(t *testing.T) {
	r := newRun("site")
	r.assert("probe", gateTest, "", "GET /", fail)
	r.gate("shutdown", gateTest, "", "SIGINT", ok)
	if r.res.Gates[1].Status != model.GatePassed {
		t.Fatal("a failed probe must not block shutdown")
	}
}

func TestClassifyNeverFakesAPass(t *testing.T) {
	cases := []struct {
		name     string
		gates    []model.VerificationStatus
		excluded bool
		status   Status
		e2e      bool
		maturity string
	}{
		{"all pass", []model.VerificationStatus{model.GatePassed}, false, StatusPassed, true, "Usable"},
		{"excluded step", []model.VerificationStatus{model.GatePassed}, true, StatusPartial, false, "Experimental"},
		{"failure", []model.VerificationStatus{model.GatePassed, model.GateFailed}, false, StatusFailed, false, "Experimental"},
		{"blocked", []model.VerificationStatus{model.GateBlocked}, false, StatusFailed, false, "Experimental"},
		{"no gates", nil, false, StatusFailed, false, "Experimental"},
	}
	for _, c := range cases {
		w := WorkloadResult{Workload: "x", DeclaredMaturity: "Stable"}
		for _, s := range c.gates {
			w.Gates = append(w.Gates, model.VerificationEvidence{Status: s, Required: true})
		}
		if c.excluded {
			w.Exclusions = []Exclusion{{Step: "plan"}}
		}
		classify(&w)
		if w.Status != c.status || w.EndToEnd != c.e2e || w.EvidencedMaturity != c.maturity {
			t.Errorf("%s: got %s/%v/%s", c.name, w.Status, w.EndToEnd, w.EvidencedMaturity)
		}
		if len(w.Findings) == 0 {
			t.Errorf("%s: Stable claim without Stable evidence must produce a finding", c.name)
		}
	}
}

func TestCompareDetectsRegressions(t *testing.T) {
	rep := &Report{Workloads: []WorkloadResult{{
		Workload: "book", Status: StatusFailed,
		Gates: []model.VerificationEvidence{
			{GateID: "book.build", Status: model.GatePassed},
			{GateID: "book.start", Status: model.GateFailed, Output: "bind: in use"},
			{GateID: "book.new", Status: model.GatePassed},
		},
	}}}
	base := Baseline{Workloads: map[string]BaselineWorkload{
		"book": {Status: StatusPassed, Passing: []string{"book.build", "book.start", "book.gone"}},
		"cli":  {Status: StatusPassed, Passing: []string{"cli.build"}},
	}}
	reg, imp := Compare(rep, base)
	want := []string{"book: status failed", "book.start: FAILED", "book.gone: gate no longer executed"}
	if len(reg) != len(want) {
		t.Fatalf("regressions = %q", reg)
	}
	for i, w := range want {
		if !strings.HasPrefix(reg[i], w) {
			t.Errorf("regression %d = %q, want prefix %q", i, reg[i], w)
		}
	}
	if len(imp) != 1 || !strings.HasPrefix(imp[0], "book.new") {
		t.Errorf("improvements = %q", imp)
	}
	if r2, _ := Compare(rep, BaselineFrom(rep)); len(r2) != 0 {
		t.Errorf("a report never regresses against its own baseline: %q", r2)
	}
}

func TestFinalizeSeparatesWorkloadAndReleaseVerdicts(t *testing.T) {
	rep := &Report{
		Workloads:  []WorkloadResult{{Workload: "cli", EndToEnd: true, Gates: []model.VerificationEvidence{{GateID: "cli.build", Status: model.GatePassed, Required: true}}}},
		AIWorkflow: Section{Status: StatusExcluded},
	}
	rep.Finalize()
	if !rep.WorkloadQualification.Verified {
		t.Fatal("workload qualification should pass")
	}
	if rep.ReleaseReadiness.Verified || !strings.Contains(rep.ReleaseReadiness.Summary, "ai_workflow excluded") {
		t.Fatalf("release readiness must stay false while the AI workflow is excluded: %+v", rep.ReleaseReadiness)
	}
}

func TestReportRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	rep := &Report{APIVersion: APIVersion, Kind: KindReport, Commit: "abc"}
	if err := rep.WriteJSON(path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadReport(path)
	if err != nil || got.Commit != "abc" {
		t.Fatalf("round trip: %v %+v", err, got)
	}
	if err := os.WriteFile(path, []byte(`{"api_version":"other/v1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReport(path); err == nil {
		t.Fatal("foreign report kinds must be rejected")
	}
}

// TestMaturityMatrixDeclaresEveryWorkload keeps the harness and the maturity
// matrix (the single source of declared status) in lockstep.
func TestMaturityMatrixDeclaresEveryWorkload(t *testing.T) {
	env, err := Discover(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range Workloads {
		if _, ok := maturityRank[env.Maturity[w]]; !ok {
			t.Errorf("workload %s: maturity %q not parsed from the matrix", w, env.Maturity[w])
		}
		if _, ok := fixtures[w]; !ok {
			t.Errorf("workload %s has no conformance fixture", w)
		}
	}
	base, err := ReadBaseline("baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range Workloads {
		if _, ok := base.Workloads[w]; !ok {
			t.Errorf("baseline.json lacks workload %s", w)
		}
	}
}

func TestAIWorkflowIsBaselined(t *testing.T) {
	gate := model.VerificationEvidence{GateID: "ai_workflow.verified.in-process", Status: model.GatePassed, Required: true}
	rep := &Report{AIWorkflow: Section{Status: StatusPassed, Gates: []model.VerificationEvidence{gate}}}
	base := BaselineFrom(rep)
	if bw := base.Workloads[aiWorkflowKey]; bw.Status != StatusPassed || len(bw.Passing) != 1 {
		t.Fatalf("ai_workflow not baselined: %+v", base)
	}
	gate.Status = model.GateFailed
	rep.AIWorkflow = Section{Status: StatusFailed, Gates: []model.VerificationEvidence{gate}}
	if reg, _ := Compare(rep, base); len(reg) != 2 {
		t.Fatalf("want status and gate regressions for ai_workflow, got %v", reg)
	}
}
