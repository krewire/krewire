package projctx

import (
	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// CodeSpec describes one stable diagnostic code.
type CodeSpec struct {
	Class    model.DiagnosticSeverity
	Severity contract.Severity
	Title    string
}

// Codes is the registry of context lint codes. Codes are append-only and
// never change meaning; only error-severity codes block CI.
//
//	KWC1xxx missing · KWC2xxx conflicting · KWC3xxx stale · KWC4xxx malformed
var Codes = map[string]CodeSpec{
	"KWC1001": {model.DiagMissing, contract.SeverityError, "go.mod is missing"},
	"KWC1002": {model.DiagMissing, contract.SeverityError, "go.mod declares no module path or go version"},
	"KWC1003": {model.DiagMissing, contract.SeverityWarning, "AGENTS.md is missing; default quality gates apply"},
	"KWC1004": {model.DiagMissing, contract.SeverityWarning, "AGENTS.md has no Quality Gates block; default gates apply"},
	"KWC1005": {model.DiagMissing, contract.SeverityInfo, "no root version declaration (version.go Version)"},
	"KWC2001": {model.DiagConflict, contract.SeverityError, "krewire.yaml project.version conflicts with version.go"},
	"KWC2002": {model.DiagConflict, contract.SeverityError, "quality gate declared twice"},
	"KWC2003": {model.DiagConflict, contract.SeverityError, "rule ID declared twice"},
	"KWC3001": {model.DiagStale, contract.SeverityWarning, "maintained document references the retired kiw executable"},
	"KWC4001": {model.DiagMalformed, contract.SeverityError, "go.mod line cannot be parsed"},
	"KWC4002": {model.DiagMalformed, contract.SeverityError, "krewire.yaml is not valid YAML"},
	"KWC4003": {model.DiagMalformed, contract.SeverityError, "version declaration is not SemVer"},
	"KWC4004": {model.DiagMalformed, contract.SeverityWarning, "Go source file cannot be parsed"},
}

// lint runs the cross-source consistency checks that need more than one
// input; single-source checks are emitted while reading.
func (g *gen) lint() {
	seenGate := map[string]bool{}
	gates := g.ctx.Gates[:0]
	for _, gate := range g.ctx.Gates {
		if seenGate[gate.ID] {
			g.diag("KWC2002", "quality gate "+gate.ID+" is declared more than once; the first declaration wins",
				gate.Provenance, "give each gate command a distinct tool")
			continue
		}
		seenGate[gate.ID] = true
		gates = append(gates, gate)
	}
	g.ctx.Gates = gates
	seenRule := map[string]bool{}
	for _, r := range g.ctx.Rules {
		if seenRule[r.ID] {
			g.diag("KWC2003", "rule "+r.ID+" is declared more than once", r.Provenance, "rename one of the rules")
		}
		seenRule[r.ID] = true
	}
	var yamlVersion *contract.ConfigValue
	for i := range g.ctx.Config {
		if g.ctx.Config[i].Key == "project.version" {
			yamlVersion = &g.ctx.Config[i]
		}
	}
	if yamlVersion != nil && g.ctx.Module.Version != "" && yamlVersion.Value != g.ctx.Module.Version {
		g.diag("KWC2001", "krewire.yaml project.version "+yamlVersion.Value+" differs from "+
			g.ctx.Module.Provenance.Path+" Version "+g.ctx.Module.Version, yamlVersion.Provenance,
			"keep one version source: bump both with `krew release`")
	}
	g.staleDocs()
}
