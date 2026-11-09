// Package release implements the Krewire ecosystem release topology and the
// version-bump propagation used by `kiw release`.
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	krewire "github.com/krewire/krewire"
	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/version"
)

// ModuleName is a typed name of an ecosystem module.
type ModuleName string

const (
	ModuleKrewire ModuleName = "krewire"
	ModuleKrew    ModuleName = "krew"
	ModuleCloud   ModuleName = "cloud"
	ModuleAuth    ModuleName = "auth"
	ModuleDash    ModuleName = "dash"
	ModuleForge   ModuleName = "forge"
)

// Manifest describes a module's location and version file within the workspace.
// It is the single source of truth for the release topology.
type Manifest struct {
	Name        string
	Dir         string // workspace-relative directory
	VersionFile string // module-relative path to the file declaring Version
	Tier        model.Tier
}

// Modules lists the ecosystem modules in dependency order (dependencies first).
var Modules = []Manifest{
	{Name: "krewire", Dir: "krewire", VersionFile: "version.go", Tier: model.TierFree},
	{Name: "krew", Dir: "krew", VersionFile: "version.go", Tier: model.TierFree},
	{Name: "auth", Dir: "auth", VersionFile: "version.go", Tier: model.TierFree},
	{Name: "cloud", Dir: "cloud", VersionFile: "version.go", Tier: model.TierFree},
	{Name: "dash", Dir: "dash", VersionFile: "version.go", Tier: model.TierFree},
	{Name: "forge", Dir: "forge", VersionFile: "version.go", Tier: model.TierFree},
}

// dependents maps each module to the modules that require it.
var dependents = map[string][]string{
	"krewire": {"krew", "auth", "cloud", "dash", "forge"},
	"krew":    {"dash"},
	"auth":    {"cloud", "dash", "forge"},
	"cloud":   {},
	"dash":    {},
	"forge":   {},
}

// DefaultRequirements maps each module to the static default versions of ecosystem modules it requires.
var DefaultRequirements = map[string]map[string]version.Version{
	"krewire": {},
	"krew": {
		"krewire": krewire.Version,
	},
	"auth": {
		"krewire": krewire.Version,
	},
	"cloud": {
		"krewire": krewire.Version,
		"auth":    krewire.Version,
	},
	"dash": {
		"krewire": krewire.Version,
		"krew":    krewire.Version,
		"auth":    krewire.Version,
	},
	"forge": {
		"krewire": krewire.Version,
		"auth":    krewire.Version,
	},
}

var (
	versionDeclRe = regexp.MustCompile(`var\s+Version\s*=\s*version\.MustParseVersion\("([^"]+)"\)`)
	reqItemRe     = regexp.MustCompile(`"([a-zA-Z0-9_-]+)"\s*:\s*version\.MustParseVersion\("([^"]+)"\)`)
)

// FindWorkspaceRoot walks up from working directory or executable location to locate go.work.
func FindWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.work not found")
}

// resolveModuleVersionPath finds the path to the version file for the named module.
func resolveModuleVersionPath(root, module string) (string, error) {
	m := manifest(module)
	if m.Name == "" {
		return "", fmt.Errorf("unknown module %q", module)
	}
	candidates := []string{
		filepath.Join(root, m.Dir, m.VersionFile),
		filepath.Join(root, m.Name, m.VersionFile),
		filepath.Join(root, m.VersionFile),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("version file not found for module %q in workspace %s", module, root)
}

// ReadModuleVersion parses and returns the Version declared in module's version.go.
func ReadModuleVersion(root, module string) (version.Version, error) {
	path, err := resolveModuleVersionPath(root, module)
	if err != nil {
		return version.Version{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return version.Version{}, err
	}
	matches := versionDeclRe.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		return version.Version{}, fmt.Errorf("could not find Version declaration in %s", path)
	}
	return version.ParseVersion(matches[1])
}

// ReadModuleRequirements parses EcosystemRequires from module's version.go.
func ReadModuleRequirements(root, module string) (map[string]version.Version, error) {
	path, err := resolveModuleVersionPath(root, module)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	reqs := make(map[string]version.Version)
	matches := reqItemRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			v, err := version.ParseVersion(m[2])
			if err == nil {
				reqs[m[1]] = v
			}
		}
	}
	return reqs, nil
}

// ReadEcosystemActualVersions parses and returns the current versions of all workspace modules.
func ReadEcosystemActualVersions(root string) (map[string]version.Version, error) {
	actual := make(map[string]version.Version)
	for _, m := range Modules {
		v, err := ReadModuleVersion(root, m.Name)
		if err == nil {
			actual[m.Name] = v
		} else {
			// fallback to CurrentVersion
			if cv, cerr := CurrentVersion(m.Name); cerr == nil {
				actual[m.Name] = cv
			}
		}
	}
	return actual, nil
}

