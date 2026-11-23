// Package adapter defines the protocol between the unified toolchain (host)
// and an AI engine — the proprietary Krew engine or any third-party agent —
// and ships both ends of its reference transport: JSON-RPC 2.0 over stdio,
// one JSON message per line.
//
// The engine is a pure reasoning component: it receives the shared
// ProjectContext and TaskBrief, returns plans, proposed tool calls, review
// findings, and claims. It never executes mutations itself; the host
// authorizes and executes every tool call (pkg/toolchain/policy) and runs
// every verification gate. This keeps policy, approvals, and evidence in one
// place and lets the engine live behind its own repository and license
// boundary.
package adapter

import (
	"context"
	"time"

	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// Protocol method names.
const (
	MethodInitialize = "initialize"
	MethodPlan       = "plan"
	MethodImplement  = "implement"
	MethodReview     = "review"
	MethodShutdown   = "shutdown"
	// NotifyCancel is a notification ({"id": <request id>}) that cancels
	// an in-flight request. The engine answers it with CodeCancelled.
	NotifyCancel = "$/cancelRequest"
)

// EnvEngine names the environment variable holding the engine command line
// the toolchain launches, e.g. `krew-engine --stdio`.
const EnvEngine = "KREW_ENGINE"

// Meta carries per-request execution bounds.
type Meta struct {
	// TimeoutMs is the remaining deadline in milliseconds; the engine must
	// abandon the request with CodeTimeout once it elapses. 0 = none.
	TimeoutMs int64 `json:"timeout_ms,omitempty"`
}

// InitializeParams opens a session.
type InitializeParams struct {
	ContractVersion string                  `json:"contract_version"`
	Host            contract.EngineIdentity `json:"host"`
}

// InitializeResult describes the engine.
type InitializeResult struct {
	Engine contract.EngineIdentity `json:"engine"`
	// Methods lists the protocol methods the engine implements.
	Methods []string `json:"methods"`
	// Ready is false when the engine lacks credentials or provider access;
	// the host then reports engine_unavailable for agentic stages only.
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// PlanParams asks for a plan of a task.
type PlanParams struct {
	Meta    Meta                     `json:"meta"`
	Context *contract.ProjectContext `json:"context"`
	Task    *contract.TaskBrief      `json:"task"`
}

// PlanResult is the engine's plan and the claims it makes about it.
type PlanResult struct {
	Plan   contract.Plan    `json:"plan"`
	Claims []contract.Claim `json:"claims,omitempty"`
}

// ImplementParams asks for the tool calls that implement one plan step.
type ImplementParams struct {
	Meta    Meta                     `json:"meta"`
	Context *contract.ProjectContext `json:"context"`
	Task    *contract.TaskBrief      `json:"task"`
	Plan    *contract.Plan           `json:"plan"`
	StepID  string                   `json:"step_id"`
	// Previous holds the host's execution records so far, including
	// denials, so the engine can adapt.
	Previous []contract.ExecutionRecord `json:"previous,omitempty"`
}

// ImplementResult proposes tool calls for the host to authorize and run.
type ImplementResult struct {
	Calls  []contract.ToolCall `json:"calls"`
	Claims []contract.Claim    `json:"claims,omitempty"`
}

// ReviewParams asks for a review of the executed change and its evidence.
type ReviewParams struct {
	Meta         Meta                         `json:"meta"`
	Context      *contract.ProjectContext     `json:"context"`
	Task         *contract.TaskBrief          `json:"task"`
	Plan         *contract.Plan               `json:"plan"`
	Executions   []contract.ExecutionRecord   `json:"executions"`
	Verification contract.VerificationSection `json:"verification"`
}

// ReviewResult carries findings and claims.
type ReviewResult struct {
	Findings []contract.Finding `json:"findings,omitempty"`
	Claims   []contract.Claim   `json:"claims,omitempty"`
}

// Engine is implemented by every AI engine and by the stdio Client.
// Implementations must honour ctx cancellation and return contract errors
// (*contract.Error) so codes survive the process boundary.
type Engine interface {
	Initialize(ctx context.Context, p InitializeParams) (InitializeResult, error)
	Plan(ctx context.Context, p PlanParams) (PlanResult, error)
	Implement(ctx context.Context, p ImplementParams) (ImplementResult, error)
	Review(ctx context.Context, p ReviewParams) (ReviewResult, error)
}

// MetaFor derives Meta from ctx's deadline.
func MetaFor(ctx context.Context) Meta {
	if dl, ok := ctx.Deadline(); ok {
		ms := time.Until(dl).Milliseconds()
		if ms < 1 {
			ms = 1
		}
		return Meta{TimeoutMs: ms}
	}
	return Meta{}
}

// Unavailable is the engine used when none is configured. Every agentic
// method fails with engine_unavailable; deterministic work is unaffected.
type Unavailable struct{ Reason string }

func (u Unavailable) err() error {
	return contract.Errorf(contract.CodeEngineUnavailable, "%s", u.Reason)
}

// Initialize implements Engine.
func (u Unavailable) Initialize(context.Context, InitializeParams) (InitializeResult, error) {
	return InitializeResult{Engine: contract.EngineIdentity{Name: "none", ContractVersion: contract.ContractVersion},
		Ready: false, Reason: u.Reason}, nil
}

// Plan implements Engine.
func (u Unavailable) Plan(context.Context, PlanParams) (PlanResult, error) {
	return PlanResult{}, u.err()
}

// Implement implements Engine.
func (u Unavailable) Implement(context.Context, ImplementParams) (ImplementResult, error) {
	return ImplementResult{}, u.err()
}

// Review implements Engine.
func (u Unavailable) Review(context.Context, ReviewParams) (ReviewResult, error) {
	return ReviewResult{}, u.err()
}
