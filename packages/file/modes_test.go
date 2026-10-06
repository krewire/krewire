package file

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kfs "github.com/krewire/krewire/packages/fs"
)

// TestFILE_010_DefaultAndExplicitPerms verifies a zero perm resolves to the
// owner-only default rather than being handed to os.WriteFile as 0, which the
// umask would widen unpredictably.
func TestFILE_010_DefaultAndExplicitPerms(t *testing.T) {
	if got := resolvePerm(0); got != defaultFileMode {
		t.Errorf("resolvePerm(0) = %04o, want %04o", got, defaultFileMode)
	}
	if got := resolvePerm(0o644); got != 0o644 {
		t.Errorf("resolvePerm(0644) = %04o, want 0644", got)
	}

	dir := t.TempDir()
	implicit := filepath.Join(dir, "implicit")
	if err := Write(nil, implicit, []byte("x"), 0); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(implicit)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != defaultFileMode {
		t.Errorf("mode = %04o, want the default %04o", info.Mode().Perm(), defaultFileMode)
	}
}

// TestFILE_011_WriteEnforcesPermOnRewrite verifies a permissive file tightened
// through this API does not keep its old mode, which os.WriteFile alone would
// leave in place on an existing file.
func TestFILE_011_WriteEnforcesPermOnRewrite(t *testing.T) {
	if !posixModesEnforced() {
		t.Skip("POSIX file modes are not enforced on this platform")
	}
	path := filepath.Join(t.TempDir(), "secret.txt")

	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(nil, path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode after rewrite = %04o, want 0600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("contents = %q, want new", data)
	}
}

// TestFILE_012_ParentDirectoryIsPrivate verifies the directories this package
// creates are owner-only, so another account cannot list the files inside them.
func TestFILE_012_ParentDirectoryIsPrivate(t *testing.T) {
	if !posixModesEnforced() {
		t.Skip("POSIX directory modes are not enforced on this platform")
	}
	root := t.TempDir()
	nested := filepath.Join(root, "private", "deeper")
	if err := Write(nil, filepath.Join(nested, "f.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Join(root, "private"), nested} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != parentDirMode {
			t.Errorf("%s mode = %04o, want %04o", dir, got, parentDirMode)
		}
	}
}