// ReadEcosystemRequirements parses and returns the declared EcosystemRequires for all workspace modules.
func ReadEcosystemRequirements(root string) (map[string]map[string]version.Version, error) {
	allReqs := make(map[string]map[string]version.Version)
	for _, m := range Modules {
		reqs, err := ReadModuleRequirements(root, m.Name)
		if err == nil {
			allReqs[m.Name] = reqs
		} else if def, ok := DefaultRequirements[m.Name]; ok {
			allReqs[m.Name] = def
		} else {
			allReqs[m.Name] = make(map[string]version.Version)
		}
	}
	return allReqs, nil
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
	case ModuleKrew:
		return "ModuleKrew"
	case ModuleCloud:
		return "ModuleCloud"
	case ModuleAuth:
		return "ModuleAuth"
	case ModuleDash:
		return "ModuleDash"
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

// CurrentTier returns the declared Tier of module name.
func CurrentTier(name string) (model.Tier, error) {
	switch ModuleName(name) {
	case ModuleKrewire:
		return krewire.Tier, nil
	case ModuleKrew, ModuleCloud, ModuleAuth, ModuleDash, ModuleForge:
		return model.TierFree, nil
	default:
		return "", fmt.Errorf("unknown module %q", name)
	}
}

// CheckTierCompliance validates licensing and tier gates for workspace modules.
// All code in krewire/krewire is 100% open source. This gate ensures modules declare
// valid tiers (currently free-tier) and preserve their open source licensing.
func CheckTierCompliance(root string) error {
	for _, m := range Modules {
		if !m.Tier.IsValid() {
			return fmt.Errorf("module %q has invalid tier %q: want free, pro, team, or enterprise", m.Name, m.Tier)
		}
		curTier, err := CurrentTier(m.Name)
		if err == nil && curTier != m.Tier {
			return fmt.Errorf("module %q manifest tier %q does not match compiled tier %q", m.Name, m.Tier, curTier)
		}
		if root != "" {
			licPath := filepath.Join(root, m.Dir, "LICENSE")
			if _, err := os.Stat(licPath); os.IsNotExist(err) {
				altPath := filepath.Join(root, m.Name, "LICENSE")
				if _, aerr := os.Stat(altPath); aerr == nil {
					licPath = altPath
				}
			}
			if data, err := os.ReadFile(licPath); err == nil {
				content := string(data)
				if !strings.Contains(content, "MIT License") && !strings.Contains(content, "Permission is hereby granted") {
					return fmt.Errorf("module %q license at %s must be open source", m.Name, licPath)
				}
			}
		}
	}
	return nil
}

// CurrentVersion returns the currently compiled or declared Version of module name.
func CurrentVersion(name string) (version.Version, error) {
	m := manifest(name)
	if m.Name == "" {
		return version.Version{}, fmt.Errorf("unknown module %q", name)
	}
	if root, err := FindWorkspaceRoot(); err == nil {
		if v, err := ReadModuleVersion(root, name); err == nil {
			return v, nil
		}
	}
	switch ModuleName(name) {
	case ModuleKrewire:
		return krewire.Version, nil
	case ModuleKrew, ModuleCloud, ModuleAuth, ModuleDash, ModuleForge:
		return krewire.Version, nil
	default:
		return version.Version{}, fmt.Errorf("unknown module %q", name)
	}
}

// RequiredVersion returns the minimum version that dependent currently requires
// of name.
func RequiredVersion(dependent, name string) (version.Version, bool) {
	if root, err := FindWorkspaceRoot(); err == nil {
		if reqs, err := ReadModuleRequirements(root, dependent); err == nil {
			if reqVer, ok := reqs[name]; ok {
				return reqVer, true
			}
		}
	}
	if reqs, ok := DefaultRequirements[dependent]; ok {
		if reqVer, ok := reqs[name]; ok {
			return reqVer, true
		}
	}
	return version.Version{}, false
}

// Bump returns a new Version with the chosen semver component incremented.
func Bump(v version.Version, b BumpType) version.Version {
	switch b {
	case BumpMajor:
		return version.Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	case BumpMinor:
		return version.Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	case BumpPatch:
		return version.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
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
			if next.IsCompatible(reqVer) {
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
	return fmt.Sprintf(`var Version = version.MustParseVersion("%s")`, ver)
}

func reqDecl(name, ver string) string {
	return fmt.Sprintf(`"%s": version.MustParseVersion("%s"),`, name, ver)
}

// Apply executes a list of Edits against the filesystem.
func Apply(edits []Edit, root string, dryRun bool) ([]string, error) {
	var modified []string
	for _, e := range edits {
		targetPath := e.File
		if root != "" && !filepath.IsAbs(targetPath) {
			m := manifest(e.Module)
			candidate := filepath.Join(root, targetPath)
			if m.Dir != "" {
				dirCandidate := filepath.Join(root, m.Dir, e.File)
				if _, err := os.Stat(dirCandidate); err == nil {
					candidate = dirCandidate
				}
			}
			if _, err := os.Stat(candidate); os.IsNotExist(err) && e.Module != "" {
				altPath := filepath.Join(root, e.Module, e.File)
				if _, aerr := os.Stat(altPath); aerr == nil {
					candidate = altPath
				}
			}
			targetPath = candidate
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
		var replaced string
		if strings.Contains(content, e.From) {
			replaced = strings.Replace(content, e.From, e.To, 1)
		} else {
			// Try flexible matching for map entries where gofmt aligned colons with multiple spaces/tabs
			words := strings.Fields(e.From)
			if len(words) >= 2 {
				pattern := regexp.QuoteMeta(words[0]) + `\s+` + regexp.QuoteMeta(strings.Join(words[1:], " "))
				re, err := regexp.Compile(pattern)
				if err == nil && re.MatchString(content) {
					replaced = re.ReplaceAllString(content, e.To)
				}
			}
			if replaced == "" {
				return nil, fmt.Errorf("%s: target text not found: %q", targetPath, e.From)
			}
		}
		if err := os.WriteFile(targetPath, []byte(replaced), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", targetPath, err)
		}
		modified = append(modified, targetPath)
	}
	return modified, nil
}

func mustCur(name string) version.Version {
	v, err := CurrentVersion(name)
	if err != nil {
		panic(err)
	}
	return v
}
