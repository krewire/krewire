package model

import (
	"fmt"
)

// ContextRule represents an authoritative instruction, invariant, or constraint.
type ContextRule struct {
	ID         string     `json:"id"`
	Scope      Scope      `json:"scope"`
	Category   string     `json:"category"` // "pillar", "layering", "testing", "conventions"
	Statement  string     `json:"statement"`
	Provenance Provenance `json:"provenance"`
	EnforcedBy string     `json:"enforced_by,omitempty"`
}

// PillarInvariants tracks compliance with Krewire's Three Pillars (3S).
// Pillar names are strictly invariant: Secure, Sustainable, and Scalable.
type PillarInvariants struct {
	Secure      []ContextRule `json:"secure"`
	Sustainable []ContextRule `json:"sustainable"`
	Scalable    []ContextRule `json:"scalable"`
}

// QualityGateCommand represents a required verification check.
type QualityGateCommand struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// DiagnosticSeverity indicates the severity of a context integrity issue.
type DiagnosticSeverity string

const (
	DiagMissing   DiagnosticSeverity = "MISSING"
	DiagConflict  DiagnosticSeverity = "CONFLICT"
	DiagStale     DiagnosticSeverity = "STALE"
	DiagMalformed DiagnosticSeverity = "MALFORMED"
)

// PrecedenceTier represents the strict source authority order.
type PrecedenceTier int

const (
	// PrecedenceToolchain: Go compiler, OS primitives, stdlib.
	PrecedenceToolchain PrecedenceTier = 0
	// PrecedenceCoreKernel: model, version, license.
	PrecedenceCoreKernel PrecedenceTier = 1
	// PrecedenceWorkspaceRoot: Root AGENTS.md, go.work, README.md.
	PrecedenceWorkspaceRoot PrecedenceTier = 2
	// PrecedenceProjectConfig: krewire.yaml, go.mod.
	PrecedenceProjectConfig PrecedenceTier = 3
	// PrecedenceArchitecture: Architecture Decision Records (ADRs).
	PrecedenceArchitecture PrecedenceTier = 4
	// PrecedenceActiveTask: Current task / GitHub Issue specification.
	PrecedenceActiveTask PrecedenceTier = 5
)

// String returns human-readable label for the precedence tier.
func (p PrecedenceTier) String() string {
	switch p {
	case PrecedenceToolchain:
		return "0 (Toolchain/Stdlib)"
	case PrecedenceCoreKernel:
		return "1 (Core Kernel)"
	case PrecedenceWorkspaceRoot:
		return "2 (Workspace Root / AGENTS.md)"
	case PrecedenceProjectConfig:
		return "3 (Project Config / krewire.yaml)"
	case PrecedenceArchitecture:
		return "4 (Architecture Decisions / ADR)"
	case PrecedenceActiveTask:
		return "5 (Active Task / GitHub Issue)"
	default:
		return fmt.Sprintf("Precedence(%d)", p)
	}
}

// Provenance records the exact origin location of a context rule or decision.
type Provenance struct {
	File       string         `json:"file"`
	Line       int            `json:"line,omitempty"`
	Precedence PrecedenceTier `json:"precedence"`
	SourceKey  string         `json:"source_key,omitempty"`
}

// ContextDiagnostic records a detected context integrity issue.
type ContextDiagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Scope    Scope              `json:"scope"`
	File     string             `json:"file"`
	Line     int                `json:"line,omitempty"`
	Message  string             `json:"message"`
	Hint     string             `json:"hint,omitempty"`
}

// ProjectContext represents the authoritative, complete context for developers and AI agents.
type ProjectContext struct {
	Version     string               `json:"version"` // schema version, e.g. "1.0.0"
	Name        string               `json:"name"`
	Tier        Tier                 `json:"tier"`
	GoVersion   string               `json:"go_version"`
	ModulePath  string               `json:"module_path"`
	Workspace   string               `json:"workspace"`
	Pillars     PillarInvariants     `json:"pillars"`
	Rules       []ContextRule        `json:"rules"`
	Gates       []QualityGateCommand `json:"quality_gates"`
	Diagnostics []ContextDiagnostic  `json:"diagnostics,omitempty"`
}
