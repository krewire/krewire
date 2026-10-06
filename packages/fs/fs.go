// Package fs provides a small, testable filesystem boundary for Krewire
// packages. It complements the standard library's io/fs with write and
// mutation operations required by application code.
package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem is the filesystem contract used by file and storage packages.
type FileSystem interface {
	Open(name string) (io.ReadCloser, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	Stat(name string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	Rename(oldPath, newPath string) error
	Remove(name string) error
}

// OS is a FileSystem backed by the host operating system.
type OS struct{}

func (OS) Open(name string) (io.ReadCloser, error) { return os.Open(name) }
func (OS) ReadFile(name string) ([]byte, error)    { return os.ReadFile(name) }
func (OS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (OS) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (OS) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }
func (OS) Rename(oldPath, newPath string) error         { return os.Rename(oldPath, newPath) }
func (OS) Remove(name string) error                     { return os.Remove(name) }

// Lstat reports the file itself without following a final symlink, letting the
// file package refuse to write through a link.
func (OS) Lstat(name string) (fs.FileInfo, error) { return os.Lstat(name) }

// Chmod applies perm to an existing file, which os.WriteFile does not do.
func (OS) Chmod(name string, perm fs.FileMode) error { return os.Chmod(name, perm) }

// Default is the host filesystem implementation.
var Default FileSystem = OS{}

// Clean joins root and name while rejecting paths that escape root.
func Clean(root, name string) (string, error) {
	if root == "" || name == "" {
		return "", os.ErrInvalid
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", err
	}
	if rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", os.ErrPermission
	}
	return candidate, nil
}
