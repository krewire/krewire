package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// VerificationStatus classifies the outcome of an individual verification gate.
type VerificationStatus string

const (
	// GatePlanned indicates the verification step is scheduled but not yet run.
	GatePlanned VerificationStatus = "PLANNED"

	// GateAttempted indicates the verification step is currently executing.
	GateAttempted VerificationStatus = "ATTEMPTED"

	// GatePassed indicates the verification step succeeded with exit code 0.
	GatePassed VerificationStatus = "PASSED"

	// GateFailed indicates the verification step failed with non-zero exit or error.
	GateFailed VerificationStatus = "FAILED"

	// GateSkipped indicates the verification step was deliberately omitted with rationale.
	GateSkipped VerificationStatus = "SKIPPED"

	// GateBlocked indicates the step was blocked by prior failure or policy denial.
	GateBlocked VerificationStatus = "BLOCKED"
)

// QualityGateType categorizes the verification discipline.
type QualityGateType string

const (
	GateTypeFormat   QualityGateType = "FORMAT"
	GateTypeStatic   QualityGateType = "STATIC_ANALYSIS"
	GateTypeTest     QualityGateType = "TEST"
	GateTypeBuild    QualityGateType = "BUILD"
	GateTypeLayering QualityGateType = "LAYERING"
	GateTypeSecurity QualityGateType = "SECURITY"
	GateTypeCustom   QualityGateType = "CUSTOM"
)

// VerificationEvidence records the reproducible outcome of an individual gate.
type VerificationEvidence struct {
	GateID       string             `json:"gate_id"`
	Type         QualityGateType    `json:"type"`
	Description  string             `json:"description"`
	Command      string             `json:"command"`
	Required     bool               `json:"required"`
	Status       VerificationStatus `json:"status"`
	ExitCode     int                `json:"exit_code"`
	Output       string             `json:"output"`
	DurationMs   int64              `json:"duration_ms"`
	EvidenceHash string             `json:"evidence_hash"`
}

// ComputeHash generates a tamper-evident SHA-256 fingerprint of the verification evidence.
func (e *VerificationEvidence) ComputeHash(commit string) string {
	h := sha256.New()
	h.Write([]byte(e.GateID))
	h.Write([]byte(e.Command))
	h.Write([]byte(commit))
	h.Write([]byte(fmt.Sprintf("%d", e.ExitCode)))
	h.Write([]byte(e.Status))
	h.Write([]byte(e.Output))
	return hex.EncodeToString(h.Sum(nil))
}

// TaskVerificationRecord aggregates evidence across all declared verification gates
// for an AI-generated change or maintenance session.
type TaskVerificationRecord struct {
	TaskID           string                 `json:"task_id"`
	AgentID          string                 `json:"agent_id"`
	Timestamp        string                 `json:"timestamp"`
	GitCommit        string                 `json:"git_commit"`
	WorkingTreeClean bool                   `json:"working_tree_clean"`
	Gates            []VerificationEvidence `json:"gates"`
	OverallVerified  bool                   `json:"overall_verified"`
	Summary          string                 `json:"summary"`
}

// EvaluateTaskVerification computes the overall verification status from individual gates.
// Invariant: OverallVerified is true IF AND ONLY IF all required gates have status GatePassed.
// If any required gate is FAILED, SKIPPED, BLOCKED, or PLANNED, OverallVerified MUST be false.
func EvaluateTaskVerification(gates []VerificationEvidence) (bool, string) {
	if len(gates) == 0 {
		return false, "no verification gates were evaluated"
	}

	var passedCount, failedCount, skippedCount, blockedCount int
	var failureDetails []string

	for _, g := range gates {
		switch g.Status {
		case GatePassed:
			passedCount++
		case GateFailed:
			failedCount++
			if g.Required {
				failureDetails = append(failureDetails, fmt.Sprintf("%s (%s) failed with exit code %d", g.GateID, g.Command, g.ExitCode))
			}
		case GateSkipped:
			skippedCount++
			if g.Required {
				failureDetails = append(failureDetails, fmt.Sprintf("%s (%s) was skipped despite being required", g.GateID, g.Command))
			}
		case GateBlocked:
			blockedCount++
			if g.Required {
				failureDetails = append(failureDetails, fmt.Sprintf("%s (%s) was blocked", g.GateID, g.Command))
			}
		case GatePlanned, GateAttempted:
			if g.Required {
				failureDetails = append(failureDetails, fmt.Sprintf("%s (%s) was not completed (%s)", g.GateID, g.Command, g.Status))
			}
		}
	}

	if len(failureDetails) > 0 {
		summary := fmt.Sprintf("verification failed: %s (%d passed, %d failed, %d skipped, %d blocked)",
			strings.Join(failureDetails, "; "), passedCount, failedCount, skippedCount, blockedCount)
		return false, summary
	}

	summary := fmt.Sprintf("all %d required verification gates passed successfully", passedCount)
	return true, summary
}

// NewTaskVerificationRecord constructs an evaluated verification record with timestamps.
func NewTaskVerificationRecord(taskID, agentID, gitCommit string, clean bool, gates []VerificationEvidence) TaskVerificationRecord {
	verified, summary := EvaluateTaskVerification(gates)
	return TaskVerificationRecord{
		TaskID:           taskID,
		AgentID:          agentID,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		GitCommit:        gitCommit,
		WorkingTreeClean: clean,
		Gates:            gates,
		OverallVerified:  verified,
		Summary:          summary,
	}
}
