package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/version"
)

// captureStdout runs fn and returns what it printed to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = saved }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestRun_VersionReportsBuildIdentity(t *testing.T) {
	want := version.Build()
	for _, arg := range []string{"version", "--version", "-v"} {
		var code int
		out := captureStdout(t, func() { code = Run([]string{arg}) })
		if code != 0 {
			t.Fatalf("%s: exit code %d", arg, code)
		}
		for _, field := range []string{want.Version, want.Commit, want.Channel} {
			if !strings.Contains(out, field) {
				t.Fatalf("%s: output %q missing %q", arg, out, field)
			}
		}
	}
}

func TestRun_HelpListsVersion(t *testing.T) {
	out := captureStdout(t, func() { Run([]string{"--help"}) })
	if !strings.Contains(out, "--version") {
		t.Fatalf("help output does not document --version: %q", out)
	}
}