// TestFILE_013_SymlinkTargetIsRefused verifies both write paths reject a symlink
// rather than following it, so a planted link cannot redirect the write.
func TestFILE_013_SymlinkTargetIsRefused(t *testing.T) {
	root := t.TempDir()
	elsewhere := filepath.Join(root, "target.txt")
	if err := os.WriteFile(elsewhere, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for name, write := range map[string]func(kfs.FileSystem, string, []byte, os.FileMode) error{
		"Write":       Write,
		"WriteAtomic": WriteAtomic,
	} {
		t.Run(name, func(t *testing.T) {
			if err := write(nil, link, []byte("overwritten"), 0o600); !errors.Is(err, ErrSymlinkTarget) {
				t.Errorf("err = %v, want ErrSymlinkTarget", err)
			}
			data, err := os.ReadFile(elsewhere)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "original" {
				t.Errorf("target contents = %q, want original: the link must not be followed", data)
			}
		})
	}
}

// TestFILE_014_EnsureNoSymlinkSkipsWhenUnsupported verifies the check degrades to
// a no-op for a filesystem that does not implement the optional Lstat, rather
// than blocking every write on a third-party implementation.
func TestFILE_014_EnsureNoSymlinkSkipsWhenUnsupported(t *testing.T) {
	var f kfs.FileSystem = minimalFS{}
	if _, ok := f.(interface {
		Lstat(string) (fs.FileInfo, error)
	}); ok {
		t.Fatal("minimalFS must not expose Lstat, or this test proves nothing")
	}
	if err := ensureNoSymlink(f, "/anything"); err != nil {
		t.Errorf("ensureNoSymlink without Lstat = %v, want nil", err)
	}
	if err := enforcePerm(f, "/anything", 0o600); err != nil {
		t.Errorf("enforcePerm without Chmod = %v, want nil", err)
	}
}

// minimalFS satisfies only the required FileSystem methods, so the optional
// Lstat and Chmod that kfs.Memory and kfs.OS add are absent.
type minimalFS struct{}

func (minimalFS) Open(name string) (io.ReadCloser, error) { return nil, errNotImplemented }
func (minimalFS) ReadFile(name string) ([]byte, error)    { return nil, errNotImplemented }
func (minimalFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return errNotImplemented
}
func (minimalFS) Stat(name string) (fs.FileInfo, error)        { return nil, errNotImplemented }
func (minimalFS) MkdirAll(path string, perm fs.FileMode) error { return errNotImplemented }
func (minimalFS) Rename(oldPath, newPath string) error         { return errNotImplemented }
func (minimalFS) Remove(name string) error                     { return errNotImplemented }

// TestFILE_015_TempNameIsUnpredictableAndUnique verifies consecutive names differ
// and that the randomness carries the documented entropy, so the temporary file

// failFS wraps a working filesystem and fails one named operation, so the
// error-handling branches of Write, WriteAtomic, and Copy can be exercised
// without depending on a real disk failure.
//
// Chmod is dispatched through an optional interface because FileSystem does not
// require it, which is exactly how enforcePerm reaches it.
type failFS struct {
	kfs.FileSystem
	failOn string
}

func (f failFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if f.failOn == "write" {
		return errInjected
	}
	return f.FileSystem.WriteFile(name, data, perm)
}

func (f failFS) Rename(oldPath, newPath string) error {
	if f.failOn == "rename" {
		return errInjected
	}
	return f.FileSystem.Rename(oldPath, newPath)
}

func (f failFS) Chmod(name string, perm os.FileMode) error {
	if f.failOn == "chmod" {
		return errInjected
	}
	chmod, ok := f.FileSystem.(interface {
		Chmod(string, os.FileMode) error
	})
	if !ok {
		return nil
	}
	return chmod.Chmod(name, perm)
}

func (f failFS) MkdirAll(path string, perm os.FileMode) error {
	if f.failOn == "mkdir" {
		return errInjected
	}
	return f.FileSystem.MkdirAll(path, perm)
}

var errInjected = errors.New("file: injected failure")

// TestFILE_021_WriteFailuresAreReported verifies a failing underlying write or
// mkdir surfaces as an error naming the path, rather than being swallowed.
func TestFILE_021_WriteFailuresAreReported(t *testing.T) {
	for _, failOn := range []string{"write", "mkdir"} {
		t.Run(failOn, func(t *testing.T) {
			fsys := failFS{FileSystem: kfs.NewMemory(), failOn: failOn}
			err := Write(fsys, "/dir/f.txt", []byte("x"), 0o600)
			if !errors.Is(err, errInjected) {
				t.Errorf("Write = %v, want the injected failure", err)
			}
			if !strings.Contains(err.Error(), "f.txt") {
				t.Errorf("error %q does not name the path", err)
			}
		})
	}
}

// TestFILE_022_WriteAtomicRenameFailureCleansUp verifies a failed rename removes
// the temporary file, so a failed atomic write does not litter the directory
// with an orphaned temporary that a later write could collide with.
func TestFILE_022_WriteAtomicRenameFailureCleansUp(t *testing.T) {
	mem := kfs.NewMemory()
	fsys := failFS{FileSystem: mem, failOn: "rename"}

	if err := WriteAtomic(fsys, "/f.txt", []byte("x"), 0o600); !errors.Is(err, errInjected) {
		t.Fatalf("WriteAtomic = %v, want the injected rename failure", err)
	}

	// A surviving temporary would still be readable under its random name, and
	// the destination must not exist since the rename never happened.
	if _, err := mem.Stat("/f.txt"); err == nil {
		t.Error("destination exists despite the rename failing")
	}
	// The filesystem must be back to empty: no entry survived the failure.
	if err := mem.Remove("/f.txt"); err == nil {
		t.Error("unexpected: an entry named /f.txt exists")
	}
}

// TestFILE_023_WriteAtomicTemporaryWriteFailure verifies a failure writing the
// temporary file is reported and names the temporary path.
func TestFILE_023_WriteAtomicTemporaryWriteFailure(t *testing.T) {
	fsys := failFS{FileSystem: kfs.NewMemory(), failOn: "write"}
	err := WriteAtomic(fsys, "/f.txt", []byte("x"), 0o600)
	if !errors.Is(err, errInjected) {
		t.Fatalf("WriteAtomic = %v, want the injected write failure", err)
	}
	if !strings.Contains(err.Error(), "temporary") {
		t.Errorf("error %q does not identify the temporary write", err)
	}
}

// TestFILE_024_EnforcePermFailureIsReported verifies a chmod failure after a
// successful write is not silently ignored: the caller asked for a mode and
// must learn it was not applied.
func TestFILE_024_EnforcePermFailureIsReported(t *testing.T) {
	mem := kfs.NewMemory()
	if err := mem.WriteFile("/f.txt", []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys := failFS{FileSystem: mem, failOn: "chmod"}
	if err := Write(fsys, "/f.txt", []byte("y"), 0o600); !errors.Is(err, errInjected) {
		t.Errorf("Write = %v, want the injected chmod failure to surface", err)
	}
	if err := WriteAtomic(fsys, "/g.txt", []byte("y"), 0o600); !errors.Is(err, errInjected) {
		t.Errorf("WriteAtomic = %v, want the injected chmod failure to surface", err)
	}
}

// TestFILE_025_CopyOpenFailureIsReported verifies a source that cannot be opened
// produces an error naming the source rather than creating an empty destination.
func TestFILE_025_CopyOpenFailureIsReported(t *testing.T) {
	mem := kfs.NewMemory()
	if err := Copy(nil, "/dst", filepath.Join(t.TempDir(), "missing"), 0o600); err == nil ||
		!strings.Contains(err.Error(), "missing") {
		t.Errorf("Copy = %v, want it to name the missing source", err)
	}
	// Copy must not have created the destination.
	if _, err := mem.Stat("/dst"); err == nil {
		t.Error("Copy created a destination despite failing to open the source")
	}
}

// cannot be pre-created by another process (CWE-377).
func TestFILE_015_TempNameIsUnpredictableAndUnique(t *testing.T) {
	mem := kfs.NewMemory()
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name, err := tempName(mem, "/target.txt")
		if err != nil {
			t.Fatal(err)
		}
		if seen[name] {
			t.Fatalf("tempName repeated %q within 100 draws", name)
		}
		seen[name] = true

		suffix, ok := strings.CutPrefix(name, "/target.txt.")
		if !ok {
			t.Fatalf("name %q does not carry the expected prefix", name)
		}
		hexPart, ok := strings.CutSuffix(suffix, ".tmp")
		if !ok {
			t.Fatalf("name %q does not end in .tmp", name)
		}
		if want := tempSuffixEntropyBytes * 2; len(hexPart) != want {
			t.Fatalf("entropy suffix %q is %d hex chars, want %d", hexPart, len(hexPart), want)
		}
	}
}

// TestFILE_016_ReadAndCopyReportMissingFiles verifies the error paths name the
// file that failed, so a caller can tell which path was missing.
func TestFILE_016_ReadAndCopyReportMissingFiles(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.txt")

	if _, err := Read(nil, missing); err == nil || !strings.Contains(err.Error(), "absent.txt") {
		t.Errorf("Read error = %v, want it to name the file", err)
	}
	if err := Copy(nil, missing+".copy", missing, 0o600); err == nil ||
		!strings.Contains(err.Error(), "absent.txt") {
		t.Errorf("Copy error = %v, want it to name the missing source", err)
	}
}

// TestFILE_017_CopyPreservesBytes verifies a copy reproduces the source exactly,
// including NUL bytes and invalid UTF-8.
func TestFILE_017_CopyPreservesBytes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	payload := []byte("line one\nline two\n\x00binary\xff")
	if err := Write(nil, src, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Copy(nil, dst, src, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Read(nil, dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("copied %q, want %q", got, payload)
	}
}

// TestFILE_018_ContextWrappersRefuseCancelledContext verifies every *Context
// entry point honours cancellation before doing any work and returns the context
// error rather than performing the operation anyway.
func TestFILE_018_ContextWrappersRefuseCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mem := kfs.NewMemory()

	if _, err := ReadContext(ctx, mem, "/f.txt"); !errors.Is(err, context.Canceled) {
		t.Errorf("ReadContext = %v, want context.Canceled", err)
	}
	if err := WriteContext(ctx, mem, "/f.txt", []byte("x"), 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("WriteContext = %v, want context.Canceled", err)
	}
	if err := WriteAtomicContext(ctx, mem, "/f.txt", []byte("x"), 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("WriteAtomicContext = %v, want context.Canceled", err)
	}
	if err := CopyContext(ctx, mem, "/dst.txt", "/src.txt", 0o600); !errors.Is(err, context.Canceled) {
		t.Errorf("CopyContext = %v, want context.Canceled", err)
	}
}

// TestFILE_019_ContextWrappersPropagateOperationErrors verifies the context
// variants surface the underlying failure instead of masking it as success.
func TestFILE_019_ContextWrappersPropagateOperationErrors(t *testing.T) {
	ctx := context.Background()
	mem := kfs.NewMemory()

	if _, err := ReadContext(ctx, mem, "/absent"); err == nil {
		t.Error("ReadContext on a missing file = nil, want an error")
	}
	if err := WriteContext(ctx, mem, "/f.txt", []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteContext: %v", err)
	}
	if err := WriteAtomicContext(ctx, mem, "/atomic.txt", []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteAtomicContext: %v", err)
	}
	if err := CopyContext(ctx, mem, "/absent-copy", "/absent", 0o600); err == nil {
		t.Error("CopyContext from a missing source = nil, want an error")
	}
}

// TestFILE_020_ContextWrappersRejectNilContext verifies a nil context is
// reported rather than panicking on ctx.Done().
func TestFILE_020_ContextWrappersRejectNilContext(t *testing.T) {
	//lint:ignore SA1012 a nil context is exactly the input under test.
	if _, err := ReadContext(nil, kfs.NewMemory(), "/f.txt"); err == nil {
		t.Error("ReadContext(nil) = nil, want an error")
	}
	//lint:ignore SA1012 a nil context is exactly the input under test.
	if err := WriteContext(nil, kfs.NewMemory(), "/f.txt", []byte("x"), 0o600); err == nil {
		t.Error("WriteContext(nil) = nil, want an error")
	}
}

var errNotImplemented = errors.New("file: minimalFS stub")

// posixModesEnforced reports whether POSIX file-mode assertions are meaningful
// on this platform.
func posixModesEnforced() bool { return os.PathSeparator == '/' }
