package model

import (
	"fmt"
	"github.com/krewire/krewire/packages/kern/errs"
	"strings"
)

// Scope is the ecosystem level a spec, test, or doc targets.
// Ordered: Workspace < Module < Domain < Service < Unit.
// See KWL-ARCH-J2K9Q.
//
// Workspace is the Krewire Workspace (hub dir ~/Workspace/Dev/krewire with bin/kiw + repos),
// which is also the Go `go.work` workspace at the hub root (see `go.work` and `AGENTS.md`).
// Domain is a DDD bounded context (e.g. catalog, user) — a cohesive set of Services
// inside a Module (internal/<domain>/). Pre-extraction it is a Service set, post-extraction
// it becomes its own Module/Project.
// Service is a Krewire runtime deployable, implemented in Go as a main package
// (e.g. cmd/<service>/ or service/<name>/); Unit is the smallest code unit inside
// a Service — a Go package, file, type, or function (e.g. framework/ui, ui.Button).
type Scope string

const (
	ScopeWorkspace Scope = "Workspace"
	ScopeModule    Scope = "Module"
	ScopeDomain    Scope = "Domain"
	ScopeService   Scope = "Service"
	ScopeUnit      Scope = "Unit"
)

// AllScopes lists every valid Scope in canonical order.
var AllScopes = []Scope{ScopeWorkspace, ScopeModule, ScopeDomain, ScopeService, ScopeUnit}

// scopeLevel maps Scope to its ordering index.
var scopeLevel = map[Scope]int{
	ScopeWorkspace: 0,
	ScopeModule:    1,
	ScopeDomain:    2,
	ScopeService:   3,
	ScopeUnit:      4,
}

// IsValid reports whether s is one of the five known scopes.
func (s Scope) IsValid() bool {
	_, ok := scopeLevel[s]
	return ok
}

// Level returns the ordering index (0..4). Invalid scopes return -1.
func (s Scope) Level() int {
	if v, ok := scopeLevel[s]; ok {
		return v
	}
	return -1
}

// Less reports whether s is ordered before other.
func (s Scope) Less(other Scope) bool {
	return s.Level() < other.Level()
}

// ParseScope parses s as a Scope, case-insensitive, returning UsageError on unknown.
// Accepted forms are the canonical names (Workspace, Module, Domain, Service, Unit),
// case-insensitive, with surrounding whitespace trimmed.
// "Package" and "Func" are no longer valid scopes — use Unit.
// "Project" is no longer a valid scope — use Module (Go module, formerly Project==Module).
func ParseScope(s string) (Scope, error) {
	trim := strings.TrimSpace(s)
	lower := strings.ToLower(trim)
	switch lower {
	case "workspace":
		return ScopeWorkspace, nil
	case "module":
		return ScopeModule, nil
	case "domain":
		return ScopeDomain, nil
	case "service":
		return ScopeService, nil
	case "unit":
		return ScopeUnit, nil
	default:
		return "", errs.UsageError(fmt.Sprintf("unknown scope %q: want one of Workspace, Module, Domain, Service, Unit", s))
	}
}
