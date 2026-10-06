// Package kern is the Krewire Kernel — the bottom layer of the Krewire
// ecosystem.
//
// It depends on nothing outside the Go standard library, so every other
// Krewire module may import it without creating a dependency cycle.
//
// This root package is a facade. The implementation is split by concern:
//
//   - errs       — errors, exit codes, diagnostics, stack traces
//   - model      — Kind, Project, Scope, the generic workload matrix, opt-in rules
//   - spec       — SpecID and RequirementID
//   - version    — the Version value type and its semver rules
//   - lifecycle  — Kernel, Registry, Executor, Supervisor
//   - env        — target environments
//   - log        — structured logging
//
// Every exported symbol below is an alias of the subpackage that owns it, so a
// consumer needs a single import to reach the whole kernel. Import the
// subpackage directly when the boundary matters.
//
// # The kernel names nothing
//
// The kernel is the lowest layer, so it must not know which modules exist above
// it. It therefore ships no module roster, no ecosystem version matrix, no
// licensing policy, and no workload table: those are ecosystem facts, and they
// live in the layer above.
//
// The kernel provides only the shapes such facts are written in:
//
//   - version.Version and its semver rules judge any version a module declares
//   - model.Matrix holds a caller-declared kind → package mapping
//   - model.OptInRule states which packages a kind may not import directly
//   - model.Kind, model.Project, model.Scope and spec.SpecID are the vocabulary
//     those declarations are written in
//
// Each module declares itself in its own module. Adding a module never edits
// this one, and TestKERN_LAYER_001_NoEcosystemNames fails CI if that regresses.
package kern

import (
	"github.com/krewire/krewire/packages/kern/errs"
	"github.com/krewire/krewire/packages/kern/lifecycle"
	"github.com/krewire/krewire/packages/kern/model"
	"github.com/krewire/krewire/packages/kern/spec"
	"github.com/krewire/krewire/packages/kern/version"
)

type (
	// Attr is one structured key/value pair attached to an error.
	//
	// Alias of errs.Attr.
	Attr = errs.Attr
	// Error pairs a human-readable message with an ExitCode.
	//
	// Alias of errs.Error.
	Error = errs.Error
	// ExitCode is a standard process exit code.
	//
	// Alias of errs.ExitCode.
	ExitCode = errs.ExitCode
	// StackFrame is one rendered-ready entry of a captured stack.
	//
	// Alias of errs.StackFrame.
	StackFrame = errs.StackFrame

	// Kind is a Krewire project kind.
	//
	// Alias of model.Kind.
	Kind = model.Kind
	// Project is a Krewire project as declared in krewire.yaml.
	//
	// Alias of model.Project.
	Project = model.Project
	// Scope is a level of the Krewire scope hierarchy.
	//
	// Alias of model.Scope.
	Scope = model.Scope
	// Status is the implementation status of a workload.
	//
	// Alias of model.Status.
	Status = model.Status
	// DomainEvent is a typed fact emitted by the running system.
	//
	// Alias of model.DomainEvent.
	DomainEvent = model.DomainEvent

	// Workload is one cell of a workload matrix, generic over the owning
	// package type.
	//
	// Alias of model.Workload.
	Workload[P any] = model.Workload[P]
	// Matrix is the ordered kind → workload registry for one package type.
	//
	// Alias of model.Matrix.
	Matrix[P any] = model.Matrix[P]
	// OptInRule declares packages a kind may not import directly.
	//
	// Alias of model.OptInRule.
	OptInRule[P ~string] = model.OptInRule[P]

	// Version is a semantic version.
	//
	// Alias of version.Version.
	Version = version.Version

	// SpecID uniquely identifies a specification document.
	//
	// Alias of spec.SpecID.
	SpecID = spec.SpecID
	// RequirementID traces a requirement back to its owning specification.
	//
	// Alias of spec.RequirementID.
	RequirementID = spec.RequirementID

	// Kernel is the imperative control plane that boots, supervises, and
	// executes workloads.
	//
	// Alias of lifecycle.Kernel.
	Kernel = lifecycle.Kernel
	// Registry resolves modules by name in dependency order.
	//
	// Alias of lifecycle.Registry.
	Registry = lifecycle.Registry
	// Executor dispatches a workload to the module that handles its Kind.
	//
	// Alias of lifecycle.Executor.
	Executor = lifecycle.Executor
	// Supervisor starts and stops Startable modules.
	//
	// Alias of lifecycle.Supervisor.
	Supervisor = lifecycle.Supervisor
	// Module is a registered unit of work.
	//
	// Alias of lifecycle.Module.
	Module = lifecycle.Module
	// Startable is a module with a managed lifecycle.
	//
	// Alias of lifecycle.Startable.
	Startable = lifecycle.Startable
	// Depender is a module that depends on other modules.
	//
	// Alias of lifecycle.Depender.
	Depender = lifecycle.Depender
)

const (
	ExitCodeSuccess = errs.ExitCodeSuccess
	ExitCodeFailure = errs.ExitCodeFailure
	ExitCodeUsage   = errs.ExitCodeUsage
)

var (
	NewError        = errs.NewError
	UsageError      = errs.UsageError
	FailureError    = errs.FailureError
	ExitCodeFromInt = errs.ExitCodeFromInt

	WithAttrs  = errs.WithAttrs
	AttrsOf    = errs.AttrsOf
	WithHint   = errs.WithHint
	HintOf     = errs.HintOf
	FormatTree = errs.FormatTree

	WithStack   = errs.WithStack
	StackOf     = errs.StackOf
	FormatStack = errs.FormatStack

	AllKinds  = model.AllKinds
	ParseKind = model.ParseKind

	AllScopes  = model.AllScopes
	ParseScope = model.ParseScope

	ValidateKrewireYamlPath = model.ValidateKrewireYamlPath
	NewDomainEvent          = model.NewDomainEvent

	ParseSpecID        = spec.ParseSpecID
	ParseRequirementID = spec.ParseRequirementID

	MustParseVersion = version.MustParseVersion
	ParseVersion     = version.ParseVersion

	NewKernel     = lifecycle.New
	NewRegistry   = lifecycle.NewRegistry
	NewSupervisor = lifecycle.NewSupervisor
)

// NewMatrix returns a workload Matrix for package type P.
//
// Alias of model.NewMatrix.
func NewMatrix[P any](cells ...Workload[P]) *Matrix[P] { return model.NewMatrix(cells...) }

// ViolatesOptIn reports whether imported breaks any rule in rules for kind.
//
// Alias of model.ViolatesOptIn.
func ViolatesOptIn[P ~string](kind Kind, imported []P, rules []OptInRule[P]) bool {
	return model.ViolatesOptIn(kind, imported, rules)
}
