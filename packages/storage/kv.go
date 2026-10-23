package storage

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound is returned by backends when a key exists check fails.
var ErrNotFound = errors.New("storage: key not found")

// KV is the minimal key-value storage contract shared across ecosystem backends.
type KV interface {
	// Get returns the value for key; ok is false when absent.
	Get(ctx context.Context, key string) (val []byte, ok bool, err error)
	// Put stores val under key, creating intermediate namespaces.
	Put(ctx context.Context, key string, val []byte) error
	// Delete removes key; deleting an absent key is not an error.
	Delete(ctx context.Context, key string) error
	// List returns keys under prefix, sorted ascending.
	List(ctx context.Context, prefix string) ([]string, error)
}

// MemoryKV is a goroutine-safe in-memory KV store implementing KV.
type MemoryKV struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// NewMemoryKV returns an empty in-memory KV store.
func NewMemoryKV() *MemoryKV {
	return &MemoryKV{items: map[string][]byte{}}
}

func (m *MemoryKV) Get(_ context.Context, key string) ([]byte, bool, error) {
	key = CleanKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}
	out := make([]byte, len(val))
	copy(out, val)
	return out, true, nil
}

func (m *MemoryKV) Put(_ context.Context, key string, val []byte) error {
	key = CleanKey(key)
	if key == "" {
		return errors.New("storage: empty key")
	}
	cp := make([]byte, len(val))
	copy(cp, val)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = cp
	return nil
}

func (m *MemoryKV) Delete(_ context.Context, key string) error {
	key = CleanKey(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryKV) List(_ context.Context, prefix string) ([]string, error) {
	prefix = CleanKey(prefix)
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for k := range m.items {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// CleanKey normalizes a key path and prevents directory traversal.
func CleanKey(key string) string {
	return strings.Trim(path.Clean("/"+strings.ReplaceAll(key, "\\", "/")), "/")
}
