// Package buildinfo resolves module versions for the devtool: from Go build
// metadata first, falling back to the master Krewire version.
package buildinfo

import (
	"runtime/debug"
	"strings"

	krewire "github.com/krewire/krewire"
)

const (
	ModKrewire = "github.com/krewire/krewire"
)

// DevelVersion is the version Go records for modules built from source
// rather than a released tag.
const DevelVersion = "(devel)"

// ModuleVersion returns the version of the module at path as recorded in the
// build info, or "" when the module is not part of the build.
func ModuleVersion(path string) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if bi.Main.Path == path {
		return bi.Main.Version
	}
	for _, dep := range bi.Deps {
		if dep.Path == path {
			return dep.Version
		}
	}
	return ""
}

// KnownVersion returns the master version declared by the Krewire monorepo.
func KnownVersion(path string) string {
	if path == ModKrewire || path == "" {
		return krewire.VersionString()
	}
	return ""
}

// ResolveVersion returns the effective version of the module at path.
func ResolveVersion(path string) (version string, fromSource bool) {
	v := ModuleVersion(path)
	if v != "" && v != DevelVersion && !strings.HasPrefix(v, "v0.0.0-") && !strings.Contains(v, "+dirty") {
		return v, false
	}
	if known := KnownVersion(path); known != "" {
		return known, true
	}
	return "", false
}
