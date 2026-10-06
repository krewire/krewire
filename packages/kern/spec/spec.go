package spec

import (
	"fmt"
	"github.com/krewire/krewire/packages/kern/errs"
	"regexp"
	"strings"
)

// SpecID is a Krewire specification identifier. Two forms are accepted:
//   - Short: KWF-M8K2Q (ProjectId-Code)
//   - Full file prefix: KWF-ARCH-M8K2Q (ProjectId-Scope-Code) — with or without slug suffix.
type SpecID string

var (
	// allowedProjects is the documented spec-ID prefix set (AGENTS.md):
	// framework, kiw, libs, mdbind, boost, and the sites.
	allowedProjects = map[string]bool{
		"KWF": true, "KWN": true, "KWL": true, "KWM": true, "KWG": true, "KRW": true,
	}
	scopeRe = regexp.MustCompile(`^[A-Z0-9]+$`)
	codeRe  = regexp.MustCompile(`^[A-Za-z0-9]{5}$`)
)

// ParseSpecID validates s as a SpecID and returns UsageError on failure.
func ParseSpecID(s string) (SpecID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errs.UsageError("spec ID is required")
	}
	// Strip slug suffix if present: KWF-ARCH-M8K2Q-unified-vision -> KWF-ARCH-M8K2Q
	// Count hyphens to decide form.
	parts := strings.Split(s, "-")
	var project, scope, code string
	switch len(parts) {
	case 2:
		// KWF-M8K2Q
		project, code = parts[0], parts[1]
		scope = ""
	case 3:
		// KWF-ARCH-M8K2Q
		project, scope, code = parts[0], parts[1], parts[2]
	default:
		// KWF-ARCH-M8K2Q-unified-vision -> first 3 parts are the SpecID
		if len(parts) >= 3 {
			project, scope, code = parts[0], parts[1], parts[2]
			// Validate that code looks like 5-char; otherwise treat as invalid
		} else {
			return "", errs.UsageError(fmt.Sprintf("invalid spec ID %q: want {ProjectId}-{Scope}-{Code} or {ProjectId}-{Code}", s))
		}
	}
	if !allowedProjects[project] {
		return "", errs.UsageError(fmt.Sprintf("invalid spec ID %q: unknown project %q", s, project))
	}
	if scope != "" && !scopeRe.MatchString(scope) {
		return "", errs.UsageError(fmt.Sprintf("invalid spec ID %q: scope %q must be [A-Z0-9]+", s, scope))
	}
	if !codeRe.MatchString(code) {
		return "", errs.UsageError(fmt.Sprintf("invalid spec ID %q: code %q must be 5 alphanumeric chars", s, code))
	}
	// Normalize to canonical short form when scope empty, otherwise preserve full prefix
	if scope == "" {
		return SpecID(project + "-" + code), nil
	}
	return SpecID(project + "-" + scope + "-" + code), nil
}

// Project returns the ProjectId component (e.g., KWF).
func (id SpecID) Project() string {
	s := string(id)
	parts := strings.Split(s, "-")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// Scope returns the Scope component or empty for short form.
func (id SpecID) Scope() string {
	s := string(id)
	parts := strings.Split(s, "-")
	if len(parts) == 3 {
		return parts[1]
	}
	return ""
}

// Code returns the 5-char code component.
func (id SpecID) Code() string {
	s := string(id)
	parts := strings.Split(s, "-")
	if len(parts) == 2 {
		return parts[1]
	}
	if len(parts) == 3 {
		return parts[2]
	}
	return ""
}

// RequirementID is a requirement identifier such as FRK-CLI-001 or KWL-CORE-001.
type RequirementID string

var reqRe = regexp.MustCompile(`^[A-Z]+-[A-Z]+-[0-9]{3,}$`)

// ParseRequirementID validates s as a RequirementID.
func ParseRequirementID(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errs.UsageError("requirement ID is required")
	}
	if !reqRe.MatchString(s) {
		return errs.UsageError(fmt.Sprintf("invalid requirement ID %q: want PREFIX-SCOPE-NNN", s))
	}
	return nil
}
