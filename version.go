package krewire

import "github.com/krewire/krewire/packages/kern"

// Version is the master version of the Krewire monorepo and its constituent packages.
var Version = kern.MustParseVersion("0.1.0")

// VersionString returns the master version as a formatted semver string.
func VersionString() string {
	return Version.String()
}
