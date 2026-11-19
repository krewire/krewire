# Package: `storage`, `fs` & `file` (Object Storage, Filesystem & Atomic I/O)

The `pkg/storage`, `pkg/fs`, and `pkg/file` libraries provide unified object storage abstractions, testable filesystem boundaries, and race-free atomic file operations with built-in symlink and traversal protections.

---

## 1. Architectural Map & Responsibility Boundaries

```text
pkg/
├── storage/        # Object storage provider abstraction & multi-backend drivers
│   ├── storage.go  # Store & StreamStore contracts, ObjectInfo, Local driver
│   ├── memory.go   # Thread-safe in-memory mock store
│   ├── s3.go       # S3-compatible cloud object storage driver
│   ├── redis.go    # Redis/Valkey backed binary object driver
│   └── kv.go       # Key-value backed persistent storage
├── fs/             # Testable filesystem abstraction on top of io/fs
│   ├── fs.go       # FileSystem interface (Open, ReadFile, WriteFile, MkdirAll, Stat, Rename, Remove)
│   ├── memory.go   # Ephemeral in-memory filesystem for unit testing
│   └── os.go       # Host OS implementation (fs.OS)
└── file/           # Atomic file operations & CWE-59/CWE-377 defensive guards
    ├── file.go     # WriteAtomic, Read, Write, Exists, EnsureDir, symlink defense
    └── context.go  # Context-aware file I/O operations
```

---

## 2. Reusable Object Storage Abstraction (`pkg/storage`)

Applications interact with `storage.Store` rather than coupling to vendor SDKs:

### Core Interfaces

```go
package storage

import (
    "io"
    "io/fs"
)

// Store stores opaque objects addressed by a unique key.
type Store interface {
    Put(key string, data []byte) error
    Get(key string) ([]byte, error)
    Delete(key string) error
    Stat(key string) (ObjectInfo, error)
}

// Provider is an alias for Store per Krewire naming standards.
type Provider = Store

// StreamStore defines streaming methods for large chunked objects.
type StreamStore interface {
    Store
    PutStream(key string, r io.Reader) error
    GetStream(key string) (io.ReadCloser, error)
}

type ObjectInfo struct {
    Key  string
    Size int64
    Mode fs.FileMode
}
```

### Pluggable Drivers

1. **`storage.NewLocal(root string, fsys fs.FileSystem)`**: Local filesystem storage with automatic path traversal protection.
2. **`storage.NewMemory()`**: Fast in-memory map implementation for unit tests.
3. **`storage.NewS3(cfg S3Config)`**: S3-compatible object storage (AWS S3, MinIO, Cloudflare R2, Wasabi).
4. **`storage.NewRedis(client)`**: Binary payload cache backed by Redis/Valkey.

### Example: Storing User Avatars with Storage Provider

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/storage"
)

type MediaService struct {
    store storage.Store
}

func (s *MediaService) SaveAvatar(userID string, imageBytes []byte) error {
    key := fmt.Sprintf("avatars/%s.png", userID)
    return s.store.Put(key, imageBytes)
}

func main() {
    // Development / Local VPS: Local storage
    diskStore, err := storage.NewLocal("/var/data/media", nil)
    if err != nil {
        panic(err)
    }

    service := &MediaService{store: diskStore}
    _ = service.SaveAvatar("user_101", []byte{0x89, 0x50, 0x4E, 0x47})

    // In unit testing: Transparently swap to MemoryStore
    memService := &MediaService{store: storage.NewMemory()}
    _ = memService.SaveAvatar("test_user", []byte{0x00})
}
```

---

## 3. Testable Filesystem Boundary (`pkg/fs`)

The standard library `io/fs` provides read-only interfaces. `pkg/fs.FileSystem` extends this to provide mutative operations while maintaining complete testability:

```go
type FileSystem interface {
    Open(name string) (io.ReadCloser, error)
    ReadFile(name string) ([]byte, error)
    WriteFile(name string, data []byte, perm fs.FileMode) error
    Stat(name string) (fs.FileInfo, error)
    MkdirAll(path string, perm fs.FileMode) error
    Rename(oldPath, newPath string) error
    Remove(name string) error
}
```

- **`fs.OS{}`**: Interacts directly with the operating system filesystem.
- **`fs.NewMemory()`**: Operates completely in-memory, allowing isolated tests without polluting temporary directories or touching disk.

---

## 4. Atomic File Operations & Symlink Defense (`pkg/file`)

Writing directly to destination files can cause partial writes during process crashes or race conditions during concurrent reads. `pkg/file` guarantees atomicity:

### Atomic Write Protocol (`file.WriteAtomic`)

1. **Symlink Rejection (CWE-59 / CWE-61)**: If the target is an existing symlink, the write is immediately rejected (`ErrSymlinkTarget`), preventing symlink manipulation attacks.
2. **Entropy Temporary Suffix (CWE-377)**: Creates a temporary file in the same directory using 16 random cryptographic bytes (`crypto/rand`), preventing predictable filename attacks.
3. **Restricted Default Permissions**: Automatically creates parent directories with `0700` (`ModeDirPrivate`) and files with `0600` (`ModeFilePrivate`) to protect confidential state.
4. **POSIX Atomic Rename**: Flushes buffers to disk and executes `os.Rename`, ensuring readers see either the old file or the complete new file.

### Complete Atomic Write Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/krewire/krewire/pkg/file"
)

func SaveApplicationState(stateJSON []byte) error {
    targetPath := "/opt/krewire/data/state.json"

    // Writes atomically via temporary file and rename
    // Rejects symlinks and sets mode 0600
    err := file.WriteAtomic(nil, targetPath, stateJSON, 0o600)
    if err != nil {
        return fmt.Errorf("failed to save state atomically: %w", err)
    }

    return nil
}
```

---

## 5. Alignment with the Three Pillars (3S)

- **Secure (S1)**:
  - Default permissions restrict read/write access to the current process owner (`0600` for files, `0700` for directories).
  - Explicit refusal to follow write symlinks eliminates symlink traversal attacks.
- **Sustainable (S2)**:
  - Zero third-party dependencies; uses only standard Go `io/fs`, `os`, and `crypto/rand`.
  - In-memory mock store and filesystem allow fast, hermetic unit tests with zero disk thrashing.
- **Scalable (S3)**:
  - Transparent transition from single-node local storage (`storage.Local`) to cloud object storage (`storage.S3`) with zero domain logic rewrites.
