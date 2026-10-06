// Package version defines the kiw CLI version (module github.com/krewire/kiw).
package version

import "github.com/krewire/krewire/packages/kern"

// Version is the semantic version of the kiw devtool.
var Version = kern.MustParseVersion("0.1.0")

// VersionString returns the version as a string for UI display.
func VersionString() string { return Version.String() }

// EcosystemRequires declares the minimum version of each Krewire module this one
// was built against. This is the authoritative compatibility contract for kiw;
// `kiw compat` validates it against every other module's own declaration.
//
// The kernel names no modules, so this module states its dependencies by name.
// Everything kiw used to name individually — app, cloud, sec, ui, tui, runtime,
// i18n, assets, web — now lives inside libs and is covered by that one entry.
var EcosystemRequires = map[string]kern.Version{
	"boost":   kern.MustParseVersion("0.1.0"),
	"hub":     kern.MustParseVersion("0.1.0"),
	"kern":    kern.MustParseVersion("0.1.0"),
	"libs":    kern.MustParseVersion("0.1.0"),
	"mdbind":  kern.MustParseVersion("0.1.0"),
	"testing": kern.MustParseVersion("0.1.0"),
}
