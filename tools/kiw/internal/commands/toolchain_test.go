// Tests for KWN-TEST-P0FWA
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec: KWN-TEST-P0FWA Scope: Unit
func TestIsSpecID_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"KWF-TEST-M4P9Q", true},
		{"KWL-ARCH-J2K9Q", true},
		{"KWN-DEVTOOL-Z0VFC", true},
		{"TestKWF", false},
		{"M4P9Q", false},
		{"KWF-TEST", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isSpecID(c.in); got != c.want {
			t.Errorf("isSpecID(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestSpecFilterRegexp_Valid(t *testing.T) {
	if got := specFilterRegexp("KWF-TEST-M4P9Q"); got != "M4P" {
		t.Errorf("specFilterRegexp KWF-TEST-M4P9Q = %q want M4P", got)
	}
	if got := specFilterRegexp("KWL-ARCH-J2K9Q"); got != "J2K" {
		t.Errorf("specFilterRegexp KWL-ARCH-J2K9Q = %q want J2K", got)
	}
}

func TestIsPackageFilter_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"./web", true},
		{"github.com/krewire/krewire/packages/web", true},
		{"web_test.go", true},
		{"TestKWF", false},
		{"M4P", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isPackageFilter(c.in); got != c.want {
			t.Errorf("isPackageFilter(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestIsKnownTestFlag_Valid(t *testing.T) {
	if !isKnownTestFlag("--filter") {
		t.Error("isKnownTestFlag --filter should be true")
	}
	if !isKnownTestFlag("-watch") {
		t.Error("isKnownTestFlag -watch should be true")
	}
	if isKnownTestFlag("--unknown") {
		t.Error("isKnownTestFlag --unknown should be false")
	}
}

func TestAuditSecurityPosture(t *testing.T) {
	tmp := t.TempDir()
	cleanYAML := `project:
  name: safe-app
  kind: app
auth:
  secret: "${env:APP_SECRET}"
`
	if err := os.WriteFile(filepath.Join(tmp, "krewire.yaml"), []byte(cleanYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	issues := auditSecurityPosture(tmp)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for clean config, got: %v", issues)
	}

	leakyYAML := `project:
  name: leaky-app
auth:
  password: supersecretpassword
`
	if err := os.WriteFile(filepath.Join(tmp, "krewire.yaml"), []byte(leakyYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	issues = auditSecurityPosture(tmp)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for leaky config, got: %v", issues)
	}
}
