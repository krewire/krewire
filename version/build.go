package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Release-pipeline link-time variables. The release workflow stamps them with
//
//	-ldflags "-X github.com/krewire/krewire/version.buildVersion=v0.2.0
//	          -X github.com/krewire/krewire/version.buildCommit=<sha>
//	          -X github.com/krewire/krewire/version.buildDate=<RFC3339>
//	          -X github.com/krewire/krewire/version.buildChannel=stable"
//
// Unstamped (local `go build`) binaries fall back to CurrentVersion and the VCS
// metadata the Go toolchain embeds via -buildvcs.
var (
	buildVersion = ""
	buildCommit  = ""
	buildDate    = ""
	buildChannel = ""
)

// BuildInfo describes the exact release identity of a compiled binary.
type BuildInfo struct {
	Version   string // release version, e.g. v0.2.0 or v0.3.0-nightly.20261123
	Commit    string // full source commit SHA, or "unknown"
	Date      string // commit/build timestamp (RFC 3339), or "unknown"
	Channel   string // release channel: stable, lts, rc, beta, alpha, nightly, experimental, dev
	GoVersion string // Go toolchain that produced the binary
	Platform  string // GOOS/GOARCH
	Modified  bool   // true when built from a dirty working tree
}

// Build returns the build identity of the running binary.
func Build() BuildInfo {
	info := BuildInfo{
		Version:   buildVersion,
		Commit:    buildCommit,
		Date:      buildDate,
		Channel:   buildChannel,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		applyVCS(&info, bi.Settings)
	}
	if info.Version == "" {
		info.Version = "v" + CurrentVersion.String()
	}
	if info.Channel == "" {
		info.Channel = "dev"
	}
	if info.Commit == "" {
		info.Commit = "unknown"
	}
	if info.Date == "" {
		info.Date = "unknown"
	}
	return info
}

// applyVCS fills fields that were not stamped at link time from the VCS
// settings recorded by the Go toolchain.
func applyVCS(info *BuildInfo, settings []debug.BuildSetting) {
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == "" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.Date == "" {
				info.Date = s.Value
			}
		case "vcs.modified":
			info.Modified = s.Value == "true"
		}
	}
}

// String renders a single-line, human-readable build identity suitable for
// `--version` output.
func (b BuildInfo) String() string {
	dirty := ""
	if b.Modified {
		dirty = "-dirty"
	}
	return fmt.Sprintf("%s (channel %s, commit %s%s, built %s, %s, %s)",
		b.Version, b.Channel, b.Commit, dirty, b.Date, b.GoVersion, b.Platform)
}
