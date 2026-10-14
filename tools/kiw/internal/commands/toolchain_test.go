// Tests for KWN-TEST-P0FWA and KWL-L0-GOVUL
package commands

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
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
		{"KWN-P0FWA", true},
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

// Spec: KWN-P0FWA RND-TS-001 Scope: Unit
func TestKWN_P0FWA_RND_TS_001_ResolveModuleWalkingUpward(t *testing.T) {
	tmp := t.TempDir()
	goModContent := "module example.com/testwalk\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0o600); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(tmp, "sub", "deep", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	mod, err := gomod.Find(nested)
	if err != nil {
		t.Fatalf("expected to resolve module from %s: %v", nested, err)
	}
	if mod.Path != "example.com/testwalk" {
		t.Fatalf("got module path %q, want %q", mod.Path, "example.com/testwalk")
	}
}

// Spec: KWN-P0FWA RND-TS-002 Scope: Unit
func TestKWN_P0FWA_RND_TS_002_OutsideGoModuleExitCode(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	code := RunTest(nil)
	if code != kern.ExitCodeUsage {
		t.Fatalf("got exit code %v, want ExitCodeUsage (2)", code)
	}
}

// Spec: KWN-P0FWA RND-TS-003 RND-TS-004 RND-TS-006 RND-TS-007 Scope: Integration
func TestKWN_P0FWA_RunTest_Success(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()

	goMod := "module example.com/testpass\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}

	testCode := `package testpass

import "testing"

func TestSimplePass(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math broke")
	}
}
`
	if err := os.WriteFile(filepath.Join(tmp, "pass_test.go"), []byte(testCode), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	code := RunTest(nil)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("got exit code %v, want ExitCodeSuccess (0)", code)
	}
}

// Spec: KWN-P0FWA RND-TS-005 Scope: Integration
func TestKWN_P0FWA_RunTest_Failure(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()

	goMod := "module example.com/testfail\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}

	testCode := `package testfail

import "testing"

func TestSimpleFail(t *testing.T) {
	t.Fatal("deliberate failure for test")
}
`
	if err := os.WriteFile(filepath.Join(tmp, "fail_test.go"), []byte(testCode), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	code := RunTest(nil)
	if code != kern.ExitCodeFailure {
		t.Fatalf("got exit code %v, want ExitCodeFailure (1)", code)
	}
}

// Spec: KWL-L0-GOVUL REQ-L0-011 Scope: Service
func TestKWL_L0_GOVUL_REQ_L0_011_RunVetVuln(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()

	goMod := "module example.com/testvuln\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}

	mainCode := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(mainCode), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	fs := flag.NewFlagSet("vet", flag.ContinueOnError)
	RegisterVet(fs)
	_ = fs.Set("vuln", "true")

	code := RunVet(fs)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("got exit code %v, want ExitCodeSuccess (0)", code)
	}
}
