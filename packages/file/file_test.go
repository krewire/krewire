package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/fs"
)

// TestWriteAtomicReplacesContents verifies an atomic write replaces prior
// contents through a temporary file and a rename.
func TestWriteAtomicReplacesContents(t *testing.T) {
	path := t.TempDir() + "/nested/data.txt"
	if err := WriteAtomic(fs.Default, path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(fs.Default, path, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Read(fs.Default, path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Fatalf("contents = %q, want second", got)
	}
}

// TestWrite_CreatesParentDirectories verifies Write materializes the parent
// directory tree rather than failing on a missing intermediate path.
func TestWrite_CreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c.txt")
	if err := Write(fs.Default, path, []byte("deep"), 0o600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(fs.Default, path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "deep" {
		t.Errorf("contents = %q", got)
	}
}

// TestDefaultModes verifies a zero perm falls back to the package default
// instead of producing a world-readable file, on both write paths.
func TestDefaultModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes")
	}
	dir := t.TempDir()
	for name, write := range map[string]func(string, []byte, os.FileMode) error{
		"write":        func(p string, d []byte, _ os.FileMode) error { return Write(fs.Default, p, d, 0) },
		"write-atomic": func(p string, d []byte, _ os.FileMode) error { return WriteAtomic(fs.Default, p, d, 0) },
	} {
		path := filepath.Join(dir, name+".txt")
		if err := write(path, []byte("x"), 0); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != defaultFileMode {
			t.Errorf("%s default mode = %v, want %v", name, got, os.FileMode(defaultFileMode))
		}
	}
}

// TestCopy_CopiesContents verifies Copy duplicates bytes to the destination and
// leaves the source untouched.
func TestCopy_CopiesContents(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := Write(fs.Default, src, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Copy(fs.Default, dst, src, 0o600); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := Read(fs.Default, dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Errorf("copied contents = %q", got)
	}
	orig, err := Read(fs.Default, src)
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != "payload" {
		t.Errorf("source changed to %q", orig)
	}
}

// TestCopy_MissingSource verifies a missing source reports an error rather than
// silently creating an empty destination.
func TestCopy_MissingSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "absent.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := Copy(fs.Default, dst, src, 0o600); err == nil {
		t.Fatal("Copy must fail when the source is missing")
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Error("Copy must not create the destination after a failed read")
	}
}

// TestRead_WrapsErrorWithPath verifies errors name the offending path so a caller
// can tell which file failed.
func TestRead_WrapsErrorWithPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	if _, err := Read(fs.Default, missing); err == nil || !strings.Contains(err.Error(), "missing.txt") {
		t.Errorf("Read error = %v, want it to name the path", err)
	}
}

