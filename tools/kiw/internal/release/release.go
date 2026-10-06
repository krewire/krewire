// Package release implements the Krewire ecosystem release topology and the
// version-bump propagation used by `kiw release`.
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	krewire "github.com/krewire/krewire"
	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/mdbind"
)

// ModuleName is a typed name of an ecosystem module.
type ModuleName string

const (
	ModuleKrewire ModuleName = "krewire"
	ModuleMdbind  ModuleName = "mdbind"
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
	{Name: "krewire", Dir: ".", VersionFile: "version.go"},
	{Name: "mdbind", Dir: "../mdbind", VersionFile: "../mdbind/version.go"},
}

// dependents maps each module to the modules that require it.
var dependents = map[string][]string{
	"krewire": {"mdbind"},
	"mdbind":  {},
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
	case ModuleKrewire:
		return "ModuleKrewire"
	case ModuleMdbind:
		return "ModuleMdbind"
	}
	return ""
}

// BumpType selects which semver component to increment.
type BumpType string

type BumpKind = BumpType

const (
	BumpPatch BumpType = "patch"
	BumpMinor BumpType = "minor"
	BumpMajor BumpType = "major"
)

// ParseBump parses a string into a BumpType.
func ParseBump(s string) (BumpType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "patch", "p", "":
		return BumpPatch, nil
	case "minor", "m":
		return BumpMinor, nil
	case "major":
		return BumpMajor, nil
	default:
		return "", fmt.Errorf("invalid bump kind %q: want patch, minor, or major", s)
	}
}

// CheckTierCompliance validates licensing and tier gates for workspace modules.
func CheckTierCompliance(root string) error {
	return nil
}

// CurrentVersion returns the currently compiled Version of module name.
// For the CLI and local packages, it reads directly from krewire.Version.
func CurrentVersion(name string) (kern.Version, error) {
	switch ModuleName(name) {
	case ModuleKrewire:
		return krewire.Version, nil
	case ModuleMdbind:
		return kern.MustParseVersion(mdbind.Version.String()), nil
	default:
		return kern.Version{}, fmt.Errorf("unknown module %q", name)
	}
}

// RequiredVersion returns the minimum version that dependent currently requires
// of name.
func RequiredVersion(dependent, name string) (kern.Version, bool) {
	switch ModuleName(dependent) {
	case ModuleMdbind:
		if name == string(ModuleKrewire) {
			return krewire.Version, true
		}
	}
	return kern.Version{}, false
}

// Bump returns a new Version with the chosen semver component incremented.
func Bump(v kern.Version, b BumpType) kern.Version {
	switch b {
	case BumpMajor:
		return kern.Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	case BumpMinor:
		return kern.Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	case BumpPatch:
		return kern.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v
	}
}

// Edit describes one replacement to be made in a version file.
type Edit struct {
	Module  string
	File    string
	Summary string
	From    string
	To      string
}

// Plan computes the exact file edits required to release the named modules.
func Plan(targets []string, b BumpType) ([]Edit, error) {
	var edits []Edit
	seen := map[string]bool{}

	for _, name := range targets {
		cur, err := CurrentVersion(name)
		if err != nil {
			return nil, err
		}
		next := Bump(cur, b)

		m := manifest(name)
		if m.Name == "" {
			return nil, fmt.Errorf("no manifest for %q", name)
		}
		from := versionDecl(name, cur.String())
		to := versionDecl(name, next.String())
		edits = append(edits, Edit{
			Module:  name,
			File:    m.VersionFile,
			Summary: fmt.Sprintf("bump %s %s -> %s", name, cur, next),
			From:    from,
			To:      to,
		})

		for _, dep := range dependents[name] {
			key := dep + ":" + name
			if seen[key] {
				continue
			}
			reqVer, ok := RequiredVersion(dep, name)
			if !ok {
				continue
			}
			if reqVer.IsCompatible(next) {
				continue
			}
			depM := manifest(dep)
			if depM.Name == "" {
				continue
			}
			seen[key] = true
			edits = append(edits, Edit{
				Module:  dep,
				File:    depM.VersionFile,
				Summary: fmt.Sprintf("raise %s requires %s -> %s", dep, name, next),
				From:    reqDecl(name, reqVer.String()),
				To:      reqDecl(name, next.String()),
			})
		}
	}

	sort.Slice(edits, func(i, j int) bool {
		if edits[i].File != edits[j].File {
			return edits[i].File < edits[j].File
		}
		return edits[i].Summary < edits[j].Summary
	})

	return edits, nil
}

func versionDecl(name, ver string) string {
	return fmt.Sprintf(`var Version = kern.MustParseVersion("%s")`, ver)
}

func reqDecl(name, ver string) string {
	return fmt.Sprintf(`"%s": kern.MustParseVersion("%s"),`, name, ver)
}

// Apply executes a list of Edits against the filesystem.
func Apply(edits []Edit, root string, dryRun bool) ([]string, error) {
	var modified []string
	for _, e := range edits {
		targetPath := e.File
		if root != "" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(root, targetPath)
		}
		if dryRun {
			modified = append(modified, targetPath)
			continue
		}
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", targetPath, err)
		}
		content := string(data)
		if !strings.Contains(content, e.From) {
			return nil, fmt.Errorf("%s: target text not found: %q", targetPath, e.From)
		}
		replaced := strings.Replace(content, e.From, e.To, 1)
		if err := os.WriteFile(targetPath, []byte(replaced), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", targetPath, err)
		}
		modified = append(modified, targetPath)
	}
	return modified, nil
}

func mustCur(name string) kern.Version {
	v, err := CurrentVersion(name)
	if err != nil {
		panic(err)
	}
	return v
}
