package krewire_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func repoDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "krewire", "go.mod")); err == nil {
		return filepath.Join(dir, "krewire")
	}
	return dir
}

// Verify pure Go execution without CGO dependencies.
func TestPlatform_PureGo(t *testing.T) {
	cmd := exec.Command("go", "vet", "./pkg/...")
	cmd.Dir = repoDir(t)
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go vet with CGO_ENABLED=0 failed: %v\nOutput: %s", err, string(out))
	}
}

// Automated compilation verification of pkg/runtime/... for WebAssembly.
func TestPlatform_WasmCompile(t *testing.T) {
	cmd := exec.Command("go", "build", "./pkg/runtime/...")
	cmd.Dir = repoDir(t)
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation for js/wasm failed: %v\nOutput: %s", err, string(out))
	}
}
