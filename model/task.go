package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ErrInvalidTaskBrief is returned when a task brief lacks mandatory fields.
	ErrInvalidTaskBrief = errors.New("model: invalid task brief")

	// ErrScopeViolation is returned when a target file path escapes granted task scope.
	ErrScopeViolation = errors.New("model: target path escapes granted task scope")

	// ErrPlanRequired is returned when an action requires an approved plan first.
	ErrPlanRequired = errors.New("model: mutating action requires an approved plan")

	// ErrApprovalRequired is returned when an action requires explicit human authorization.
	ErrApprovalRequired = errors.New("model: action requires explicit human authorization")

	// ErrEscalationTriggered is returned when a hard blocking condition is encountered.
	ErrEscalationTriggered = errors.New("model: task requires hard human escalation")
)

// AutonomyLevel represents the human sovereignty & decision boundary tier.
// Conforms to ENG-AI-001 Section 4.
type AutonomyLevel string

const (
	// AutonomyAutonomous: Level 1 - Routine autonomous actions (reading, inspecting, testing, formatting, atomic commits).
	AutonomyAutonomous AutonomyLevel = "AUTONOMOUS"

	// AutonomyApprovalRequired: Level 2 - Consequential actions requiring explicit human confirmation (file deletions, git push, live migrations).
	AutonomyApprovalRequired AutonomyLevel = "APPROVAL_REQUIRED"

	// AutonomyEscalation: Level 3 - Hard blocking conditions requiring developer escalation (repeated gate failures, circular dependencies).
	AutonomyEscalation AutonomyLevel = "ESCALATION"
)

// ActionCategory categorizes agent operational intents.
type ActionCategory string

const (
	ActionRead    ActionCategory = "READ"
	ActionWrite   ActionCategory = "WRITE"
	ActionDelete  ActionCategory = "DELETE"
	ActionCommand ActionCategory = "COMMAND"
	ActionNetwork ActionCategory = "NETWORK"
	ActionGitPush ActionCategory = "GIT_PUSH"
	ActionDeploy  ActionCategory = "DEPLOY"
)

// ActionIntent declares an agent's intended action prior to execution.
type ActionIntent struct {
	ID           string         `json:"id"`
	Category     ActionCategory `json:"category"`
	ToolName     string         `json:"tool_name"`
	Target       string         `json:"target"`
	Description  string         `json:"description"`
	Autonomy     AutonomyLevel  `json:"autonomy"`
	RequiresPlan bool           `json:"requires_plan"`
	PayloadHash  string         `json:"payload_hash,omitempty"`
}

// ComputeHash computes a deterministic hash of the action intent.
func (a *ActionIntent) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(a.ID))
	h.Write([]byte(a.Category))
	h.Write([]byte(a.ToolName))
	h.Write([]byte(a.Target))
	h.Write([]byte(a.Autonomy))
	return hex.EncodeToString(h.Sum(nil))
}

// TaskBrief specifies the task requirements, bounds, constraints, and qualification criteria.
type TaskBrief struct {
	ID                   string        `json:"id"`
	Title                string        `json:"title"`
	Objective            string        `json:"objective"`
	Repository           string        `json:"repository"`
	Scope                []string      `json:"scope"`
	Constraints          []string      `json:"constraints"`
	AcceptanceCriteria   []string      `json:"acceptance_criteria"`
	VerificationCommands []string      `json:"verification_commands"`
	RequirePlan          bool          `json:"require_plan"`
	RequireReview        bool          `json:"require_review"`
	MaxAutonomy          AutonomyLevel `json:"max_autonomy"`
	TimeoutSec           int           `json:"timeout_sec,omitempty"`
	MaxRetries           int           `json:"max_retries,omitempty"`
}

// Validate checks that the task brief contains necessary fields and valid bounds.
func (b *TaskBrief) Validate() error {
	if strings.TrimSpace(b.ID) == "" {
		return fmt.Errorf("%w: task ID is required", ErrInvalidTaskBrief)
	}
	if strings.TrimSpace(b.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidTaskBrief)
	}
	if strings.TrimSpace(b.Objective) == "" {
		return fmt.Errorf("%w: objective is required", ErrInvalidTaskBrief)
	}
	if len(b.Scope) == 0 {
		return fmt.Errorf("%w: at least one scope boundary is required", ErrInvalidTaskBrief)
	}
	if b.MaxAutonomy == "" {
		b.MaxAutonomy = AutonomyAutonomous
	}
	if b.MaxRetries < 0 {
		return fmt.Errorf("%w: max retries cannot be negative", ErrInvalidTaskBrief)
	}
	return nil
}

// CheckScope verifies if a file path is within the allowed scope boundaries of the brief.
func (b *TaskBrief) CheckScope(path string) bool {
	if len(b.Scope) == 0 {
		return false
	}
	cleanPath := filepath.Clean(filepath.ToSlash(path))
	for _, s := range b.Scope {
		cleanScope := filepath.Clean(filepath.ToSlash(s))
		if cleanScope == "." || cleanScope == "*" || cleanScope == "" {
			return true
		}
		if cleanPath == cleanScope {
			return true
		}
		if strings.HasPrefix(cleanPath, cleanScope+"/") {
			return true
		}
	}
	return false
}

// PlanStep defines an atomic action in an SDLC execution plan.
type PlanStep struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	TargetFiles []string `json:"target_files"`
	RiskLevel   string   `json:"risk_level"` // "low", "medium", "high"
	Rollback    string   `json:"rollback,omitempty"`
}

// TaskPlan is the structured implementation plan generated prior to file modifications.
type TaskPlan struct {
	TaskID     string     `json:"task_id"`
	Summary    string     `json:"summary"`
	Steps      []PlanStep `json:"steps"`
	Approved   bool       `json:"approved"`
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt time.Time  `json:"approved_at,omitempty"`
	Token      string     `json:"token,omitempty"`
}

// FileMutation describes a recorded change produced during task execution.
type FileMutation struct {
	Path         string `json:"path"`
	Action       string `json:"action"` // "created", "modified", "deleted"
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	ContentHash  string `json:"content_hash,omitempty"`
}
