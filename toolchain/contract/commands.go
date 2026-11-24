package contract

// Executable is the one canonical developer-facing executable. `kiw` is
// retired (ADR-T1C4A); there is no second top-level developer CLI.
const Executable = "krew"

// CommandClass separates commands that never need an LLM from those that do.
type CommandClass string

// Command classes.
const (
	// ClassDeterministic commands run without model credentials or network
	// access to an AI provider, always.
	ClassDeterministic CommandClass = "deterministic"
	// ClassAgentic commands delegate reasoning to an engine through the
	// adapter protocol; without one they fail with CodeEngineUnavailable.
	ClassAgentic CommandClass = "agentic"
	// ClassOrchestrated commands run the whole lifecycle: deterministic
	// stages run in-process, agentic stages go through the adapter.
	ClassOrchestrated CommandClass = "orchestrated"
)

// Command is one entry of the canonical command hierarchy.
type Command struct {
	// Path is the command path below the executable, e.g. "context".
	Path    string       `json:"path"`
	Group   string       `json:"group"`
	Class   CommandClass `json:"class"`
	Summary string       `json:"summary"`
	// Stages lists the lifecycle stages the command serves, if any.
	Stages []Stage `json:"stages,omitempty"`
	// Consumes / Produces list contract kinds read and written.
	Consumes []Kind `json:"consumes,omitempty"`
	Produces []Kind `json:"produces,omitempty"`
	// RequiresEngine is true only for agentic commands.
	RequiresEngine bool `json:"requires_engine"`
	// Mutating is true when the command can change the working tree.
	Mutating bool `json:"mutating"`
}

// CommandCatalog is the published command hierarchy.
type CommandCatalog struct {
	TypeMeta
	Executable      string    `json:"executable"`
	ContractVersion string    `json:"contract_version"`
	Commands        []Command `json:"commands"`
}

func det(path, group, summary string, stages []Stage, consumes, produces []Kind, mutating bool) Command {
	return Command{Path: path, Group: group, Class: ClassDeterministic, Summary: summary,
		Stages: stages, Consumes: consumes, Produces: produces, Mutating: mutating}
}

func agt(path, summary string, stages []Stage, consumes, produces []Kind, mutating bool) Command {
	return Command{Path: path, Group: "ai", Class: ClassAgentic, Summary: summary,
		Stages: stages, Consumes: consumes, Produces: produces, RequiresEngine: true, Mutating: mutating}
}

// Catalog returns the canonical `krew` command hierarchy that the toolchain
// implements. Existing krew command names are kept; the AI lifecycle
// commands are added to the same tree instead of a separate CLI.
func Catalog() CommandCatalog {
	ctx := []Kind{KindProjectContext}
	ctxTask := []Kind{KindProjectContext, KindTaskBrief}
	cmds := []Command{
		det("version", "inspect", "print toolchain, framework, and contract versions", nil, nil, nil, false),
		det("doctor", "inspect", "diagnose environment and toolchain prerequisites", nil, nil, nil, false),
		det("context", "inspect", "derive the project context (--json) and lint it (--lint)", []Stage{StageDiscover}, nil, ctx, false),
		det("config", "inspect", "validate krewire.yaml or print its schema", nil, nil, nil, false),
		det("compat", "inspect", "check version compatibility across modules", nil, ctx, nil, false),
		det("contract schema", "inspect", "print the JSON Schema of a contract kind", nil, nil, nil, false),
		det("contract validate", "inspect", "validate any contract document", nil, nil, nil, false),
		det("task validate", "develop", "validate a task brief against the project context", nil, ctxTask, nil, false),
		det("fmt", "develop", "check or apply formatting", []Stage{StageVerify}, nil, nil, true),
		det("vet", "develop", "static analysis and security posture checks", []Stage{StageVerify}, nil, nil, false),
		det("test", "develop", "run the project's tests", []Stage{StageVerify}, nil, nil, false),
		det("build", "build", "build the project", []Stage{StageVerify}, nil, nil, true),
		det("verify", "develop", "run the required quality gates and record evidence", []Stage{StageVerify}, ctxTask, []Kind{KindEvidenceReport}, false),
		det("report", "develop", "render or check an evidence report", []Stage{StageReport}, []Kind{KindEvidenceReport}, nil, false),
		det("engine status", "ai", "show the configured engine and run the adapter handshake", nil, nil, nil, false),
		det("release", "ship", "bump versions and stage a release", nil, ctx, nil, true),
		det("completion", "project", "print shell completion scripts", nil, nil, nil, false),
		det("self-update", "project", "update the krew toolchain", nil, nil, nil, true),
		agt("plan", "plan and decompose a task brief into an approvable plan", []Stage{StagePlan, StageDecompose}, ctxTask, []Kind{KindPlan}, false),
		agt("agent", "implement an approved plan through policy-checked host tools", []Stage{StageImplement}, []Kind{KindProjectContext, KindTaskBrief, KindPlan}, []Kind{KindExecutionRecord}, true),
		agt("review", "review changes and risks against the brief and evidence", []Stage{StageReview}, ctxTask, nil, false),
		agt("chat", "interactive AI session bound to the project context", nil, ctx, nil, true),
		{Path: "flow", Group: "ai", Class: ClassOrchestrated,
			Summary:  "run discover → plan → implement → verify → review → report",
			Stages:   Stages,
			Consumes: ctxTask, Produces: []Kind{KindPlan, KindExecutionRecord, KindEvidenceReport},
			Mutating: true},
	}
	return CommandCatalog{TypeMeta: Meta(KindCommandCatalog), Executable: Executable,
		ContractVersion: ContractVersion, Commands: cmds}
}

// Lookup returns the command with the given path.
func (c CommandCatalog) Lookup(path string) (Command, bool) {
	for _, cmd := range c.Commands {
		if cmd.Path == path {
			return cmd, true
		}
	}
	return Command{}, false
}
