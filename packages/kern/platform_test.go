// Tests for KWL-L0-MTPAT and KWL-L0-REPRO
package kern

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Spec: KWL-L0-MTPAT REQ-L0-001 Scope: Domain
// Verify pure Go execution without CGO dependencies.
func TestKWL_L0_MTPAT_REQ_L0_001_PureGo(t *testing.T) {
	root := workspaceRoot(t)
	cmd := exec.Command("go", "vet", "./packages/kern/...")
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go vet with CGO_ENABLED=0 failed: %v\nOutput: %s", err, string(out))
	}
}

// Spec: KWL-L0-MTPAT REQ-L0-002 Scope: Service
// Cross-compilation build verification for tools/kiw on macOS and Windows.
func TestKWL_L0_MTPAT_REQ_L0_002_CrossCompileKiw(t *testing.T) {
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

// Spec: KWL-L0-MTPAT REQ-L0-003 Scope: Domain
// Automated compilation verification of packages/runtime/... for WebAssembly.
func TestKWL_L0_MTPAT_REQ_L0_003_WasmCompile(t *testing.T) {
	root := workspaceRoot(t)
	cmd := exec.Command("go", "build", "./packages/runtime/...")
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation for js/wasm failed: %v\nOutput: %s", err, string(out))
	}
}

// Spec: KWL-L0-REPRO REQ-L0-024 Scope: Monorepo
// Automated verification that consecutive clean builds produce bitwise identical binaries.
func TestKWL_L0_REPRO_REQ_L0_024_BitwiseReproducibility(t *testing.T) {
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
