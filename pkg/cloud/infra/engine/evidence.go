package engine

import (
	"encoding/json"
	"fmt"

	"github.com/krewire/krewire/pkg/model"
)

// EvidenceKind identifies the infra evidence document format.
const EvidenceKind = "infra.krewire.dev/evidence/v1"

// Evidence is the structured record that ties one plan, its approval, its
// apply result, and (optionally) the drift report it reconciled into a single
// task execution record for AI-native workflows. Every field is already
// redacted: plans and results never carry resolved secret values.
type Evidence struct {
	Kind        string       `json:"kind"`
	TaskID      string       `json:"taskId,omitempty"`
	Environment string       `json:"environment"`
	Principal   Principal    `json:"principal"`
	Plan        *Plan        `json:"plan,omitempty"`
	Approval    *Approval    `json:"approval,omitempty"`
	Result      *Result      `json:"result,omitempty"`
	Drift       *DriftReport `json:"drift,omitempty"`
}

// NewEvidence assembles an evidence record for one lifecycle run.
func (e *Engine) NewEvidence(taskID string, p *Plan, a *Approval, r *Result, d *DriftReport) Evidence {
	return Evidence{Kind: EvidenceKind, TaskID: taskID, Environment: e.Environment, Principal: e.Principal,
		Plan: p, Approval: a, Result: r, Drift: d}
}

// JSON renders the evidence document.
func (ev Evidence) JSON() ([]byte, error) { return json.MarshalIndent(ev, "", "  ") }

// Verification maps the evidence onto the shared verification-gate record
// (pkg/model) so infra runs appear beside build and test gates.
func (ev Evidence) Verification() model.VerificationEvidence {
	v := model.VerificationEvidence{
		GateID: "infra." + ev.Environment, Type: model.GateTypeCustom, Required: true,
		Description: "infrastructure lifecycle (plan/approve/apply)", Status: model.GatePlanned,
	}
	if ev.Plan != nil {
		v.Command = fmt.Sprintf("krew infra apply --env %s --plan %s", ev.Environment, ev.Plan.Digest)
		s := ev.Plan.Summary
		v.Output = fmt.Sprintf("plan %s: %d create, %d update, %d replace, %d delete, %d unchanged",
			ev.Plan.Digest, s.Create, s.Update, s.Replace, s.Delete, s.NoOp)
	}
	switch {
	case ev.Result == nil:
	case ev.Result.Succeeded:
		v.Status = model.GatePassed
	default:
		v.Status, v.ExitCode = model.GateFailed, 1
		for _, a := range ev.Result.Actions {
			if a.Status == StatusFailed {
				v.Output += fmt.Sprintf("; failed %s %s: %s", a.Op, a.Address, a.Error)
			}
		}
	}
	return v
}
