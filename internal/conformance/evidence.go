// Package conformance is the end-to-end workload qualification harness
// (krewire/krewire#141).
//
// For each of the eight workloads (observe.Workloads: app, service, gateway,
// worker, site, book, infra, cli) it scaffolds a project from the real starter templates into
// a clean temporary directory, builds it as a separate Go module against this
// checkout, runs the resulting binary, probes representative behaviour, and
// shuts it down. Every step is recorded as a model.VerificationEvidence gate,
// so a Report can be embedded into the AI-native toolchain EvidenceReport
// (pkg/toolchain/contract, #131/#140) without conversion.
//
// Nothing here fakes a pass: a step the core repository cannot exercise yet
// is recorded as an Exclusion with a reason, and the workload is reported as
// partial instead of passed.
package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/model"
)

// APIVersion versions the report schema. It mirrors the
// "toolchain.krewire.dev/v1" convention used by the toolchain contracts.
const APIVersion = "conformance.krewire.dev/v1"

// KindReport is the report kind.
const KindReport = "QualificationReport"

// Status is a workload or section outcome.
type Status string

// Outcomes. Partial means every executed gate passed but declared lifecycle
// steps were excluded, so the workload is not proven end to end.
const (
	StatusPassed   Status = "passed"
	StatusPartial  Status = "partial"
	StatusFailed   Status = "failed"
	StatusExcluded Status = "excluded"
)

// Exclusion is a lifecycle step that was deliberately not executed.
type Exclusion struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
	// Tracking names the issue or ADR that owns closing the gap.
	Tracking string `json:"tracking,omitempty"`
}

// Artifact is a file produced by a build step, fingerprinted for
// reproducibility checks.
type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// WorkloadResult is the qualification evidence for one workload.
type WorkloadResult struct {
	Workload string `json:"workload"`
	// DeclaredMaturity is read from docs/workloads/maturity-matrix.md.
	DeclaredMaturity string `json:"declared_maturity"`
	// EvidencedMaturity is the highest level the automated evidence alone
	// supports (never Stable: production use, compatibility and benchmark
	// baselines are not provable by this harness).
	EvidencedMaturity string                       `json:"evidenced_maturity"`
	Status            Status                       `json:"status"`
	EndToEnd          bool                         `json:"end_to_end"`
	Fixture           string                       `json:"fixture"`
	DurationMs        int64                        `json:"duration_ms"`
	Gates             []model.VerificationEvidence `json:"gates"`
	Artifacts         []Artifact                   `json:"artifacts,omitempty"`
	Exclusions        []Exclusion                  `json:"exclusions,omitempty"`
	Findings          []string                     `json:"findings,omitempty"`
}

// Section is a separately reported qualification area (e.g. the integrated
// AI-native workflow), so workload and AI qualification stay distinct.
type Section struct {
	Status     Status                       `json:"status"`
	Gates      []model.VerificationEvidence `json:"gates,omitempty"`
	Exclusions []Exclusion                  `json:"exclusions,omitempty"`
}

// Toolchain records the environment the evidence was produced with.
type Toolchain struct {
	Go     string `json:"go"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	CGO    string `json:"cgo_enabled"`
}

// Verdict is a combined pass/fail decision with its rationale.
type Verdict struct {
	Verified bool   `json:"verified"`
	Summary  string `json:"summary"`
}

// Report is the machine-readable qualification evidence.
type Report struct {
	APIVersion string           `json:"api_version"`
	Kind       string           `json:"kind"`
	Commit     string           `json:"commit"`
	Dirty      bool             `json:"dirty"`
	Module     string           `json:"module"`
	Version    string           `json:"version"`
	Toolchain  Toolchain        `json:"toolchain"`
	StartedAt  string           `json:"started_at"`
	DurationMs int64            `json:"duration_ms"`
	Workloads  []WorkloadResult `json:"workloads"`
	AIWorkflow Section          `json:"ai_workflow"`
	// WorkloadQualification evaluates every required workload gate.
	WorkloadQualification Verdict `json:"workload_qualification"`
	// ReleaseReadiness combines workload and AI-native qualification.
	ReleaseReadiness Verdict `json:"release_readiness"`
}

// Finalize computes the derived verdicts. Call it after all results are in.
func (r *Report) Finalize() {
	sort.Slice(r.Workloads, func(i, j int) bool { return order(r.Workloads[i].Workload) < order(r.Workloads[j].Workload) })
	var all []model.VerificationEvidence
	for _, w := range r.Workloads {
		all = append(all, w.Gates...)
	}
	ok, summary := model.EvaluateTaskVerification(all)
	r.WorkloadQualification = Verdict{Verified: ok, Summary: summary}

	var gaps []string
	for _, w := range r.Workloads {
		if !w.EndToEnd {
			gaps = append(gaps, w.Workload)
		}
	}
	ready := ok && len(gaps) == 0 && r.AIWorkflow.Status == StatusPassed
	var why []string
	if !ok {
		why = append(why, "workload gates failed")
	}
	if len(gaps) > 0 {
		why = append(why, "not end-to-end: "+strings.Join(gaps, ", "))
	}
	if r.AIWorkflow.Status != StatusPassed {
		why = append(why, "ai_workflow "+string(r.AIWorkflow.Status))
	}
	if ready {
		r.ReleaseReadiness = Verdict{Verified: true, Summary: "all workloads end-to-end and AI-native workflow qualified"}
	} else {
		r.ReleaseReadiness = Verdict{Summary: "not release-ready: " + strings.Join(why, "; ")}
	}
}

// Workload returns the result for name, if present.
func (r *Report) Workload(name string) (WorkloadResult, bool) {
	for _, w := range r.Workloads {
		if w.Workload == name {
			return w, true
		}
	}
	return WorkloadResult{}, false
}

// WriteJSON writes the report as indented JSON.
func (r *Report) WriteJSON(path string) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// ReadReport loads a report written by WriteJSON.
func ReadReport(path string) (*Report, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("conformance: decode %s: %w", path, err)
	}
	if r.APIVersion != APIVersion || r.Kind != KindReport {
		return nil, fmt.Errorf("conformance: %s is %s/%s, want %s/%s", path, r.APIVersion, r.Kind, APIVersion, KindReport)
	}
	return &r, nil
}
