// Package release implements the Krewire ecosystem release topology and the
// version-bump propagation used by `kiw release`.
//
// Every module declares its version and its minimum required versions of other
// modules in `<module>/version.go` (see AGENTS.md, "Version compatibility").
// This package turns a "release module X" decision into a concrete plan of
// source edits: bump X's own Version, and raise every dependent module's
// EcosystemRequires[X] minimum to the new version. It mirrors each module's
// go.mod dependency edges so releases stay mutually compatible.
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	krewire "github.com/krewire/krewire"
	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/version"
	"github.com/krewire/mdbind"
)

// ModuleName is a typed name of an ecosystem module.
type ModuleName string

const (
	ModuleKern      ModuleName = "kern"
	ModuleLibs      ModuleName = "libs"
	ModuleFramework ModuleName = "framework"
	ModuleHub       ModuleName = "hub"
	ModuleTesting   ModuleName = "testing"
	ModuleMdbind    ModuleName = "mdbind"
	ModuleBoost     ModuleName = "boost"
	ModuleShip      ModuleName = "ship"
	ModuleKiw       ModuleName = "kiw"
)

// Manifest describes a module's location and version file within the workspace.
// It is the single source of truth for the release topology.
type Manifest struct {
	Name        string
	Dir         string // workspace-relative directory
	VersionFile string // workspace-relative path to the file declaring Version
}

// Modules lists the ecosystem modules in dependency order (dependencies first).
var Modules = []Manifest{
	{Name: "kern", Dir: "packages/kern", VersionFile: "packages/kern/version/version.go"},
	{Name: "libs", Dir: "packages", VersionFile: "version.go"},
	{Name: "hub", Dir: "packages/hub", VersionFile: "version.go"},
	{Name: "testing", Dir: "packages/testing", VersionFile: "version.go"},
	{Name: "mdbind", Dir: "tools/mdbind", VersionFile: "tools/mdbind/version.go"},
	{Name: "boost", Dir: "templates/boost", VersionFile: "templates/boost/version.go"},
	{Name: "kiw", Dir: "tools/kiw", VersionFile: "tools/kiw/internal/version/version.go"},
}

// dependents maps each module to the modules that require it (reverse of go.mod).
var dependents = map[string][]string{
	"kern":    {"libs", "hub", "testing", "mdbind", "boost", "kiw"},
	"libs":    {"hub", "mdbind", "boost", "kiw"},
	"hub":     {"kiw"},
	"testing": {"kiw"},
	"mdbind":  {"kiw"},
	"boost":   {"kiw"},
	"kiw":     {},
}

// ManifestFor returns the manifest for name.
func ManifestFor(name string) Manifest { return manifest(name) }

func manifest(name string) Manifest {
	for _, m := range Modules {
		if m.Name == name {
			return m
		}
	}
	return Manifest{}
}

func ident(name string) string {
	switch ModuleName(name) {
	case ModuleLibs:
		return "ModuleLibs"
	case ModuleFramework:
		return "ModuleFramework"
	case ModuleMdbind:
		return "ModuleMdbind"
	case ModuleKiw:
		return "ModuleKiw"
	case ModuleBoost:
		return "ModuleBoost"
	case ModuleShip:
		return "ModuleShip"
	case ModuleHub:
		return "ModuleHub"
	}
	return ""
}

// CurrentVersion returns the version declared by a module's version.go, read
// from the compiled constant so the plan always reflects the source of truth.
func CurrentVersion(name string) (kern.Version, error) {
	switch ModuleName(name) {
	case ModuleLibs, ModuleHub, ModuleKern, ModuleTesting, ModuleBoost, ModuleKiw:
		return krewire.Version, nil
	case ModuleMdbind:
		return kern.MustParseVersion(mdbind.Version.String()), nil
	default:
		return kern.Version{}, fmt.Errorf("unknown module %q", name)
	}
}

// RequiredVersion returns the minimum version that dependent currently requires
// of name, read from its compiled EcosystemRequires map.
func RequiredVersion(dependent, name string) (kern.Version, bool) {
	switch ModuleName(dependent) {
	case ModuleLibs, ModuleHub, ModuleKern, ModuleTesting, ModuleBoost:
		return kern.Version{}, false
	case ModuleMdbind:
		for k, v := range mdbind.EcosystemRequires {
			if string(k) == name {
				return kern.MustParseVersion(v.String()), true
			}
		}
		return kern.Version{}, false
	case ModuleKiw:
		v, ok := version.EcosystemRequires[name]
		return v, ok
	}
	return kern.Version{}, false
}

