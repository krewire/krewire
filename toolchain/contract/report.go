package contract

import "github.com/krewire/krewire/model"

// Stage is a step of the shared AI-native workflow lifecycle (#140 §3).
type Stage string

// Lifecycle stages in execution order.
const (
	StageDiscover  Stage = "discover"
	StagePlan      Stage = "plan"
	StageDecompose Stage = "decompose"
	StageImplement Stage = "implement"
	StageVerify    Stage = "verify"
	StageReview    Stage = "review"
	StageReport    Stage = "report"
)

// Stages lists the lifecycle in order.
var Stages = []Stage{StageDiscover, StagePlan, StageDecompose, StageImplement, StageVerify, StageReview, StageReport}

// StageStatus is the outcome of one lifecycle stage.
type StageStatus string

// Stage statuses.
const (
	StageOK      StageStatus = "ok"
	StageFailed  StageStatus = "failed"
	StageSkipped StageStatus = "skipped"
)

// StageRecord records one lifecycle stage.
type StageRecord struct {
	Stage      Stage       `json:"stage"`
	Status     StageStatus `json:"status"`
	Actor      string      `json:"actor"` // "toolchain" or "engine:<name>"
	DurationMs int64       `json:"duration_ms"`
	Error      *Error      `json:"error,omitempty"`
}

// Claim is a statement made by an engine. Claims are never evidence: the
// report keeps them apart from deterministic results and marks them
// unverified unless a passed gate is referenced.
type Claim struct {
	Source    string `json:"source"`
	Stage     Stage  `json:"stage"`
	Statement string `json:"statement"`
	// VerifiedBy names a gate ID that passed and substantiates the claim;
	// empty means the claim is unverified.
	VerifiedBy string `json:"verified_by,omitempty"`
}

// Finding is a review observation.
type Finding struct {
	Severity Severity `json:"severity"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// EngineIdentity names the engine that participated, if any.
type EngineIdentity struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ContractVersion string `json:"contract_version"`
}

// VerificationSection holds the raw deterministic outcomes. Only the host
// writes it; Verified follows model.EvaluateTaskVerification.
type VerificationSection struct {
	Gates           []model.VerificationEvidence `json:"gates"`
	Verified        bool                         `json:"verified"`
	Summary         string                       `json:"summary"`
	GitCommit       string                       `json:"git_commit,omitempty"`
	LedgerBlockHash string                       `json:"ledger_block_hash,omitempty"`
}

// Outcome is the overall result of a workflow run.
type Outcome string

// Outcomes.
const (
	OutcomeVerified           Outcome = "verified"
	OutcomeVerificationFailed Outcome = "verification_failed"
	OutcomeDenied             Outcome = "denied"
	OutcomeCancelled          Outcome = "cancelled"
	OutcomeEngineUnavailable  Outcome = "engine_unavailable"
	OutcomeFailed             Outcome = "failed"
)

// EvidenceReport is the evidence-backed completion report.
type EvidenceReport struct {
	TypeMeta
	TaskID             string               `json:"task_id"`
	ContextFingerprint string               `json:"context_fingerprint"`
	PlanDigest         string               `json:"plan_digest,omitempty"`
	Engine             *EngineIdentity      `json:"engine,omitempty"`
	Outcome            Outcome              `json:"outcome"`
	Stages             []StageRecord        `json:"stages"`
	Approvals          []ApprovalRecord     `json:"approvals,omitempty"`
	Executions         []ExecutionRecord    `json:"executions,omitempty"`
	Mutations          []model.FileMutation `json:"mutations,omitempty"`
	Verification       VerificationSection  `json:"verification"`
	Claims             []Claim              `json:"claims,omitempty"`
	Findings           []Finding            `json:"findings,omitempty"`
}
