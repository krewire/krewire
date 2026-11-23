package workgraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultCacheDir is the local cache location relative to the root. It sits
// under .krewire/, which discovery never hashes.
const DefaultCacheDir = ".krewire/cache/workgraph"

// Cache lookup errors.
var (
	ErrCacheMiss    = errors.New("workgraph: cache miss")
	ErrCacheCorrupt = errors.New("workgraph: cache entry corrupt")
)

// Entry is a cached successful target result. Only passing results are
// stored: a failure is never restored, so a failing gate always re-runs.
type Entry struct {
	Schema     string `json:"schema"`
	Key        string `json:"key"`
	Target     string `json:"target"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	DurationMs int64  `json:"duration_ms"`
	Checksum   string `json:"checksum"`
}

func (e Entry) checksum() string {
	e.Checksum = ""
	data, _ := json.Marshal(e)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Cache stores target results by key. Implementations must return
// ErrCacheMiss or an error wrapping ErrCacheCorrupt from Get.
type Cache interface {
	Get(key string) (*Entry, error)
	Put(e *Entry) error
}

// LocalCache is a content-addressed directory cache.
type LocalCache struct{ Dir string }

func (c LocalCache) path(key string) (string, error) {
	if len(key) < 3 || filepath.Base(key) != key {
		return "", fmt.Errorf("workgraph: invalid cache key %q", key)
	}
	return filepath.Join(c.Dir, key[:2], key+".json"), nil
}

// Get returns the verified entry for key. Unreadable, malformed, mismatched,
// or non-passing entries are deleted and reported as corrupt, so the caller
// falls back to execution.
func (c LocalCache) Get(key string) (*Entry, error) {
	p, err := c.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrCacheMiss
	}
	corrupt := func(why string) (*Entry, error) {
		_ = os.Remove(p)
		return nil, fmt.Errorf("%w: %s: %s", ErrCacheCorrupt, key, why)
	}
	if err != nil {
		return corrupt(err.Error())
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return corrupt("malformed JSON")
	}
	switch {
	case e.Schema != KeySchema:
		return corrupt("schema " + e.Schema)
	case e.Key != key:
		return corrupt("key mismatch")
	case e.ExitCode != 0:
		return corrupt("non-passing entry")
	case e.Checksum != e.checksum():
		return corrupt("checksum mismatch")
	}
	return &e, nil
}

// Put stores a passing entry atomically (write to a temp file, then rename).
func (c LocalCache) Put(e *Entry) error {
	if e.ExitCode != 0 {
		return fmt.Errorf("workgraph: refusing to cache failing target %s", e.Target)
	}
	p, err := c.path(e.Key)
	if err != nil {
		return err
	}
	e.Schema = KeySchema
	e.Checksum = e.checksum()
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), p)
}
