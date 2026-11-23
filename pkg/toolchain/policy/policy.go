// Package policy authorizes engine-requested tool calls against the host
// policy, the task brief's scope, and HMAC-authenticated approval records.
// It is the one policy model shared by deterministic and agentic workflows
// (#131 §6): engines propose, the toolchain decides.
package policy

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/sec"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// PlanScope is the approval scope bound to an exact plan digest.
func PlanScope(taskID, planDigest string) string { return "task:" + taskID + ":plan:" + planDigest }

// CallScope is the approval scope bound to an exact tool call.
func CallScope(taskID string, call contract.ToolCall) string {
	return "task:" + taskID + ":call:" + call.Digest()
}

// Sign issues an approval record for scope, authenticated with secret. The
// secret stays with the host (or the approving control surface, e.g. Dash)
// and never appears in any contract document.
func Sign(approverID, scope, secret string, ttl time.Duration) (contract.ApprovalRecord, error) {
	s, err := sec.NewApprovalSignal(approverID, scope, secret, ttl)
	if err != nil {
		return contract.ApprovalRecord{}, contract.Errorf(contract.CodeInvalidParams, "%v", err)
	}
	return contract.ApprovalRecord{TypeMeta: contract.Meta(contract.KindApprovalRecord), ApproverID: s.ApproverID,
		Scope: s.Scope, ApprovedAt: s.ApprovedAt, ValidUntil: s.ValidUntil, Token: s.Token}, nil
}

// Authorizer evaluates tool calls for one task.
type Authorizer struct {
	Root    string
	Policy  contract.Policy
	Task    *contract.TaskBrief
	Context *contract.ProjectContext
	// Secret verifies approval tokens; empty means no approval can verify.
	Secret string
}

// VerifyApproval reports whether rec is authentic, unexpired, bound to
// scope, and issued by a permitted approver.
func (a *Authorizer) VerifyApproval(rec contract.ApprovalRecord, scope string) bool {
	if rec.Scope != scope || a.Secret == "" {
		return false
	}
	if allowed := a.Task.Approval.Approvers; len(allowed) > 0 && !slices.Contains(allowed, rec.ApproverID) {
		return false
	}
	sig := sec.ApprovalSignal{ApproverID: rec.ApproverID, ApprovedAt: rec.ApprovedAt, Token: rec.Token,
		Scope: rec.Scope, ValidUntil: rec.ValidUntil}
	return sig.VerifyToken(a.Secret)
}

// FindApproval returns the first verified record for scope.
func (a *Authorizer) FindApproval(recs []contract.ApprovalRecord, scope string) (contract.ApprovalRecord, bool) {
	for _, r := range recs {
		if a.VerifyApproval(r, scope) {
			r.Verified = true
			return r, true
		}
	}
	return contract.ApprovalRecord{}, false
}

func deny(code contract.Code, format string, args ...any) contract.Decision {
	return contract.Decision{Code: code.String(), Reason: fmt.Sprintf(format, args...)}
}

// Authorize decides one tool call. approvals are records collected so far;
// a call needing approval without a verified record is denied with
// approval_required and the scope the approval must carry.
func (a *Authorizer) Authorize(call contract.ToolCall, approvals []contract.ApprovalRecord) contract.Decision {
	cat := call.Category()
	switch call.Tool {
	case contract.ToolWriteFile, contract.ToolDeleteFile:
		if call.Tool == contract.ToolWriteFile && !a.Policy.AllowFileWrite {
			return deny(contract.CodePolicyDenied, "file writes are disabled by policy")
		}
		if !a.Task.Scope.InScope(call.Path) {
			return deny(contract.CodeScopeViolation, "%q is outside the task scope or prohibited", call.Path)
		}
		if _, err := sec.SafePath(a.Root, call.Path); err != nil {
			return deny(contract.CodeScopeViolation, "%q escapes the workspace: %v", call.Path, err)
		}
	case contract.ToolRunCommand:
		if len(call.Argv) == 0 {
			return deny(contract.CodePolicyDenied, "run_command needs argv")
		}
		if !a.isGate(call.Argv) && !a.Task.Scope.AllowsCommand(call.Argv) {
			return deny(contract.CodeScopeViolation, "command %q is neither a quality gate nor allowed by the task", strings.Join(call.Argv, " "))
		}
		guard := sec.AgentPolicy{DenyDestructive: a.Policy.DenyDestructive}
		if err := sec.AuthorizeAgentAction(guard, sec.ActionRunCommand, strings.Join(call.Argv, " ")); err != nil {
			return deny(contract.CodePolicyDenied, "%v", err)
		}
	case contract.ToolFetch:
		if !a.Policy.AllowNetwork || !a.Task.Limits.AllowNetwork {
			return deny(contract.CodePolicyDenied, "network access is disabled by policy or task")
		}
	default:
		return deny(contract.CodePolicyDenied, "unknown tool %q", call.Tool)
	}
	if a.needsApproval(cat) {
		scope := CallScope(a.Task.ID, call)
		if _, ok := a.FindApproval(approvals, scope); !ok {
			d := deny(contract.CodeApprovalRequired, "%s requires a verified approval", cat)
			d.ApprovalScope = scope
			return d
		}
		return contract.Decision{Allowed: true, Reason: fmt.Sprintf("%s approved", cat)}
	}
	return contract.Decision{Allowed: true, Reason: fmt.Sprintf("%s permitted by policy and task scope", cat)}
}

func (a *Authorizer) needsApproval(c model.ActionCategory) bool {
	return slices.Contains(a.Policy.ApprovalRequired, c) || a.Task.Approval.NeedsApproval(c)
}

// isGate reports whether argv is exactly a declared quality gate.
func (a *Authorizer) isGate(argv []string) bool {
	if a.Context == nil {
		return false
	}
	for _, g := range a.Context.Gates {
		if slices.Equal(g.Argv, argv) {
			return true
		}
	}
	return false
}

// Redact removes credentials from s (via pkg/sec), strips any of the given
// literal secret values, and truncates to max bytes (max <= 0: no limit).
func Redact(s string, max int, secrets ...string) string {
	for _, v := range secrets {
		if len(v) >= 4 {
			s = strings.ReplaceAll(s, v, "[REDACTED_SECRET]")
		}
	}
	s = sec.RedactSensitiveOutput(s)
	if max > 0 && len(s) > max {
		s = s[:max] + "\n[truncated]"
	}
	return s
}
