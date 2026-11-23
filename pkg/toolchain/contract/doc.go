// Package contract defines the versioned, machine-readable contracts shared by
// the unified Krewire developer toolchain (`krew`), its deterministic commands,
// and every AI engine that plugs into it — the proprietary Krew engine as well
// as third-party agents.
//
// The contract is the single integration surface between deterministic
// developer tooling and AI-native engineering workflows (ADR-T1C4A, issues
// #131 and #140). It covers:
//
//   - ProjectContext — authoritative, provenance-tagged project facts.
//   - TaskBrief — governed task scope, dependencies, gates, and approvals.
//   - Plan — an engine-proposed, host-approved execution plan.
//   - ToolCall / Decision / ApprovalRecord — policy-bounded execution.
//   - ExecutionRecord — what the host actually did on the engine's behalf.
//   - EvidenceReport — raw deterministic verification next to (never merged
//     with) engine-generated claims.
//   - CommandCatalog — the canonical command hierarchy, classifying every
//     command as deterministic or agentic.
//
// Every document carries `api_version` and `kind`. The wire format is JSON;
// the published JSON Schemas live under schemas/ and are generated from these
// Go types (see Schema), so the Go representation and the schema cannot drift.
//
// The package depends only on the standard library and pkg/model, so any
// consumer — including one that never sees private Krew source — can validate
// documents with it or with any JSON Schema validator.
package contract

import (
	"fmt"
	"strings"
)

const (
	// Group is the API group of every toolchain contract document.
	Group = "toolchain.krewire.dev"

	// Version is the wire version of the contract documents. A new wire
	// version is introduced only for breaking changes.
	Version = "v1"

	// APIVersion is the value of the api_version field of every document.
	APIVersion = Group + "/" + Version

	// ContractVersion is the SemVer of the contract. Minor bumps are additive
	// (new optional fields, new methods, new enum values); major bumps change
	// APIVersion. Engines negotiate on the major component.
	ContractVersion = "1.0.0"
)

// Kind names a contract document type.
type Kind string

// Document kinds.
const (
	KindProjectContext  Kind = "ProjectContext"
	KindTaskBrief       Kind = "TaskBrief"
	KindPlan            Kind = "Plan"
	KindPolicy          Kind = "Policy"
	KindApprovalRecord  Kind = "ApprovalRecord"
	KindExecutionRecord Kind = "ExecutionRecord"
	KindEvidenceReport  Kind = "EvidenceReport"
	KindCommandCatalog  Kind = "CommandCatalog"
)

// Kinds lists every document kind in canonical order.
var Kinds = []Kind{
	KindProjectContext, KindTaskBrief, KindPlan, KindPolicy,
	KindApprovalRecord, KindExecutionRecord, KindEvidenceReport, KindCommandCatalog,
}

// TypeMeta identifies a contract document's version and kind.
type TypeMeta struct {
	APIVersion string `json:"api_version"`
	Kind       Kind   `json:"kind"`
}

// Meta returns the TypeMeta for a document of the given kind at the current
// contract version.
func Meta(k Kind) TypeMeta { return TypeMeta{APIVersion: APIVersion, Kind: k} }

// Check reports whether the document carries the expected api_version and kind.
func (m TypeMeta) Check(want Kind) error {
	if m.APIVersion != APIVersion {
		return Errorf(CodeVersionUnsupported, "api_version %q is not supported (want %q)", m.APIVersion, APIVersion)
	}
	if m.Kind != want {
		return Errorf(CodeInvalidDocument, "kind %q does not match expected %q", m.Kind, want)
	}
	return nil
}

// CompatibleContract reports whether a peer's contract version is wire
// compatible with this one: same major version, any minor or patch.
func CompatibleContract(peer string) error {
	major := func(v string) string {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		m, _, _ := strings.Cut(v, ".")
		return m
	}
	if peer == "" || major(peer) != major(ContractVersion) {
		return Errorf(CodeVersionUnsupported, "contract version %q is incompatible with %s", peer, ContractVersion)
	}
	return nil
}

// requireText returns an invalid-document error when s is blank.
func requireText(field, s string) error {
	if strings.TrimSpace(s) == "" {
		return Errorf(CodeInvalidDocument, "%s is required", field)
	}
	return nil
}

// fieldf prefixes a validation error with the offending field path.
func fieldf(field string, i int) string { return fmt.Sprintf("%s[%d]", field, i) }
