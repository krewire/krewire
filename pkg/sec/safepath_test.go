package sec_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/sec"
)

func TestSafePath_ValidCases(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(filepath.Join(baseDir, "uploads", "avatars"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple relative filename",
			input:    "profile.png",
			expected: filepath.Join(baseDir, "profile.png"),
		},
		{
			name:     "nested relative path",
			input:    "uploads/avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "path with self-references",
			input:    "./uploads/./avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "path with benign backward resolution inside base",
			input:    "uploads/sub/../avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "leading slash treated as relative to base",
			input:    "/uploads/avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sec.SafePath(baseDir, tc.input)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestSafePath_TraversalAttacks(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	attacks := []struct {
		name  string
		input string
	}{
		{name: "parent step-out", input: "../secret.txt"},
		{name: "multi parent step-out", input: "../../../etc/passwd"},
		{name: "nested step-out", input: "uploads/../../secret.txt"},
		{name: "deeply nested step-out", input: "a/b/c/../../../../etc/shadow"},
		{name: "backslashes on windows/cross-platform", input: "..\\..\\etc\\passwd"},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sec.SafePath(baseDir, tc.input)
			if err == nil {
				t.Fatalf("expected error for traversal attack %q, got path %q", tc.input, got)
			}
			if !errors.Is(err, sec.ErrPathTraversal) {
				t.Errorf("expected ErrPathTraversal, got: %v", err)
			}
		})
	}
}

func TestSafePath_InvalidParameters(t *testing.T) {
	_, err := sec.SafePath("", "file.txt")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for empty base, got: %v", err)
	}

	_, err = sec.SafePath("/some/dir", "")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for empty target, got: %v", err)
	}

	_, err = sec.SafePath("   ", "file.txt")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for whitespace base, got: %v", err)
	}

	_, err = sec.SafePath(t.TempDir(), "sub/file\x00.txt")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for null byte in target, got: %v", err)
	}
}

