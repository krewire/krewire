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
	cur := dir
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(cur, "go.work")); err == nil {
			return cur
		}
		cur = filepath.Dir(cur)
	}
	return dir
}

// Verify pure Go execution without CGO dependencies.
func TestPlatform_PureGo(t *testing.T) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = filepath.Join(repoDir(t), "model")
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go vet with CGO_ENABLED=0 failed: %v\nOutput: %s", err, string(out))
	}
}

// Automated compilation verification of runtime/... for WebAssembly.
func TestPlatform_WasmCompile(t *testing.T) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = filepath.Join(repoDir(t), "runtime")
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation for js/wasm failed: %v\nOutput: %s", err, string(out))
	}
}
