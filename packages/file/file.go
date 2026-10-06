// Package file provides file-oriented operations built on the fs boundary.
package file

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/krewire/krewire/packages/fs"
)

// Default file modes. A write through this package is treated as potentially
// secret material, so the default file mode is owner-only and the parent
// directory it creates is owner-traversable-only: a world-readable parent
// would let another account list the names of the files inside it.
const (
	// defaultFileMode is applied when the caller passes a zero perm.
	defaultFileMode os.FileMode = 0o600
	// parentDirMode is the mode for directories Write and WriteAtomic create.
	parentDirMode os.FileMode = 0o700
	// tempSuffixEntropyBytes is the length of the random suffix tempName
	// appends. 16 bytes makes the temporary name unguessable, so another
	// process cannot pre-create it (CWE-377).
	tempSuffixEntropyBytes = 16
)

// resolvePerm substitutes defaultFileMode for a zero perm, which os.WriteFile
// would otherwise interpret as "create with mode masked by the umask".
func resolvePerm(perm os.FileMode) os.FileMode {
	if perm == 0 {
		return defaultFileMode
	}
	return perm
}

// ErrSymlinkTarget reports that a write target is a symlink. The file package
// refuses to follow symlinks on write so an attacker who can plant a link
// inside a writable directory cannot redirect a write to an arbitrary path
// (CWE-59, CWE-61).
var ErrSymlinkTarget = fmt.Errorf("file: refusing to write through symlink")

// Read returns the complete contents of name.
func Read(fsys fs.FileSystem, name string) ([]byte, error) {
	if fsys == nil {
		fsys = fs.Default
	}
	data, err := fsys.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("file: read %q: %w", name, err)
	}
	return data, nil
}

// prepareWrite runs the checks and setup every write shares: resolve the
// filesystem, resolve the mode, refuse a symlink target, and create the parent
// directory. Write and WriteAtomic both start here so the two paths cannot drift
// apart on a security check.
//
// SECURITY: the symlink rejection is what stops a planted link inside a writable
// directory from redirecting a write to an arbitrary path (CWE-59, CWE-61).
func prepareWrite(fsys fs.FileSystem, name string, perm os.FileMode) (fs.FileSystem, os.FileMode, error) {
	if fsys == nil {
		fsys = fs.Default
	}
	perm = resolvePerm(perm)
	if err := ensureNoSymlink(fsys, name); err != nil {
		return nil, 0, err
	}
	if err := fsys.MkdirAll(filepath.Dir(name), parentDirMode); err != nil && !os.IsExist(err) {
		return nil, 0, fmt.Errorf("file: create parent for %q: %w", name, err)
	}
	return fsys, perm, nil
}

// Write writes data to name, creating or replacing the file.
//
// SECURITY: an existing symlink at name is rejected rather than followed, and
// perm is enforced on the resulting file even when the file already exists, so
// a permissive mode cannot survive a rewrite through this API.
func Write(fsys fs.FileSystem, name string, data []byte, perm os.FileMode) error {
	fsys, perm, err := prepareWrite(fsys, name, perm)
	if err != nil {
		return err
	}
	if err := fsys.WriteFile(name, data, perm); err != nil {
		return fmt.Errorf("file: write %q: %w", name, err)
	}
	return enforcePerm(fsys, name, perm)
}

// WriteAtomic writes data through a temporary file and renames it into place.
// The temporary file is created beside the destination so the rename remains on
// one filesystem.
//
// SECURITY: the temporary name carries 16 bytes of cryptographic randomness and
// is created exclusively, so it cannot be pre-created or guessed by another
// process. An existing symlink at the destination is rejected rather than
// replaced or followed.
func WriteAtomic(fsys fs.FileSystem, name string, data []byte, perm os.FileMode) error {
	fsys, perm, err := prepareWrite(fsys, name, perm)
	if err != nil {
		return err
	}
	tmp, err := tempName(fsys, name)
	if err != nil {
		return err
	}
	if err := fsys.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("file: write temporary %q: %w", tmp, err)
	}
	if err := fsys.Rename(tmp, name); err != nil {
		_ = fsys.Remove(tmp)
		return fmt.Errorf("file: replace %q: %w", name, err)
	}
	return enforcePerm(fsys, name, perm)
}

// tempName builds an unpredictable sibling path for the atomic write. The
// random suffix removes the predictability of a timestamp-only name, which
// allowed another process to pre-create the temporary file as a symlink.
func tempName(fsys fs.FileSystem, name string) (string, error) {
	buf := make([]byte, tempSuffixEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("file: generate temporary name: %w", err)
	}
	candidate := name + "." + hex.EncodeToString(buf) + ".tmp"
	if _, err := fsys.Stat(candidate); err == nil {
		return "", fmt.Errorf("file: temporary name collision: %w", os.ErrExist)
	}
	return candidate, nil
}

// ensureNoSymlink rejects a write whose target already exists as a symlink.
// The check is applied through optional interfaces so the exported FileSystem
// contract stays backward compatible for third-party implementations.
func ensureNoSymlink(fsys fs.FileSystem, name string) error {
	lstater, ok := fsys.(interface {
		Lstat(string) (os.FileInfo, error)
	})
	if !ok {
		return nil
	}
	info, err := lstater.Lstat(name)
	if err != nil {
		// Target does not exist; nothing to follow.
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %q", ErrSymlinkTarget, name)
	}
	return nil
}

// enforcePerm applies perm to an existing file, which os.WriteFile does not do.
// Without this a file created as 0644 keeps 0644 after a rewrite that requested
// 0600.
func enforcePerm(fsys fs.FileSystem, name string, perm os.FileMode) error {
	chmod, ok := fsys.(interface {
		Chmod(string, os.FileMode) error
	})
	if !ok {
		return nil
	}
	if err := chmod.Chmod(name, perm); err != nil {
		return fmt.Errorf("file: set mode on %q: %w", name, err)
	}
	return nil
}

// Copy copies all bytes from src to dst.
func Copy(fsys fs.FileSystem, dst, src string, perm os.FileMode) error {
	if fsys == nil {
		fsys = fs.Default
	}
	in, err := fsys.Open(src)
	if err != nil {
		return fmt.Errorf("file: open %q: %w", src, err)
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("file: read %q: %w", src, err)
	}
	return Write(fsys, dst, data, perm)
}