// TestOperations_WorkWithMemoryFilesystem verifies the operations are backend
// agnostic, which is what lets tests substitute an in-memory filesystem.
func TestOperations_WorkWithMemoryFilesystem(t *testing.T) {
	mem := fs.NewMemory()
	if err := WriteAtomic(mem, "/cfg/data.json", []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	got, err := Read(mem, "/cfg/data.json")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Errorf("contents = %q", got)
	}
	if err := Copy(mem, "/cfg/copy.json", "/cfg/data.json", 0o600); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	copied, err := Read(mem, "/cfg/copy.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != `{"a":1}` {
		t.Errorf("copied contents = %q", copied)
	}
	if err := Write(mem, "/cfg/data.json", []byte("v2"), 0o600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	rewritten, err := Read(mem, "/cfg/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(rewritten) != "v2" {
		t.Errorf("rewritten contents = %q", rewritten)
	}
}

// TestContextOperations verifies the Context variants: they must refuse to run
// once the context is canceled, must succeed when it is live, and must report a
// nil context instead of panicking.
func TestContextOperations(t *testing.T) {
	mem := fs.NewMemory()
	if err := Write(mem, "/seed.txt", []byte("v"), 0o600); err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := ReadContext(canceled, mem, "/seed.txt"); !errors.Is(err, context.Canceled) {
		t.Errorf("ReadContext = %v, want canceled", err)
	}
	if err := WriteContext(canceled, mem, "/w.txt", []byte("v"), 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("WriteContext = %v, want canceled", err)
	}
	if err := WriteAtomicContext(canceled, mem, "/wa.txt", []byte("v"), 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("WriteAtomicContext = %v, want canceled", err)
	}
	if err := CopyContext(canceled, mem, "/c.txt", "/seed.txt", 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("CopyContext = %v, want canceled", err)
	}

	live := context.Background()
	if err := WriteContext(live, mem, "/seed.txt", []byte("v"), 0o600); err != nil {
		t.Fatalf("WriteContext: %v", err)
	}
	got, err := ReadContext(live, mem, "/seed.txt")
	if err != nil {
		t.Fatalf("ReadContext: %v", err)
	}
	if string(got) != "v" {
		t.Errorf("contents = %q", got)
	}
	if err := WriteAtomicContext(live, mem, "/atomic.txt", []byte("a"), 0o600); err != nil {
		t.Fatalf("WriteAtomicContext: %v", err)
	}
	if err := CopyContext(live, mem, "/copy.txt", "/seed.txt", 0o600); err != nil {
		t.Fatalf("CopyContext: %v", err)
	}

	//nolint:staticcheck // a nil context is the behaviour under test
	if _, err := ReadContext(nil, mem, "/seed.txt"); err == nil {
		t.Error("ReadContext(nil) must return an error rather than panic")
	}
}

// TestTempName_IsDistinctFromOccupiedNames verifies the generated temporary name
// does not collide with an entry that already exists.
func TestTempName_IsDistinctFromOccupiedNames(t *testing.T) {
	mem := fs.NewMemory()
	first, err := tempName(mem, "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.WriteFile(first, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := tempName(mem, "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Error("tempName must not return an already-occupied name")
	}
}

// TestSEC_FILE_001_Write_RefusesSymlinkTarget verifies that a write does not
// follow a symlink planted at the destination. Following it would let an
// attacker who can create a file in a writable directory overwrite an arbitrary
// file elsewhere on the system (CWE-59).
func TestSEC_FILE_001_Write_RefusesSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("ORIGINAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "target.txt")
	if err := os.Symlink(victim, dst); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	err := Write(nil, dst, []byte("OVERWRITTEN"), 0o600)
	if err == nil {
		t.Fatal("Write must reject a symlink target")
	}
	if !errors.Is(err, ErrSymlinkTarget) {
		t.Errorf("error = %v, want ErrSymlinkTarget", err)
	}
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("symlink target was modified: %q", got)
	}
}

// TestSEC_FILE_002_WriteAtomic_RefusesSymlinkTarget is the atomic variant.
func TestSEC_FILE_002_WriteAtomic_RefusesSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("ORIGINAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "atomic.txt")
	if err := os.Symlink(victim, dst); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if err := WriteAtomic(nil, dst, []byte("OVERWRITTEN"), 0o600); err == nil {
		t.Fatal("WriteAtomic must reject a symlink target")
	}
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("symlink target was modified: %q", got)
	}
}

// TestSEC_FILE_003_Write_EnforcesPermOnExistingFile verifies that the requested
// mode is applied even when the file already exists. os.WriteFile ignores perm
// for an existing file, which let a 0644 file stay world-readable after a
// rewrite that requested 0600.
func TestSEC_FILE_003_Write_EnforcesPermOnExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "perm.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(nil, path, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %v, want 0600", got)
	}
}

// TestSEC_FILE_004_WriteAtomic_UnpredictableTempName verifies the temporary
// file name carries cryptographic randomness. A timestamp-derived name is
// predictable, so another process could pre-create it as a symlink and capture
// the write.
func TestSEC_FILE_004_WriteAtomic_UnpredictableTempName(t *testing.T) {
	dir := t.TempDir()
	seen := map[string]bool{}
	for i := 0; i < 32; i++ {
		name, err := tempName(fs.Default, filepath.Join(dir, "data.json"))
		if err != nil {
			t.Fatal(err)
		}
		if seen[name] {
			t.Fatalf("temporary name repeated: %s", name)
		}
		seen[name] = true
		suffix := strings.TrimPrefix(filepath.Base(name), "data.json.")
		suffix = strings.TrimSuffix(suffix, ".tmp")
		if len(suffix) != 32 {
			t.Errorf("temporary name carries %d hex chars, want 32: %s", len(suffix), name)
		}
	}
}
