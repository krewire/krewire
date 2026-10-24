package fs

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Memory struct {
	mu    sync.RWMutex
	files map[string]memoryFile
}

type memoryFile struct {
	data []byte
	mode fs.FileMode
}

// NewMemory returns an empty in-memory filesystem.
func NewMemory() *Memory { return &Memory{files: make(map[string]memoryFile)} }

func (m *Memory) ensure() {
	if m.files == nil {
		m.files = make(map[string]memoryFile)
	}
}
func (m *Memory) Open(name string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), f.data...))), nil
}
func (m *Memory) ReadFile(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), f.data...), nil
}
func (m *Memory) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	m.files[name] = memoryFile{data: append([]byte(nil), data...), mode: perm}
	return nil
}
func (m *Memory) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return memoryInfo{name: filepath.Base(name), size: int64(len(f.data)), mode: f.mode}, nil
}

// Lstat mirrors Stat: the in-memory filesystem has no links, so this simply
// reports the entry.
func (m *Memory) Lstat(name string) (fs.FileInfo, error) { return m.Stat(name) }

// Chmod updates the recorded mode so a rewrite applies the requested perm even
// when the entry already exists.
func (m *Memory) Chmod(name string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[name]
	if !ok {
		return os.ErrNotExist
	}
	f.mode = perm
	m.files[name] = f
	return nil
}

func (m *Memory) MkdirAll(string, fs.FileMode) error { return nil }
func (m *Memory) Rename(oldPath, newPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[oldPath]
	if !ok {
		return os.ErrNotExist
	}
	m.ensure()
	m.files[newPath] = f
	delete(m.files, oldPath)
	return nil
}
func (m *Memory) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[name]; !ok {
		return os.ErrNotExist
	}
	delete(m.files, name)
	return nil
}

type memoryInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (i memoryInfo) Name() string       { return i.name }
func (i memoryInfo) Size() int64        { return i.size }
func (i memoryInfo) Mode() fs.FileMode  { return i.mode }
func (i memoryInfo) ModTime() time.Time { return time.Time{} }
func (i memoryInfo) IsDir() bool        { return false }
func (i memoryInfo) Sys() any           { return nil }

var _ FileSystem = (*Memory)(nil)
