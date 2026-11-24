package storage

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// MemoryStore implements Store using an in-memory thread-safe map.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryObject
}

type memoryObject struct {
	data    []byte
	modTime time.Time
}

// NewMemoryStore creates an in-memory Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]memoryObject),
	}
}

// Put saves an object in memory.
func (m *MemoryStore) Put(key string, data []byte) error {
	clean := CleanKey(key)
	if clean == "" {
		return fmt.Errorf("storage: empty key")
	}

	cp := make([]byte, len(data))
	copy(cp, data)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[clean] = memoryObject{
		data:    cp,
		modTime: time.Now(),
	}
	return nil
}

// Get returns the stored object.
func (m *MemoryStore) Get(key string) ([]byte, error) {
	clean := CleanKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, ok := m.items[clean]
	if !ok {
		return nil, ErrNotFound
	}

	out := make([]byte, len(obj.data))
	copy(out, obj.data)
	return out, nil
}

// Delete removes the object.
func (m *MemoryStore) Delete(key string) error {
	clean := CleanKey(key)
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, clean)
	return nil
}

// Stat returns metadata for the object.
func (m *MemoryStore) Stat(key string) (ObjectInfo, error) {
	clean := CleanKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, ok := m.items[clean]
	if !ok {
		return ObjectInfo{}, ErrNotFound
	}

	return ObjectInfo{
		Key:  filepath.ToSlash(clean),
		Size: int64(len(obj.data)),
		Mode: fs.FileMode(0o644),
	}, nil
}

var _ Store = (*MemoryStore)(nil)
