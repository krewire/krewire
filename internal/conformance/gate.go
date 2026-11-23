package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/model"
)

// Baseline is the committed qualification floor (baseline.json). The
// regression gate fails when a workload loses a gate that passed when the
// baseline was recorded, or its status drops. Improvements never fail; they
// are reported so the baseline can be raised deliberately.
type Baseline struct {
	Workloads map[string]BaselineWorkload `json:"workloads"`
}

// BaselineWorkload is one workload's floor.
type BaselineWorkload struct {
	Status Status `json:"status"`
	// Passing lists gate IDs that must stay PASSED.
	Passing []string `json:"passing"`
}

var statusRank = map[Status]int{StatusFailed: 0, StatusExcluded: 0, StatusPartial: 1, StatusPassed: 2}

// BaselineFrom derives a baseline from a report.
func BaselineFrom(r *Report) Baseline {
	b := Baseline{Workloads: map[string]BaselineWorkload{}}
	for _, w := range r.Workloads {
		bw := BaselineWorkload{Status: w.Status, Passing: []string{}}
		for _, g := range w.Gates {
			if g.Status == model.GatePassed {
				bw.Passing = append(bw.Passing, g.GateID)
			}
		}
		sort.Strings(bw.Passing)
		b.Workloads[w.Workload] = bw
	}
	return b
}

// ReadBaseline loads baseline.json.
func ReadBaseline(path string) (Baseline, error) {
	var b Baseline
	raw, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, fmt.Errorf("conformance: decode %s: %w", path, err)
	}
	return b, nil
}

// WriteBaseline writes b as indented JSON.
func WriteBaseline(path string, b Baseline) error {
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// Compare returns regressions (gate-failing) and improvements (informational)
// of r against b. Only workloads present in r are compared, so a filtered
// run checks just what it executed.
func Compare(r *Report, b Baseline) (regressions, improvements []string) {
	for _, w := range r.Workloads {
		bw, ok := b.Workloads[w.Workload]
		if !ok {
			improvements = append(improvements, w.Workload+": not in baseline")
			continue
		}
		if statusRank[w.Status] < statusRank[bw.Status] {
			regressions = append(regressions, fmt.Sprintf("%s: status %s, baseline %s", w.Workload, w.Status, bw.Status))
		}
		now := map[string]model.VerificationEvidence{}
		for _, g := range w.Gates {
			now[g.GateID] = g
		}
		was := map[string]bool{}
		for _, id := range bw.Passing {
			was[id] = true
			g, ok := now[id]
			switch {
			case !ok:
				regressions = append(regressions, id+": gate no longer executed")
			case g.Status != model.GatePassed:
				regressions = append(regressions, fmt.Sprintf("%s: %s (%s)", id, g.Status, headline(g.Output)))
			}
		}
		for _, g := range w.Gates {
			if g.Status == model.GatePassed && !was[g.GateID] {
				improvements = append(improvements, g.GateID+": newly passing")
			}
		}
	}
	return regressions, improvements
}

// Summary renders a human-readable qualification summary in Markdown.
func Summary(r *Report, regressions, improvements []string) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# Workload conformance report\n\n")
	fmt.Fprintf(&s, "- Commit: `%s` (dirty: %v)\n- Version: %s\n- Toolchain: %s %s/%s CGO_ENABLED=%s\n- Duration: %dms\n\n",
		r.Commit, r.Dirty, r.Version, r.Toolchain.Go, r.Toolchain.GOOS, r.Toolchain.GOARCH, r.Toolchain.CGO, r.DurationMs)
	s.WriteString("| Workload | Declared | Evidenced | Status | End-to-end | Gates passed | Exclusions |\n|---|---|---|---|---|---|---|\n")
	for _, w := range r.Workloads {
		passed := 0
		for _, g := range w.Gates {
			if g.Status == model.GatePassed {
				passed++
			}
		}
		var ex []string
		for _, e := range w.Exclusions {
			ex = append(ex, e.Step)
		}
		fmt.Fprintf(&s, "| `%s` | %s | %s | **%s** | %v | %d/%d | %s |\n", w.Workload, w.DeclaredMaturity, w.EvidencedMaturity,
			w.Status, w.EndToEnd, passed, len(w.Gates), strings.Join(ex, ", "))
	}
	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&s, "\n## %s\n\n", title)
		for _, i := range items {
			fmt.Fprintf(&s, "- %s\n", i)
		}
	}
	var failing, findings []string
	for _, w := range r.Workloads {
		for _, g := range w.Gates {
			if g.Status != model.GatePassed {
				failing = append(failing, fmt.Sprintf("`%s` %s: %s", g.GateID, g.Status, headline(g.Output)))
			}
		}
		for _, e := range w.Exclusions {
			findings = append(findings, fmt.Sprintf("`%s` excluded %s: %s (%s)", w.Workload, e.Step, e.Reason, e.Tracking))
		}
		for _, f := range w.Findings {
			findings = append(findings, fmt.Sprintf("`%s`: %s", w.Workload, f))
		}
	}
	section("Failing gates", failing)
	section("Exclusions and findings", findings)
	fmt.Fprintf(&s, "\n## AI-native workflow\n\n- Status: **%s**\n", r.AIWorkflow.Status)
	for _, e := range r.AIWorkflow.Exclusions {
		fmt.Fprintf(&s, "- %s: %s (%s)\n", e.Step, e.Reason, e.Tracking)
	}
	fmt.Fprintf(&s, "\n## Verdicts\n\n- Workload qualification: %v — %s\n- Release readiness: %v — %s\n",
		r.WorkloadQualification.Verified, r.WorkloadQualification.Summary, r.ReleaseReadiness.Verified, r.ReleaseReadiness.Summary)
	section("Regressions against baseline", regressions)
	section("Improvements over baseline", improvements)
	return s.String()
}

// headline returns the first non-empty line of gate output, which carries
// the diagnostic; the trailing line is usually just "exit status N".
func headline(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return truncateTo(s, 160)
}
