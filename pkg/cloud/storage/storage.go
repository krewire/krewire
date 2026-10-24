// Package storage provides app-level key-value storage: a small context-aware
// contract with in-memory and filesystem backends, plus a Provider that binds
// the active backend into the app DI container.
//
// Keys are "/"-separated paths ("sessions/abc", "uploads/2026/x.bin"). The
// filesystem backend maps them onto files under its root; path traversal is
// rejected. Values are raw bytes; encode structs at the call site.
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	foundation "github.com/krewire/krewire/pkg/storage"
)

// ErrNotFound is re-exported from the foundational storage package for backward compatibility.
var ErrNotFound = foundation.ErrNotFound

// KV is the canonical key-value storage contract defined in packages/storage.
type KV = foundation.KV

// MemoryKV is a goroutine-safe in-memory backend re-exported from packages/storage.
type MemoryKV = foundation.MemoryKV

// NewMemory returns an empty in-memory store.
func NewMemory() *MemoryKV {
	return foundation.NewMemoryKV()
}

// FileKV persists keys as files beneath root.
type FileKV struct {
	root string
}

// NewFile returns a filesystem-backed store rooted at dir, creating it.
func NewFile(dir string) (*FileKV, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root %s: %w", dir, err)
	}
	return &FileKV{root: dir}, nil
}

func (f *FileKV) Get(_ context.Context, key string) ([]byte, bool, error) {
	p, err := f.path(key)
	if err != nil {
		return nil, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("storage: get %s: %w", key, err)
	}
	return b, true, nil
}

func (f *FileKV) Put(_ context.Context, key string, val []byte) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir for %s: %w", key, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".kiw-tmp-*")
	if err != nil {
		return fmt.Errorf("storage: temp for %s: %w", key, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(val); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storage: write %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: close %s: %w", key, err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: rename %s: %w", key, err)
	}
	return nil
}

func (f *FileKV) Delete(_ context.Context, key string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

func (f *FileKV) List(_ context.Context, prefix string) ([]string, error) {
	prefix = cleanKey(prefix)
	dir := f.root
	if prefix != "" {
		dir = filepath.Join(f.root, filepath.FromSlash(prefix))
	}
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(f.root, p)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: list %s: %w", prefix, err)
	}
	sort.Strings(out)
	return out, nil
}

func (f *FileKV) path(key string) (string, error) {
	for _, seg := range strings.Split(strings.ReplaceAll(key, "\\", "/"), "/") {
		if seg == ".." {
			return "", fmt.Errorf("storage: unsafe key %q", key)
		}
	}
	key = cleanKey(key)
	if key == "" {
		return "", errors.New("storage: empty key")
	}
	return filepath.Join(f.root, filepath.FromSlash(key)), nil
}

func cleanKey(key string) string {
	return strings.Trim(path.Clean("/"+strings.ReplaceAll(key, "\\", "/")), "/")
}
