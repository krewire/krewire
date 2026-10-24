package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
)

func TestIsPackageFilter_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"./web", true},
		{"github.com/krewire/krewire/pkg/web", true},
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

func TestResolveModuleWalkingUpward(t *testing.T) {
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

func TestOutsideGoModuleExitCode(t *testing.T) {
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
	if code != errs.ExitCodeUsage {
		t.Fatalf("got exit code %v, want ExitCodeUsage (2)", code)
	}
}

func TestRunTest_Success(t *testing.T) {
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
	if code != errs.ExitCodeSuccess {
		t.Fatalf("got exit code %v, want ExitCodeSuccess (0)", code)
	}
}

func TestRunTest_Failure(t *testing.T) {
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
	if code != errs.ExitCodeFailure {
		t.Fatalf("got exit code %v, want ExitCodeFailure (1)", code)
	}
}