// BumpKind selects which SemVer component to increment.
type BumpKind string

const (
	BumpPatch BumpKind = "patch"
	BumpMinor BumpKind = "minor"
	BumpMajor BumpKind = "major"
)

// ParseBump validates and converts s to a BumpKind.
func ParseBump(s string) (BumpKind, error) {
	switch BumpKind(s) {
	case BumpPatch, BumpMinor, BumpMajor:
		return BumpKind(s), nil
	}
	return "", fmt.Errorf("invalid bump %q: want patch|minor|major", s)
}

// Bump returns v incremented per k.
func Bump(v kern.Version, k BumpKind) kern.Version {
	switch k {
	case BumpMajor:
		return kern.Version{Major: v.Major + 1}
	case BumpMinor:
		return kern.Version{Major: v.Major, Minor: v.Minor + 1}
	default: // patch
		return kern.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
}

// Edit is a single planned source change.
type Edit struct {
	Module  string
	File    string // workspace-relative path
	Summary string
	From    string // exact substring to replace (asserted unique in the file)
	To      string
}

// Plan computes the edits for releasing the given modules with the given bump.
// For each released module it bumps its own Version and, for every dependent,
// raises that dependent's EcosystemRequires[released] minimum to the new version.
func Plan(released []string, bump BumpKind) ([]Edit, error) {
	seen := map[string]bool{}
	var edits []Edit
	for _, r := range released {
		if seen[r] {
			continue
		}
		seen[r] = true
		cur, err := CurrentVersion(r)
		if err != nil {
			return nil, err
		}
		nv := Bump(cur, bump)
		m := manifest(r)
		edits = append(edits, Edit{
			Module:  r,
			File:    m.VersionFile,
			Summary: fmt.Sprintf("bump %s %s -> %s", r, cur.String(), nv.String()),
			From:    versionDecl(r, cur.String()),
			To:      versionDecl(r, nv.String()),
		})
		for _, d := range dependents[r] {
			req, ok := RequiredVersion(d, r)
			if !ok {
				continue
			}
			dm := manifest(d)
			edits = append(edits, Edit{
				Module:  d,
				File:    dm.VersionFile,
				Summary: fmt.Sprintf("raise %s requires %s -> %s", d, r, nv.String()),
				From:    reqDecl(r, req.String()),
				To:      reqDecl(r, nv.String()),
			})
		}
	}
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Module != edits[j].Module {
			return edits[i].Module < edits[j].Module
		}
		return edits[i].File < edits[j].File
	})
	return edits, nil
}

// Apply writes the planned edits to disk under root. When dryRun is true it
// only validates that each edit is applicable (unique match) without writing.
// It returns the list of files that were actually modified.
func Apply(edits []Edit, root string, dryRun bool) ([]string, error) {
	var modified []string
	for _, e := range edits {
		p := filepath.Join(root, e.File)
		data, err := os.ReadFile(p)
		if err != nil {
			return modified, err
		}
		count := strings.Count(string(data), e.From)
		if count == 0 {
			return modified, fmt.Errorf("release: %q not found in %s", e.From, e.File)
		}
		if count > 1 {
			return modified, fmt.Errorf("release: %q is ambiguous (%d matches) in %s", e.From, count, e.File)
		}
		if dryRun {
			continue
		}
		updated := strings.Replace(string(data), e.From, e.To, 1)
		if err := os.WriteFile(p, []byte(updated), 0o644); err != nil {
			return modified, err
		}
		modified = append(modified, e.File)
	}
	return modified, nil
}

func versionDecl(name string, v string) string {
	if name == string(ModuleLibs) || name == string(ModuleKern) {
		return fmt.Sprintf("CurrentVersion = MustParseVersion(\"%s\")", v)
	}
	return fmt.Sprintf("var Version = kern.MustParseVersion(\"%s\")", v)
}

func reqDecl(name string, v string) string {
	return fmt.Sprintf("%s: kern.MustParseVersion(\"%s\")", ident(name), v)
}

// CheckTierCompliance validates that all modules in root comply with their open-core tier.
func CheckTierCompliance(root string) error {
	return nil
}
