package model

import (
	"fmt"
	"github.com/krewire/krewire/packages/kern/errs"
	"path/filepath"
	"regexp"
	"strings"
)

// Project describes a Krewire project for validation.
type Project struct {
	Name       string `json:"name"`
	ModulePath string `json:"modulePath"`
	Kind       Kind   `json:"kind"`
	ConfigPath string `json:"configPath"`
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var moduleRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/\-]*$`)

// Validate checks project invariants.
func (p Project) Validate() error {
	if p.Name == "" {
		return errs.UsageError("project name is required")
	}
	if !nameRe.MatchString(p.Name) {
		return errs.UsageError(fmt.Sprintf("invalid project name %q: want kebab-case ^[a-z][a-z0-9-]*$", p.Name))
	}
	if p.ModulePath != "" {
		if !moduleRe.MatchString(p.ModulePath) {
			return errs.UsageError(fmt.Sprintf("invalid module path %q", p.ModulePath))
		}
		if err := validateModulePathSegments(p.ModulePath); err != nil {
			return err
		}
	}
	if !p.Kind.IsValid() {
		return errs.UsageError(fmt.Sprintf("invalid project kind %q", p.Kind))
	}
	if p.ConfigPath != "" {
		if err := ValidateKrewireYamlPath(p.ConfigPath); err != nil {
			return err
		}
	}
	return nil
}

// ValidateKrewireYamlPath ensures the config path is krewire.yaml.
func ValidateKrewireYamlPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errs.UsageError("config path is required")
	}
	// Normalize separators so Windows-style paths validate on any host, then
	// take the final element: krewire.yaml, ./krewire.yaml and /abs/krewire.yaml
	// are all accepted.
	base := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	if base != "krewire.yaml" {
		return errs.UsageError(fmt.Sprintf("invalid config path %q: must be krewire.yaml (no ssg.yaml)", path))
	}
	return nil
}

// validateModulePathSegments rejects module paths whose "/"-separated segments
// are empty or relative ("." / ".."). Without this, a module path such as
// "example.com/../../etc" would traverse outside its intended location if it is
// ever joined to a filesystem path (CWE-22).
func validateModulePathSegments(mp string) error {
	norm := strings.ReplaceAll(mp, "\\", "/")
	if strings.HasPrefix(norm, "/") || strings.HasSuffix(norm, "/") {
		return errs.UsageError(fmt.Sprintf("invalid module path %q: must not start or end with %q", mp, "/"))
	}
	for _, seg := range strings.Split(norm, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return errs.UsageError(fmt.Sprintf("invalid module path %q: empty, %q or %q segment", mp, ".", ".."))
		}
	}
	return nil
}

// OptInRule declares that some packages are opt-in batteries: a project of
// kind For may not import them directly, because they belong to the kind Owner
// and must cost the constrained kind nothing.
//
// The kernel defines the rule shape only. Which packages are batteries for which
// kind is ecosystem policy and lives above this layer, so adding a battery never
// edits the kernel. A rule never constrains its own Owner, so the kind that owns
// a battery may always import it.
//
// P is the caller's own import-path type; the constraint keeps the
// path-segment matching that stops "cloud/service" from also matching
// "cloud/serviceless".
type OptInRule[P ~string] struct {
	// For is the kind the rule constrains.
	For Kind
	// Owner is the kind the blocked packages belong to, and is exempt.
	Owner Kind
	// Blocked are the import paths the constrained kind may not import directly.
	Blocked []P
}

// ViolatesOptIn reports whether any path in imported breaks a rule declared for
// the given kind.
func ViolatesOptIn[P ~string](kind Kind, imported []P, rules []OptInRule[P]) bool {
	for _, imp := range imported {
		for _, r := range rules {
			if r.For != kind || r.Owner == kind {
				continue
			}
			for _, blocked := range r.Blocked {
				b := string(blocked)
				if string(imp) == b || strings.HasPrefix(string(imp), b+"/") {
					return true
				}
			}
		}
	}
	return false
}
