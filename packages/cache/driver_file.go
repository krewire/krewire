package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileCacheConfig contains options for disk-backed file caching.
type FileCacheConfig struct {
	Directory string
}

// FileCache implements Cache using local filesystem storage.
type FileCache struct {
	dir string
}

// NewFileCache creates a disk-based Cache.
func NewFileCache(cfg FileCacheConfig) (*FileCache, error) {
	dir := cfg.Directory
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "krewire_cache")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cache: create dir %s: %w", dir, err)
	}

	return &FileCache{dir: dir}, nil
}

func (f *FileCache) keyPath(key string) string {
	h := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(h[:])
	// Split by prefix for inode distribution: dir/ab/cd/hash
	subDir := filepath.Join(f.dir, hashStr[:2], hashStr[2:4])
	return filepath.Join(subDir, hashStr)
}

func (f *FileCache) Get(_ context.Context, key string) ([]byte, error) {
	path := f.keyPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	// File format: [8 bytes UnixNano expiry (0 = never)] + [payload]
	if len(data) < 8 {
		_ = os.Remove(path)
		return nil, ErrCacheMiss
	}

	expiryNano := int64(binary.BigEndian.Uint64(data[:8]))
	if expiryNano > 0 && time.Now().UnixNano() > expiryNano {
		_ = os.Remove(path)
		return nil, ErrCacheMiss
	}

	return data[8:], nil
}

func (f *FileCache) GetString(ctx context.Context, key string) (string, error) {
	b, err := f.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (f *FileCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	path := f.keyPath(key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var expiryNano int64
	if ttl > 0 {
		expiryNano = time.Now().Add(ttl).UnixNano()
	}

	buf := bytes.NewBuffer(make([]byte, 0, 8+len(val)))
	_ = binary.Write(buf, binary.BigEndian, expiryNano)
	buf.Write(val)

	// Atomic write via temp file
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *FileCache) SetString(ctx context.Context, key string, val string, ttl time.Duration) error {
	return f.Set(ctx, key, []byte(val), ttl)
}

func (f *FileCache) Delete(_ context.Context, key string) error {
	path := f.keyPath(key)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (f *FileCache) Has(ctx context.Context, key string) (bool, error) {
	_, err := f.Get(ctx, key)
	if err == nil {
		return true, nil
	}
	if err == ErrCacheMiss {
		return false, nil
	}
	return false, err
}

func (f *FileCache) Clear(_ context.Context) error {
	return os.RemoveAll(f.dir)
}

func (f *FileCache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	val, err := f.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if err != ErrCacheMiss {
		return nil, err
	}

	computed, err := fn()
	if err != nil {
		return nil, err
	}

	_ = f.Set(ctx, key, computed, ttl)
	return computed, nil
}

func (f *FileCache) Close() error {
	return nil
}

var _ Cache = (*FileCache)(nil)
