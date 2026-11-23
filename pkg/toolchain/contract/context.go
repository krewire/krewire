package contract

import "github.com/krewire/krewire/pkg/model"

// Provenance records where a resolved fact came from. Every fact in a
// ProjectContext carries one, so an agent can cite and a human can audit it.
type Provenance struct {
	// Path is the repository-relative, slash-separated source file.
	Path string `json:"path"`
	// Key is the key, directive, or symbol inside the file (e.g. "module",
	// "require github.com/x/y", "Version", "project.version").
	Key string `json:"key,omitempty"`
	// Line is the 1-based source line, when known.
	Line int `json:"line,omitempty"`
	// Schema names the format the fact was read with (e.g. "go.mod",
	// "go-source", "krewire.yaml", "agents-md", "contract-default").
	Schema string `json:"schema"`
}

// ModuleIdentity identifies the repository's Go module.
type ModuleIdentity struct {
	Path       string     `json:"path"`
	Version    string     `json:"version,omitempty"`
	Tier       string     `json:"tier,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// ToolchainIdentity names the toolchain and contract that produced the context.
type ToolchainIdentity struct {
	// Executable is the canonical developer-facing executable ("krew").
	Executable string `json:"executable"`
	// Version is the toolchain version, when the generator knows it.
	Version string `json:"version,omitempty"`
	// ContractVersion is the contract SemVer the document conforms to.
	ContractVersion string `json:"contract_version"`
}

// Workload describes what the repository ships and how mature it is.
type Workload struct {
	// Kind is descriptive (ADR-R8C2P): "library", "app", "service", …
	Kind string `json:"kind"`
	// Maturity is "preview" (v0.x), "stable" (>= v1), or "unknown".
	Maturity   string     `json:"maturity"`
	Provenance Provenance `json:"provenance"`
}

// Package is one Go package in the repository inventory.
type Package struct {
	ImportPath string `json:"import_path"`
	Dir        string `json:"dir"`
	Name       string `json:"name"`
	// Imports lists the package's imports that resolve inside the module,
	// sorted; together they form the internal dependency graph.
	Imports    []string   `json:"imports,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// Dependency is an external module requirement.
type Dependency struct {
	Path       string     `json:"path"`
	Version    string     `json:"version"`
	Indirect   bool       `json:"indirect,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// GraphRef summarizes the internal package graph and pins its digest, so
// richer graph engines (#135) can be referenced without being inlined.
type GraphRef struct {
	Nodes  int    `json:"nodes"`
	Edges  int    `json:"edges"`
	Digest string `json:"digest"`
	// Source names the producer of the graph ("go-imports").
	Source string `json:"source"`
}

// Rule is an architecture constraint, pillar invariant, or convention.
type Rule struct {
	ID       string `json:"id"`
	Category string `json:"category"` // "architecture", "pillar", "convention", "decision"
	// Statement is the human-readable rule.
	Statement string `json:"statement"`
	// EnforcedBy names the executable check that enforces the rule, if any.
	EnforcedBy string     `json:"enforced_by,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// ConfigValue is one resolved configuration key. Secret-looking keys are
// reported with Redacted=true and no value: secret contents never appear.
type ConfigValue struct {
	Key        string     `json:"key"`
	Value      string     `json:"value,omitempty"`
	Redacted   bool       `json:"redacted,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// Gate is a deterministic quality gate. Gates are argv vectors executed
// without a shell and never require an LLM or network access to an AI
// provider.
type Gate struct {
	ID       string                `json:"id"`
	Type     model.QualityGateType `json:"type"`
	Argv     []string              `json:"argv"`
	Env      []string              `json:"env,omitempty"`
	Required bool                  `json:"required"`
	// FailOnOutput marks gates whose failure signal is non-empty output
	// (e.g. `gofmt -l`), not only a non-zero exit code.
	FailOnOutput bool `json:"fail_on_output,omitempty"`
	// Provenance points at the declaration of the gate.
	Provenance Provenance `json:"provenance"`
}

// Target is a supported GOOS/GOARCH build target.
type Target struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

// Compatibility captures language, toolchain, and target requirements.
type Compatibility struct {
	GoVersion  string     `json:"go_version"`
	Toolchain  string     `json:"toolchain,omitempty"`
	Targets    []Target   `json:"targets,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// Pillars are the Three Pillars every Krewire change is filtered through.
var Pillars = []string{"Secure", "Sustainable", "Scalable"}

// Severity grades a diagnostic. Only SeverityError blocks CI.
type Severity string

// Diagnostic severities.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic is a stable, actionable context or contract finding.
type Diagnostic struct {
	// Code is stable across releases, e.g. "KWC1001".
	Code     string                   `json:"code"`
	Class    model.DiagnosticSeverity `json:"class"` // MISSING, CONFLICT, STALE, MALFORMED
	Severity Severity                 `json:"severity"`
	Message  string                   `json:"message"`
	Location Provenance               `json:"location"`
	Hint     string                   `json:"hint,omitempty"`
}

// ProjectContext is the authoritative, machine-readable project context
// consumed identically by deterministic commands and AI engines.
type ProjectContext struct {
	TypeMeta
	// Fingerprint is the SHA-256 over every input file read; identical
	// inputs yield an identical fingerprint and an identical document.
	Fingerprint   string            `json:"fingerprint"`
	Module        ModuleIdentity    `json:"module"`
	Toolchain     ToolchainIdentity `json:"toolchain"`
	Workload      Workload          `json:"workload"`
	Packages      []Package         `json:"packages"`
	Dependencies  []Dependency      `json:"dependencies"`
	Graph         GraphRef          `json:"graph"`
	Pillars       []string          `json:"pillars"`
	Rules         []Rule            `json:"rules"`
	Config        []ConfigValue     `json:"config"`
	Gates         []Gate            `json:"quality_gates"`
	Compatibility Compatibility     `json:"compatibility"`
	Diagnostics   []Diagnostic      `json:"diagnostics"`
}

// Blocking reports whether any diagnostic has error severity.
func (c *ProjectContext) Blocking() bool {
	for _, d := range c.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Gate returns the gate with the given ID.
func (c *ProjectContext) Gate(id string) (Gate, bool) {
	for _, g := range c.Gates {
		if g.ID == id {
			return g, true
		}
	}
	return Gate{}, false
}
