package krewire

import "github.com/krewire/krewire/packages/kern"

// Version is the master version of the Krewire monorepo and its constituent packages.
var Version = kern.MustParseVersion("0.1.0")

// Tier is the default ecosystem tier assigned to the krewire monorepo.
// All code in krewire/krewire is 100% open source; all packages are currently
// free-tier by default.
var Tier = kern.TierFree

// VersionString returns the master version as a formatted semver string.
func VersionString() string {
	return Version.String()
}

// TierString returns the monorepo tier as a string.
func TierString() string {
	return Tier.String()
}
