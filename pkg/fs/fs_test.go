package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanRejectsEscapingPath(t *testing.T) {
	root := t.TempDir()
	if _, err := Clean(root, "../outside.txt"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("Clean error = %v, want permission error", err)
	}
}

// filepathHasPrefix reports whether path stays inside root.
func filepathHasPrefix(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) &&
		(len(rel) < 3 || rel[:3] != ".."+string(filepath.Separator))
}

// TestCleanRejectsEmptyInput covers the guard clauses that keep a blank root or
// name from being joined into a surprising path.
func TestCleanRejectsEmptyInput(t *testing.T) {
	if _, err := Clean("", "file.txt"); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("Clean with empty root = %v, want invalid", err)
	}
	if _, err := Clean("/srv", ""); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("Clean with empty name = %v, want invalid", err)
	}
}

// TestCleanTraversalForms exercises the shapes an attacker uses to climb out of
// a root. Each must be refused rather than silently normalized inside.
func TestCleanTraversalForms(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"../outside.txt",
		"../../outside.txt",
		"nested/../../outside.txt",
		"./../outside.txt",
	} {
		if got, err := Clean(root, name); err == nil {
			t.Errorf("Clean(root, %q) = %q, want rejection", name, got)
		}
	}
}

// TestCleanAcceptsInteriorDotSegments verifies traversal rejection does not
// reject harmless relative paths that stay inside the root.
func TestCleanAcceptsInteriorDotSegments(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"file.txt",
		"a/b/c.txt",
		"./file.txt",
		"a/./b.txt",
	} {
		got, err := Clean(root, name)
		if err != nil {
			t.Errorf("Clean(root, %q) unexpected error %v", name, err)
			continue
		}
		if !filepathHasPrefix(got, root) {
			t.Errorf("Clean(root, %q) = %q, escaped root %q", name, got, root)
		}
	}
}

// TestOSFilesystem exercises every method of the OS implementation against a
// real temporary directory, so the contract the file and storage packages rely
// on is covered end to end.
func TestOSFilesystem(t *testing.T) {
	dir := t.TempDir()
	var f FileSystem = OS{}
	path := filepath.Join(dir, "nested", "data.txt")

	if err := f.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := f.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := f.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("ReadFile = %q", got)
	}

	rc, err := f.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read open handle: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("Open contents = %q", body)
	}

	info, err := f.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != int64(len("hello")) {
		t.Errorf("Stat size = %d", info.Size())
	}

	// Lstat and Chmod are the optional capabilities the file package probes for.
	lstater, ok := f.(interface {
		Lstat(string) (os.FileInfo, error)
	})
	if !ok {
		t.Fatal("OS must implement Lstat so writes can refuse symlink targets")
	}
	if _, err := lstater.Lstat(path); err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	chmod, ok := f.(interface {
		Chmod(string, os.FileMode) error
	})
	if !ok {
		t.Fatal("OS must implement Chmod so the requested mode is enforced")
	}
	if err := chmod.Chmod(path, 0o600); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	moved := filepath.Join(dir, "moved.txt")
	if err := f.Rename(path, moved); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := f.Stat(path); err == nil {
		t.Error("source must be gone after Rename")
	}
	if err := f.Remove(moved); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := f.Remove(moved); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Remove of missing file = %v, want not-exist", err)
	}
}

// TestMemoryFilesystem exercises the in-memory implementation against the same
// contract as OS, so tests can swap backends without changing behaviour.
func TestMemoryFilesystem(t *testing.T) {
	var f FileSystem = NewMemory()

	// Every accessor must report a missing entry rather than panic.
	if _, err := f.ReadFile("absent.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadFile missing = %v, want not-exist", err)
	}
	if _, err := f.Stat("absent.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat missing = %v, want not-exist", err)
	}
	if _, err := f.Open("absent.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open missing = %v, want not-exist", err)
	}
	if err := f.Remove("absent.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Remove missing = %v, want not-exist", err)
	}
	if err := f.Rename("absent.txt", "other.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Rename missing = %v, want not-exist", err)
	}
	if err := f.MkdirAll("any", 0o700); err != nil {
		t.Errorf("MkdirAll is a no-op and must not fail: %v", err)
	}

	if err := f.WriteFile("a.txt", []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	rc, err := f.Open("a.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := io.ReadAll(rc); err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	info, err := f.Stat("a.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Name() != "a.txt" || info.Size() != 4 || info.IsDir() {
		t.Errorf("Stat info = %q size=%d isDir=%v", info.Name(), info.Size(), info.IsDir())
	}
	if !info.ModTime().IsZero() || info.Sys() != nil {
		t.Error("Memory FileInfo must report a zero ModTime and nil Sys")
	}

	// Chmod updates the recorded mode so a rewrite applies the new permissions.
	chmod, ok := f.(interface {
		Chmod(string, os.FileMode) error
	})
	if !ok {
		t.Fatal("Memory must implement Chmod")
	}
	if err := chmod.Chmod("a.txt", 0o644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	after, err := f.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != 0o644 {
		t.Errorf("mode after Chmod = %v, want 0644", after.Mode().Perm())
	}
	if err := chmod.Chmod("absent.txt", 0o600); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Chmod missing = %v, want not-exist", err)
	}

	// Lstat mirrors Stat because the in-memory filesystem has no links.
	lstater, ok := f.(interface {
		Lstat(string) (os.FileInfo, error)
	})
	if !ok {
		t.Fatal("Memory must implement Lstat")
	}
	if _, err := lstater.Lstat("a.txt"); err != nil {
		t.Fatalf("Lstat: %v", err)
	}

	if err := f.Rename("a.txt", "b.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := f.Stat("b.txt"); err != nil {
		t.Errorf("destination must exist after Rename: %v", err)
	}
	if _, err := f.Stat("a.txt"); err == nil {
		t.Error("source must be gone after Rename")
	}

	// ReadFile must return a copy, so a caller cannot mutate stored state.
	data, err := f.ReadFile("b.txt")
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	again, err := f.ReadFile("b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != "data" {
		t.Errorf("stored data mutated through the returned slice: %q", again)
	}

	if err := f.Remove("b.txt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

// TestMemoryZeroValueIsUsable verifies a Memory used without NewMemory does not
// panic, since the type is exported as a struct.
func TestMemoryZeroValueIsUsable(t *testing.T) {
	var m Memory
	if err := m.WriteFile("x.txt", []byte("v"), 0o600); err != nil {
		t.Fatalf("WriteFile on zero value: %v", err)
	}
	got, err := m.ReadFile("x.txt")
	if err != nil {
		t.Fatalf("ReadFile on zero value: %v", err)
	}
	if string(got) != "v" {
		t.Errorf("contents = %q", got)
	}
}
