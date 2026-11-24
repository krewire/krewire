package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/krewire/krewire/model"
)

// Validate checks the context's structural invariants.
func (c *ProjectContext) Validate() error {
	if err := c.Check(KindProjectContext); err != nil {
		return err
	}
	if err := requireText("module.path", c.Module.Path); err != nil {
		return err
	}
	if err := requireText("fingerprint", c.Fingerprint); err != nil {
		return err
	}
	seen := map[string]bool{}
	for i, g := range c.Gates {
		if g.ID == "" || len(g.Argv) == 0 {
			return Errorf(CodeInvalidDocument, "%s: gate needs id and argv", fieldf("quality_gates", i))
		}
		if seen[g.ID] {
			return Errorf(CodeInvalidDocument, "%s: duplicate gate id %q", fieldf("quality_gates", i), g.ID)
		}
		seen[g.ID] = true
	}
	return nil
}

// Validate checks the brief's structure and, when ctx is non-nil, that every
// required gate exists in the project context.
func (b *TaskBrief) Validate(ctx *ProjectContext) error {
	if err := b.Check(KindTaskBrief); err != nil {
		return err
	}
	for f, v := range map[string]string{"id": b.ID, "title": b.Title, "objective": b.Objective, "expected_outcome": b.ExpectedOutcome} {
		if err := requireText(f, v); err != nil {
			return err
		}
	}
	if len(b.Scope.Allowed) == 0 {
		return Errorf(CodeInvalidDocument, "scope.allowed must declare at least one path")
	}
	for i, p := range append(append([]string(nil), b.Scope.Allowed...), b.Scope.Prohibited...) {
		if cleanRel(p) == "" {
			return Errorf(CodeInvalidDocument, "scope path #%d %q is absolute or escapes the repository", i, p)
		}
	}
	switch b.Risk {
	case RiskLow, RiskMedium, RiskHigh:
	default:
		return Errorf(CodeInvalidDocument, "risk %q is not low|medium|high", b.Risk)
	}
	switch b.MaxAutonomy {
	case model.AutonomyAutonomous, model.AutonomyApprovalRequired, model.AutonomyEscalation:
	default:
		return Errorf(CodeInvalidDocument, "max_autonomy %q is not a model.AutonomyLevel", b.MaxAutonomy)
	}
	if b.Verification.Condition != ConditionAllRequiredPassed {
		return Errorf(CodeInvalidDocument, "verification.condition must be %q", ConditionAllRequiredPassed)
	}
	if len(b.Verification.Gates) == 0 {
		return Errorf(CodeInvalidDocument, "verification.gates must name at least one deterministic gate")
	}
	if b.Limits.TimeoutSec < 0 || b.Limits.MaxToolCalls < 0 || b.Limits.MaxRetries < 0 {
		return Errorf(CodeInvalidDocument, "limits must not be negative")
	}
	if ctx != nil {
		for _, id := range b.Verification.Gates {
			if _, ok := ctx.Gate(id); !ok {
				return Errorf(CodeInvalidDocument, "verification gate %q is not declared in the project context", id)
			}
		}
	}
	return nil
}

// Validate checks the plan against its brief: steps are unique, ordered
// dependencies resolve to earlier steps, and every target is in scope.
func (p *Plan) Validate(b *TaskBrief) error {
	if err := p.Check(KindPlan); err != nil {
		return err
	}
	if b != nil && p.TaskID != b.ID {
		return Errorf(CodeInvalidDocument, "plan task_id %q does not match brief %q", p.TaskID, b.ID)
	}
	if len(p.Steps) == 0 {
		return Errorf(CodeInvalidDocument, "plan has no steps")
	}
	seen := map[string]bool{}
	for i, s := range p.Steps {
		if s.ID == "" || seen[s.ID] {
			return Errorf(CodeInvalidDocument, "%s: step id is empty or duplicate", fieldf("steps", i))
		}
		for _, d := range s.DependsOn {
			if !seen[d] {
				return Errorf(CodeInvalidDocument, "%s: depends on %q, which is not an earlier step", fieldf("steps", i), d)
			}
		}
		seen[s.ID] = true
		if b != nil {
			for _, f := range s.TargetFiles {
				if !b.Scope.InScope(f) {
					return Errorf(CodeScopeViolation, "%s: target %q is outside the task scope", fieldf("steps", i), f)
				}
			}
		}
	}
	return nil
}

// ComputeDigest returns the SHA-256 of the plan's canonical JSON with the
// Digest field cleared. Approvals bind to this value.
func (p Plan) ComputeDigest() string {
	p.Digest = ""
	return digestJSON(p)
}

// Digest returns the SHA-256 of the call's canonical JSON with the content
// replaced by its hash, so approvals bind to the exact action.
func (t ToolCall) Digest() string {
	return digestJSON(t.Redacted())
}

// Redacted returns the call with Content replaced by ContentSHA256.
func (t ToolCall) Redacted() ToolCall {
	if t.Content != "" {
		sum := sha256.Sum256([]byte(t.Content))
		t.ContentSHA256 = hex.EncodeToString(sum[:])
		t.Content = ""
	}
	return t
}

func digestJSON(v any) string {
	data, _ := json.Marshal(v) // contract types always marshal
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Validate checks that the report keeps claims apart from evidence: the
// verified flag must follow model.EvaluateTaskVerification over the raw
// gates, and a claim may cite only a gate that actually passed.
func (r *EvidenceReport) Validate() error {
	if err := r.Check(KindEvidenceReport); err != nil {
		return err
	}
	if err := requireText("task_id", r.TaskID); err != nil {
		return err
	}
	verified, _ := model.EvaluateTaskVerification(r.Verification.Gates)
	if r.Verification.Verified != verified {
		return Errorf(CodeInvalidDocument, "verification.verified=%v contradicts the raw gate outcomes", r.Verification.Verified)
	}
	if r.Outcome == OutcomeVerified && !verified {
		return Errorf(CodeInvalidDocument, "outcome %q requires passing required gates", r.Outcome)
	}
	passed := map[string]bool{}
	for _, g := range r.Verification.Gates {
		passed[g.GateID] = g.Status == model.GatePassed
	}
	for i, c := range r.Claims {
		if c.VerifiedBy != "" && !passed[c.VerifiedBy] {
			return Errorf(CodeInvalidDocument, "%s cites gate %q, which did not pass", fieldf("claims", i), c.VerifiedBy)
		}
	}
	return nil
}

// Validate checks the catalog's central invariant: deterministic commands
// never require an engine and agentic commands always do.
func (c CommandCatalog) Validate() error {
	if err := c.Check(KindCommandCatalog); err != nil {
		return err
	}
	if c.Executable != Executable {
		return Errorf(CodeInvalidDocument, "executable %q is not the canonical %q", c.Executable, Executable)
	}
	seen := map[string]bool{}
	for i, cmd := range c.Commands {
		if seen[cmd.Path] {
			return Errorf(CodeInvalidDocument, "%s: duplicate command %q", fieldf("commands", i), cmd.Path)
		}
		seen[cmd.Path] = true
		if (cmd.Class == ClassAgentic) != cmd.RequiresEngine {
			return Errorf(CodeInvalidDocument, "%s: %q class %s contradicts requires_engine=%v", fieldf("commands", i), cmd.Path, cmd.Class, cmd.RequiresEngine)
		}
	}
	return nil
}
