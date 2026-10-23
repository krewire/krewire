package krewire_test

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Verify pure Go execution without CGO dependencies.
func TestPlatform_PureGo(t *testing.T) {
	root := workspaceRoot(t)
	cmd := exec.Command("go", "vet", "./packages/...")
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go vet with CGO_ENABLED=0 failed: %v\nOutput: %s", err, string(out))
	}
}

// Cross-compilation build verification for tools/kiw on macOS and Windows.
func TestPlatform_CrossCompileKiw(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cross-compilation in short mode")
	}
	root := workspaceRoot(t)
	targets := []struct {
		goos   string
		goarch string
	}{
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}

	for _, tgt := range targets {
		t.Run(tgt.goos+"_"+tgt.goarch, func(t *testing.T) {
			outBin := filepath.Join(t.TempDir(), "kiw-"+tgt.goos+"-"+tgt.goarch)
			cmd := exec.Command("go", "build", "-o", outBin, "./tools/kiw/cmd/kiw")
			cmd.Dir = root
			cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0", "GOOS="+tgt.goos, "GOARCH="+tgt.goarch)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("cross-compile kiw for %s/%s failed: %v\nOutput: %s", tgt.goos, tgt.goarch, err, string(out))
			}
		})
	}
}

// Automated compilation verification of packages/runtime/... for WebAssembly.
func TestPlatform_WasmCompile(t *testing.T) {
	root := workspaceRoot(t)
	cmd := exec.Command("go", "build", "./packages/runtime/...")
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation for js/wasm failed: %v\nOutput: %s", err, string(out))
	}
}

// Automated verification that consecutive clean builds produce bitwise identical binaries.
func TestPlatform_BitwiseReproducibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bitwise reproducibility test in short mode")
	}
	root := workspaceRoot(t)
	tmpDir := t.TempDir()
	bin1 := filepath.Join(tmpDir, "kiw-1")
	bin2 := filepath.Join(tmpDir, "kiw-2")

	build := func(out string) []byte {
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", out, "./tools/kiw/cmd/kiw")
		cmd.Dir = root
		cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
		if outLog, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build failed: %v\nOutput: %s", err, string(outLog))
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read binary: %v", err)
		}
		sum := sha256.Sum256(data)
		return sum[:]
	}

	h1 := build(bin1)
	h2 := build(bin2)
	if !bytes.Equal(h1, h2) {
		t.Fatalf("build is not bitwise reproducible: hash1=%x, hash2=%x", h1, h2)
	}
}
