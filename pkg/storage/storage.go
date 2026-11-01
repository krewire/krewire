// Package storage defines a small object-storage boundary and a local
// filesystem implementation. Consumers depend on Store rather than a backend.
package storage

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/krewire/krewire/pkg/file"
	kfs "github.com/krewire/krewire/pkg/fs"
)

// Store stores opaque objects addressed by a caller-defined key.
type Store interface {
	Put(key string, data []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Stat(key string) (ObjectInfo, error)
}

// StreamStore defines the streaming interface for chunked object storage.
type StreamStore interface {
	Store
	PutStream(key string, r io.Reader) error
	GetStream(key string) (io.ReadCloser, error)
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key  string
	Size int64
	Mode fs.FileMode
}

// Local stores objects below Root on a filesystem.
type Local struct {
	Root string
	FS   kfs.FileSystem
}

// NewLocal creates a local filesystem-backed Store.
func NewLocal(root string, filesystem kfs.FileSystem) (*Local, error) {
	if root == "" {
		return nil, fmt.Errorf("storage: root must not be empty")
	}
	if filesystem == nil {
		filesystem = kfs.Default
	}
	return &Local{Root: root, FS: filesystem}, nil
}

func (s *Local) path(key string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("storage: nil local store")
	}
	path, err := kfs.Clean(s.Root, key)
	if err != nil {
		return "", fmt.Errorf("storage: invalid key %q: %w", key, err)
	}
	return path, nil
}

// Put creates or replaces an object using an atomic file replacement.
func (s *Local) Put(key string, data []byte) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	return file.WriteAtomic(s.FS, path, data, 0o600)
}

// PutStream writes an object from a streaming reader without buffering all in memory.
func (s *Local) PutStream(key string, r io.Reader) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	return file.WriteStream(s.FS, path, r, 0o600)
}

// Get returns an object's complete contents.
func (s *Local) Get(key string) ([]byte, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return file.Read(s.FS, path)
}

// GetStream returns an io.ReadCloser for streaming object contents.
func (s *Local) GetStream(key string) (io.ReadCloser, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	rc, err := s.FS.Open(path)
	if err != nil {
		return nil, fmt.Errorf("storage: open stream %q: %w", key, err)
	}
	return rc, nil
}

// Delete removes an object.
func (s *Local) Delete(key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := s.FS.Remove(path); err != nil {
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	return nil
}

// Stat returns metadata for an object.
func (s *Local) Stat(key string) (ObjectInfo, error) {
	path, err := s.path(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	info, err := s.FS.Stat(path)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("storage: stat %q: %w", key, err)
	}
	return ObjectInfo{Key: filepath.ToSlash(key), Size: info.Size(), Mode: info.Mode()}, nil
}

var _ Store = (*Local)(nil)
var _ StreamStore = (*Local)(nil)
