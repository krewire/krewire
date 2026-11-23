package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/errs"
)

// Risk classifies the impact of an action.
type Risk string

// Risk levels, lowest first.
const (
	RiskNone Risk = "none"
	// RiskLow is an internal, non-destructive change.
	RiskLow Risk = "low"
	// RiskHigh is a change to an externally visible or cost-incurring kind.
	RiskHigh Risk = "high"
	// RiskDestructive deletes or replaces an object.
	RiskDestructive Risk = "destructive"
)

func (r Risk) rank() int {
	switch r {
	case RiskLow:
		return 1
	case RiskHigh:
		return 2
	case RiskDestructive:
		return 3
	}
	return 0
}

func kindRisk(s contract.KindSchema) Risk {
	if s.ExternallyVisible || s.CostIncurring {
		return RiskHigh
	}
	return RiskLow
}

// Policy decides when a plan needs explicit approval.
type Policy struct {
	// ApprovalThreshold is the lowest risk that requires approval; RiskHigh
	// when empty. Use RiskLow to require approval for every change.
	ApprovalThreshold Risk
}

// Principal identifies who drives the engine.
type Principal struct {
	ID string `json:"id"`
	// Agent marks an AI agent. Agent-driven plans with any change always
	// require approval from a different principal.
	Agent bool `json:"agent,omitempty"`
}

// Approval authorises one exact plan in one environment.
type Approval struct {
	PlanDigest  string    `json:"planDigest"`
	Environment string    `json:"environment"`
	Approver    string    `json:"approver"`
	ApprovedAt  time.Time `json:"approvedAt"`
	// ExpiresAt bounds the approval; zero means no expiry.
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Note      string    `json:"note,omitempty"`
}

// ApprovalScope is the scope string that binds an external approval signal
// (for example pkg/sec.ApprovalSignal) to this plan and environment.
func ApprovalScope(p *Plan) string {
	return "infra:" + string(p.Mode) + ":" + p.Environment + ":" + p.Digest
}

// Approval errors.
var (
	ErrApprovalRequired = errors.New("infra: plan requires approval")
	ErrApprovalMismatch = errors.New("infra: approval does not match plan")
	ErrApprovalExpired  = errors.New("infra: approval expired")
	ErrSelfApproval     = errors.New("infra: agent cannot approve its own plan")
)

// RequiresApproval reports whether applying p needs an Approval.
func (e *Engine) RequiresApproval(p *Plan) bool {
	if !p.HasChanges() {
		return false
	}
	if e.Principal.Agent {
		return true
	}
	threshold := e.Policy.ApprovalThreshold
	if threshold == "" {
		threshold = RiskHigh
	}
	return p.MaxRisk().rank() >= threshold.rank()
}

// authorize enforces the approval policy for p.
func (e *Engine) authorize(p *Plan, a *Approval) error {
	if !e.RequiresApproval(p) {
		return nil
	}
	if a == nil {
		return errs.WithHint(fmt.Errorf("%w: max risk %s in %s", ErrApprovalRequired, p.MaxRisk(), p.Environment),
			"review the plan and obtain an approval bound to digest "+p.Digest)
	}
	switch {
	case a.Approver == "":
		return fmt.Errorf("%w: approver is empty", ErrApprovalMismatch)
	case a.PlanDigest != p.Digest:
		return errs.WithHint(fmt.Errorf("%w: approved %s, plan is %s", ErrApprovalMismatch, a.PlanDigest, p.Digest),
			"the plan changed after approval; review and approve the new plan")
	case a.Environment != p.Environment:
		return fmt.Errorf("%w: approved for %q, plan targets %q", ErrApprovalMismatch, a.Environment, p.Environment)
	case !a.ExpiresAt.IsZero() && !e.now().Before(a.ExpiresAt):
		return fmt.Errorf("%w at %s", ErrApprovalExpired, a.ExpiresAt.Format(time.RFC3339))
	case e.Principal.Agent && a.Approver == e.Principal.ID:
		return ErrSelfApproval
	}
	return nil
}
