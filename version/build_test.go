package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuild_DefaultsWhenUnstamped(t *testing.T) {
	b := Build()
	if b.Version != "v"+CurrentVersion.String() && b.Version != buildVersion {
		t.Fatalf("unexpected version %q", b.Version)
	}
	if b.Channel == "" || b.Commit == "" || b.Date == "" {
		t.Fatalf("empty fields in %+v", b)
	}
	if !strings.Contains(b.Platform, "/") {
		t.Fatalf("platform %q is not GOOS/GOARCH", b.Platform)
	}
}

func TestBuild_LinkTimeStampWins(t *testing.T) {
	saved := [4]string{buildVersion, buildCommit, buildDate, buildChannel}
	t.Cleanup(func() {
		buildVersion, buildCommit, buildDate, buildChannel = saved[0], saved[1], saved[2], saved[3]
	})
	buildVersion, buildCommit, buildDate, buildChannel = "v9.9.9-rc.1", "abc123", "2026-11-23T00:00:00Z", "rc"

	b := Build()
	if b.Version != "v9.9.9-rc.1" || b.Commit != "abc123" || b.Channel != "rc" || b.Date != "2026-11-23T00:00:00Z" {
		t.Fatalf("stamped values not reported: %+v", b)
	}
	s := b.String()
	for _, want := range []string{"v9.9.9-rc.1", "channel rc", "commit abc123"} {
		if !strings.Contains(s, want) {
			t.Fatalf("String() = %q, missing %q", s, want)
		}
	}
}

func TestApplyVCS(t *testing.T) {
	var b BuildInfo
	applyVCS(&b, []debug.BuildSetting{
		{Key: "vcs.revision", Value: "deadbeef"},
		{Key: "vcs.time", Value: "2026-11-22T10:00:00Z"},
		{Key: "vcs.modified", Value: "true"},
	})
	if b.Commit != "deadbeef" || b.Date != "2026-11-22T10:00:00Z" || !b.Modified {
		t.Fatalf("vcs settings not applied: %+v", b)
	}
	if !strings.Contains(b.String(), "deadbeef-dirty") {
		t.Fatalf("dirty marker missing: %q", b.String())
	}

	stamped := BuildInfo{Commit: "stamped"}
	applyVCS(&stamped, []debug.BuildSetting{{Key: "vcs.revision", Value: "other"}})
	if stamped.Commit != "stamped" {
		t.Fatalf("vcs overrode link-time commit: %+v", stamped)
	}
}
