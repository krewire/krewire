package contract

import (
	"path"
	"strings"

	"github.com/krewire/krewire/pkg/model"
)

// Risk classifies the blast radius of a task or plan step.
type Risk string

// Risk levels.
const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// TaskScope bounds what an engine may touch.
type TaskScope struct {
	// Allowed lists repository-relative paths (files or directory prefixes)
	// the task may modify. "." grants the whole repository.
	Allowed []string `json:"allowed"`
	// Prohibited lists paths that must not change even inside Allowed.
	Prohibited []string `json:"prohibited,omitempty"`
	// Commands lists argv prefixes the engine may ask the host to run, e.g.
	// ["go","test"]. Gate commands are always permitted to the host.
	Commands [][]string `json:"commands,omitempty"`
}

// ApprovalPolicy declares which actions need a human approval record.
type ApprovalPolicy struct {
	// Plan requires an approval record bound to the plan digest before any
	// mutating tool call runs.
	Plan bool `json:"plan"`
	// Actions lists action categories that need a per-call approval.
	Actions []model.ActionCategory `json:"actions,omitempty"`
	// Approvers optionally restricts who may approve.
	Approvers []string `json:"approvers,omitempty"`
}

// Verification declares the deterministic success condition of a task.
type Verification struct {
	// Gates lists quality-gate IDs from the ProjectContext that must pass.
	Gates []string `json:"gates"`
	// Condition is the success condition; "all_required_passed" is the only
	// value defined by v1.
	Condition string `json:"condition"`
}

// ConditionAllRequiredPassed is the v1 success condition.
const ConditionAllRequiredPassed = "all_required_passed"

// Limits bounds execution.
type Limits struct {
	TimeoutSec   int  `json:"timeout_sec,omitempty"`
	MaxToolCalls int  `json:"max_tool_calls,omitempty"`
	MaxRetries   int  `json:"max_retries,omitempty"`
	AllowNetwork bool `json:"allow_network,omitempty"`
}

// TaskBrief is the governed task contract shared by humans, deterministic
// commands, and engines.
type TaskBrief struct {
	TypeMeta
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Objective       string              `json:"objective"`
	ExpectedOutcome string              `json:"expected_outcome"`
	Repository      string              `json:"repository,omitempty"`
	Scope           TaskScope           `json:"scope"`
	DependsOn       []string            `json:"depends_on,omitempty"`
	Constraints     []string            `json:"constraints,omitempty"`
	Risk            Risk                `json:"risk"`
	MaxAutonomy     model.AutonomyLevel `json:"max_autonomy"`
	Approval        ApprovalPolicy      `json:"approval"`
	Verification    Verification        `json:"verification"`
	Deliverables    []string            `json:"deliverables,omitempty"`
	Limits          Limits              `json:"limits"`
}

// cleanRel normalizes a repository-relative path; it returns "" for paths
// that are absolute or escape the repository.
func cleanRel(p string) string {
	p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	if p == ".." || strings.HasPrefix(p, "../") || strings.HasPrefix(p, "/") {
		return ""
	}
	return p
}

// within reports whether p equals prefix or sits beneath it.
func within(p, prefix string) bool {
	return prefix == "." || p == prefix || strings.HasPrefix(p, prefix+"/")
}

// InScope reports whether a repository-relative path may be modified.
func (s TaskScope) InScope(p string) bool {
	c := cleanRel(p)
	if c == "" || c == "." {
		return false
	}
	for _, x := range s.Prohibited {
		if within(c, cleanRel(x)) {
			return false
		}
	}
	for _, a := range s.Allowed {
		if a := cleanRel(a); a != "" && within(c, a) {
			return true
		}
	}
	return false
}

// AllowsCommand reports whether argv starts with one of the allowed prefixes.
func (s TaskScope) AllowsCommand(argv []string) bool {
	for _, prefix := range s.Commands {
		if len(prefix) == 0 || len(prefix) > len(argv) {
			continue
		}
		ok := true
		for i := range prefix {
			if prefix[i] != argv[i] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// NeedsApproval reports whether the category requires a per-call approval.
func (a ApprovalPolicy) NeedsApproval(c model.ActionCategory) bool {
	for _, x := range a.Actions {
		if x == c {
			return true
		}
	}
	return false
}

// ToModel converts the brief to the legacy model.TaskBrief consumed by
// pkg/ai/sdlc, so existing engines keep working during migration.
func (b *TaskBrief) ToModel(ctx *ProjectContext) model.TaskBrief {
	var cmds []string
	for _, id := range b.Verification.Gates {
		if ctx != nil {
			if g, ok := ctx.Gate(id); ok {
				cmds = append(cmds, strings.Join(g.Argv, " "))
			}
		}
	}
	return model.TaskBrief{
		ID: b.ID, Title: b.Title, Objective: b.Objective, Repository: b.Repository,
		Scope: append([]string(nil), b.Scope.Allowed...), Constraints: append([]string(nil), b.Constraints...),
		AcceptanceCriteria: append([]string(nil), b.Deliverables...), VerificationCommands: cmds,
		RequirePlan: b.Approval.Plan, RequireReview: true, MaxAutonomy: b.MaxAutonomy,
		TimeoutSec: b.Limits.TimeoutSec, MaxRetries: b.Limits.MaxRetries,
	}
}

// PlanStep is one atomic, ordered step of a Plan.
type PlanStep struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	TargetFiles []string `json:"target_files"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Risk        Risk     `json:"risk"`
	Rollback    string   `json:"rollback,omitempty"`
}

// Plan is the engine-proposed decomposition of a task. The host computes
// Digest; approvals bind to it, so an edited plan invalidates its approval.
type Plan struct {
	TypeMeta
	TaskID  string     `json:"task_id"`
	Summary string     `json:"summary"`
	Steps   []PlanStep `json:"steps"`
	Digest  string     `json:"digest,omitempty"`
}