func TestSafePath_SymlinkEscapes(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outsideDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(outsideDir, "secret.key")
	if err := os.WriteFile(outsideFile, []byte("super-secret-key"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. Existing file outside symlinked inside
	symlinkToFile := filepath.Join(baseDir, "leaked_secret.txt")
	if err := os.Symlink(outsideFile, symlinkToFile); err != nil {
		t.Fatal(err)
	}
	_, err := sec.SafePath(baseDir, "leaked_secret.txt")
	if err == nil || !errors.Is(err, sec.ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal for symlinked external file, got: %v", err)
	}

	// 2. Directory outside symlinked inside
	symlinkToDir := filepath.Join(baseDir, "ext_dir")
	if err := os.Symlink(outsideDir, symlinkToDir); err != nil {
		t.Fatal(err)
	}

	// Existing target inside symlinked dir
	_, err = sec.SafePath(baseDir, "ext_dir/secret.key")
	if err == nil || !errors.Is(err, sec.ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal for existing file inside external symlink dir, got: %v", err)
	}

	// Non-existent target inside symlinked dir
	_, err = sec.SafePath(baseDir, "ext_dir/new_exploit.txt")
	if err == nil || !errors.Is(err, sec.ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal for non-existent target inside external symlink dir, got: %v", err)
	}

	// 3. Benign symlink inside workspace should be permitted
	internalRealDir := filepath.Join(baseDir, "real_internal")
	if err := os.MkdirAll(internalRealDir, 0o755); err != nil {
		t.Fatal(err)
	}
	internalFile := filepath.Join(internalRealDir, "internal.txt")
	if err := os.WriteFile(internalFile, []byte("internal"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkInternal := filepath.Join(baseDir, "internal_link")
	if err := os.Symlink(internalRealDir, symlinkInternal); err != nil {
		t.Fatal(err)
	}

	resolved, err := sec.SafePath(baseDir, "internal_link/internal.txt")
	if err != nil {
		t.Fatalf("expected internal symlink to succeed, got: %v", err)
	}
	realTarget, _ := filepath.EvalSymlinks(internalFile)
	if resolved != realTarget {
		t.Errorf("expected %q, got %q", realTarget, resolved)
	}
}

func TestSafePath_HostPathRedactionInErrors(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "secret-workspace-directory")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := sec.SafePath(baseDir, "../../../etc/shadow")
	if err == nil {
		t.Fatal("expected traversal error")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, baseDir) {
		t.Errorf("error message leaks sensitive host base directory: %s", errMsg)
	}
}

func TestSafeWorkspace_LifecycleAndConfinement(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Fail closed on empty or invalid workspace
	if _, err := sec.OpenSafeWorkspace(""); !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for empty workspace root, got: %v", err)
	}
	if _, err := sec.OpenSafeWorkspace("/path/does/not/exist/9999"); err == nil {
		t.Error("expected error for non-existent workspace root")
	}

	ws, err := sec.OpenSafeWorkspace(root)
	if err != nil {
		t.Fatalf("unexpected error opening workspace: %v", err)
	}
	defer ws.Close()

	// 2. Write file
	if err := ws.WriteFile("nested/test.txt", []byte("hello safe world"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// 3. Read file
	data, err := ws.ReadFile("nested/test.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "hello safe world" {
		t.Errorf("unexpected content: %s", string(data))
	}

	// 4. Edit file
	if err := ws.EditFile("nested/test.txt", "safe", "secure"); err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	data, _ = ws.ReadFile("nested/test.txt")
	if string(data) != "hello secure world" {
		t.Errorf("unexpected edited content: %s", string(data))
	}

	// 5. Read directory
	entries, err := ws.ReadDir("nested")
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "test.txt" {
		t.Errorf("unexpected entries: %+v", entries)
	}

	// 6. Delete file
	if err := ws.Remove("nested/test.txt"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if _, err := ws.ReadFile("nested/test.txt"); err == nil {
		t.Error("expected read to fail after removal")
	}

	// 7. Traversal attempts via SafeWorkspace
	if err := ws.WriteFile("../escape.txt", []byte("bad"), 0o644); !errors.Is(err, sec.ErrPathTraversal) {
		t.Errorf("expected ErrPathTraversal for relative escape, got: %v", err)
	}
	if _, err := ws.ReadFile("/etc/passwd"); err == nil {
		// Scoped into root, which does not exist in root
		if !os.IsNotExist(err) && !errors.Is(err, sec.ErrPathTraversal) {
			t.Errorf("expected NotExist or PathTraversal, got: %v", err)
		}
	}
}

func TestSafeWorkspace_SymlinkSwapRace(t *testing.T) {
	wsDir := filepath.Join(t.TempDir(), "race-ws")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outsideDir := filepath.Join(t.TempDir(), "race-outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outsideTargetFile := filepath.Join(outsideDir, "pwned.txt")

	ws, err := sec.OpenSafeWorkspace(wsDir)
	if err != nil {
		t.Fatalf("failed to open safe workspace: %v", err)
	}
	defer ws.Close()

	swapDir := filepath.Join(wsDir, "target_swap")
	realDir := filepath.Join(wsDir, "real_swap_dir")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Symlink(realDir, swapDir)

	stop := make(chan struct{})
	done := make(chan struct{})

	// Swapper goroutine rapidly flips target_swap between benign internal dir and dangerous outside dir
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_ = os.Remove(swapDir)
				_ = os.Symlink(outsideDir, swapDir)
				_ = os.Remove(swapDir)
				_ = os.Symlink(realDir, swapDir)
			}
		}
	}()

	// Worker goroutine attempts rapid writes
	for i := 0; i < 200; i++ {
		_ = ws.WriteFile("target_swap/pwned.txt", []byte("malicious content"), 0o644)
	}

	close(stop)
	<-done

	// Verify that the outside target was NEVER touched!
	if _, err := os.Stat(outsideTargetFile); !os.IsNotExist(err) {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY: symlink swap race escaped workspace and created %s", outsideTargetFile)
	}
}
