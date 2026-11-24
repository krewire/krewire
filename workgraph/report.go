package workgraph

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/krewire/krewire/model"
)

// TargetResult is the decision and outcome for one target.
type TargetResult struct {
	ID     string   `json:"id"`
	Task   string   `json:"task"`
	Node   string   `json:"node"`
	Kind   TaskKind `json:"kind"`
	Argv   []string `json:"argv"`
	Status Status   `json:"status"`
	// Reason explains the decision (not affected, cache hit, cache miss,
	// dependency failed, exit code, ...).
	Reason string `json:"reason"`
	// Cause is why the target's node is affected, when it is.
	Cause      *Cause     `json:"cause,omitempty"`
	Key        string     `json:"cache_key,omitempty"`
	Inputs     *KeyInputs `json:"inputs,omitempty"`
	ExitCode   int        `json:"exit_code"`
	DurationMs int64      `json:"duration_ms"`
	Output     string     `json:"output,omitempty"`
}

// Report is the machine-readable run report for CI and AI workflows.
type Report struct {
	APIVersion  string         `json:"api_version"`
	Kind        string         `json:"kind"`
	Mode        string         `json:"mode"`
	Base        string         `json:"base,omitempty"`
	Head        string         `json:"head,omitempty"`
	GraphDigest string         `json:"graph_digest"`
	GoVersion   string         `json:"go_version,omitempty"`
	Summary     map[Status]int `json:"summary"`
	Targets     []TargetResult `json:"targets"`
}

type selection struct {
	mode      string
	run       map[string]bool
	cause     map[string]Cause
	requested map[string]bool
}

func (s selection) why(id string, t *Target) string {
	if _, ok := s.cause[t.Node]; !ok {
		return "not affected"
	}
	if !s.requested[id] {
		return "task not requested"
	}
	return "not selected"
}

// selection picks requested targets on affected nodes plus the affected
// part of their dependency closure.
func (tg *TaskGraph) selection(ws *Workspace, opt RunOptions) selection {
	aff := opt.Affected
	if aff == nil {
		aff = ws.All()
	}
	s := selection{mode: aff.Mode, run: map[string]bool{}, cause: aff.index, requested: map[string]bool{}}
	var stack []string
	for _, id := range tg.order {
		t := tg.targets[id]
		if _, ok := aff.index[t.Node]; ok && (len(opt.Tasks) == 0 || slices.Contains(opt.Tasks, t.Task)) {
			s.requested[id] = true
			stack = append(stack, id)
		}
	}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if s.run[id] {
			continue
		}
		if _, ok := aff.index[tg.targets[id].Node]; !ok {
			continue // unaffected dependency: its base result still holds
		}
		s.run[id] = true
		stack = append(stack, tg.targets[id].Deps...)
	}
	return s
}

func newReport(ws *Workspace, opt RunOptions, mode string) *Report {
	r := &Report{APIVersion: APIVersion, Kind: KindReportDocument, Mode: mode,
		GraphDigest: ws.Graph.Digest(), Summary: map[Status]int{}}
	if opt.Affected != nil {
		r.Base, r.Head = opt.Affected.Base, opt.Affected.Head
	}
	if opt.Hasher != nil {
		r.GoVersion = opt.Hasher.goVersion
	}
	return r
}

func (r *Report) add(t TargetResult) {
	r.Targets = append(r.Targets, t)
	r.Summary[t.Status]++
}

// OK reports whether no selected target failed or was blocked.
func (r *Report) OK() bool { return r.Summary[StatusFailed] == 0 && r.Summary[StatusBlocked] == 0 }

// WriteJSON writes the report as indented JSON.
func (r *Report) WriteJSON(w io.Writer) error { return writeJSON(w, r) }

// WriteText writes a summary line and one line per target with its reason.
func (r *Report) WriteText(w io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s run: %d executed, %d restored, %d skipped, %d failed, %d blocked\n", r.Mode,
		r.Summary[StatusExecuted], r.Summary[StatusRestored], r.Summary[StatusSkipped],
		r.Summary[StatusFailed], r.Summary[StatusBlocked])
	for _, t := range r.Targets {
		fmt.Fprintf(&b, "  %-8s %s: %s\n", t.Status, t.ID, t.Reason)
		if t.Status == StatusFailed && t.Output != "" {
			for _, line := range strings.Split(strings.TrimRight(t.Output, "\n"), "\n") {
				fmt.Fprintf(&b, "           | %s\n", line)
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

var gateTypes = map[TaskKind]model.QualityGateType{
	TaskFormat: model.GateTypeFormat, TaskLint: model.GateTypeStatic,
	TaskTest: model.GateTypeTest, TaskBuild: model.GateTypeBuild,
}

var gateStatus = map[Status]model.VerificationStatus{
	StatusExecuted: model.GatePassed, StatusRestored: model.GatePassed,
	StatusFailed: model.GateFailed, StatusSkipped: model.GateSkipped, StatusBlocked: model.GateBlocked,
}

// Evidence converts the report into verification evidence for AI-native
// evidence reports (one entry per target, hashed against commit). Restored
// targets pass with the cache key recorded in the output, so a reviewer can
// tell a re-run from a restore.
func (r *Report) Evidence(commit string) []model.VerificationEvidence {
	out := make([]model.VerificationEvidence, 0, len(r.Targets))
	for _, t := range r.Targets {
		typ, ok := gateTypes[t.Kind]
		if !ok {
			typ = model.GateTypeCustom
		}
		e := model.VerificationEvidence{
			GateID: t.ID, Type: typ, Description: string(t.Status) + ": " + t.Reason,
			Command: strings.Join(t.Argv, " "), Required: true, Status: gateStatus[t.Status],
			ExitCode: t.ExitCode, Output: t.Output, DurationMs: t.DurationMs,
		}
		if t.Status == StatusRestored {
			e.Output = "restored from cache key " + t.Key + "\n" + t.Output
		}
		e.EvidenceHash = e.ComputeHash(commit)
		out = append(out, e)
	}
	return out
}
