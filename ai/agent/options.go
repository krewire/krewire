package agent

import (
	"encoding/json"

	"github.com/krewire/krewire/ai"
	"github.com/krewire/krewire/ai/memory"
	"github.com/krewire/krewire/ai/persona"
	"github.com/krewire/krewire/ai/tools"
)

// Approver is a function called before invoking a tool to confirm permission.
type Approver func(toolName string, args json.RawMessage) bool

// SecurityProfile defines the operational safety boundary for an Agent.
type SecurityProfile string

const (
	// ProfileProduction is the default secure profile. In this profile, all privileged tools
	// (command execution, filesystem modifications, and network egress) strictly fail-closed
	// unless an explicit security policy (Approver or Authorizer) is configured.
	ProfileProduction SecurityProfile = "production"

	// ProfileReadOnly restricts the agent exclusively to non-mutating inspection tools.
	// Command execution, file writing/editing/patching, and privileged operations are strictly
	// denied unconditionally, even if an approver is provided.
	ProfileReadOnly SecurityProfile = "read_only"

	// ProfileInsecureDevMode permits unrestricted tool execution without mandatory approval checks.
	// This mode is provided strictly for local developer CLI workflows and hermetic tests.
	// WARNING: Must NEVER be enabled in production or multi-tenant server environments.
	ProfileInsecureDevMode SecurityProfile = "insecure_dev_mode"
)

// Options configures an Agent instance.
type Options struct {
	Provider        ai.Provider
	Model           string
	SystemPrompt    string
	Persona         persona.Persona
	Registry        *tools.Registry
	Memory          memory.Memory
	MaxSteps        int
	Temperature     *float64
	Approver        Approver
	StrictApprovals bool // abort loop and return ErrPermissionDenied if approver rejects
	OnEvent         EventHandler
	Streaming       bool // stream LLM tokens as text_delta events
	SecurityProfile SecurityProfile
	Authorizer      Authorizer
	Roles           []string

	// MaxToolResultTokens limits the estimated token count of individual tool output turns.
	MaxToolResultTokens int

	// MaxConcurrentTools limits the maximum number of independent parallel-safe tools
	// that can be executed concurrently in a single step (default: DefaultMaxConcurrentTools).
	MaxConcurrentTools int

	// Subagent delegation plumbing.
	SubDepth int    // delegation levels below the root agent (0 = root)
	ThreadID string // nested thread id (parent_id:child_id) for events
}

// Option modifies Options.
type Option func(*Options)

// WithProvider sets the LLM provider.
func WithProvider(p ai.Provider) Option {
	return func(o *Options) { o.Provider = p }
}

// WithModel sets the model identifier.
func WithModel(m string) Option {
	return func(o *Options) { o.Model = m }
}

// WithPersona assigns a specialized persona and system prompt.
func WithPersona(p persona.Persona) Option {
	return func(o *Options) {
		o.Persona = p
		o.SystemPrompt = p.Instructions
	}
}

// WithSystemPrompt customizes the agent's base system prompt.
func WithSystemPrompt(sys string) Option {
	return func(o *Options) { o.SystemPrompt = sys }
}

// WithTools assigns a tools registry.
func WithTools(r *tools.Registry) Option {
	return func(o *Options) { o.Registry = r }
}

// WithMemory assigns conversation memory.
func WithMemory(m memory.Memory) Option {
	return func(o *Options) { o.Memory = m }
}

// WithMaxSteps sets the upper bound of iterative tool execution steps.
func WithMaxSteps(steps int) Option {
	return func(o *Options) { o.MaxSteps = steps }
}

// WithTemperature sets the generation temperature.
func WithTemperature(temp float64) Option {
	return func(o *Options) { o.Temperature = &temp }
}

// WithStreaming toggles token-level text_delta streaming via the provider's
// Stream endpoint when supported.
func WithStreaming(on bool) Option {
	return func(o *Options) { o.Streaming = on }
}

// WithApprover sets an approval callback for sensitive tool executions.
func WithApprover(appr Approver) Option {
	return func(o *Options) { o.Approver = appr }
}

// WithStrictApprovals controls whether tool denials abort execution with ErrPermissionDenied.
func WithStrictApprovals(strict bool) Option {
	return func(o *Options) { o.StrictApprovals = strict }
}

// WithEventHandler registers a streaming event listener.
func WithEventHandler(h EventHandler) Option {
	return func(o *Options) { o.OnEvent = h }
}

// WithMaxToolResultTokens bounds the estimated token count of each tool result payload.
func WithMaxToolResultTokens(tokens int) Option {
	return func(o *Options) { o.MaxToolResultTokens = tokens }
}

// WithMaxConcurrentTools bounds the concurrency for executing parallel-safe tools within a step.
func WithMaxConcurrentTools(concurrency int) Option {
	return func(o *Options) { o.MaxConcurrentTools = concurrency }
}

// WithSecurityProfile sets the agent's execution safety profile (ProfileProduction, ProfileReadOnly, ProfileInsecureDevMode).
func WithSecurityProfile(p SecurityProfile) Option {
	return func(o *Options) { o.SecurityProfile = p }
}

// WithInsecureDevMode explicitly opts into unrestricted local developer execution.
// In this mode, privileged tools can execute without an approval or authorization policy.
// WARNING: This setting disables essential security controls and must NEVER be enabled
// in production servers or multi-tenant deployments.
func WithInsecureDevMode() Option {
	return func(o *Options) { o.SecurityProfile = ProfileInsecureDevMode }
}

// WithAuthorizer binds an RBAC Authorizer and active roles to the agent for tool authorization.
func WithAuthorizer(auth Authorizer, roles ...string) Option {
	return func(o *Options) {
		o.Authorizer = auth
		o.Roles = roles
	}
}
