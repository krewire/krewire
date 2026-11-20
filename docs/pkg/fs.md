# Package Reference: `fs`

Import: `github.com/krewire/krewire/pkg/fs`

## 1. Purpose & Architecture

The `pkg/fs` package establishes an abstraction boundary over filesystem operations. It extends the standard library `io/fs` (which is strictly read-only) with file mutation, directory creation, deletion, and renaming capabilities required by application and build systems, while enabling 100% in-memory filesystem mocking for unit tests.

```text
pkg/fs/
├── fs.go       # FileSystem interface & Mode constants
├── os.go       # fs.OS implementation directly wrapping the host OS
└── memory.go   # fs.Memory implementation for in-memory testing
```

---

## 2. Main API Reference

### Constants & File Permissions

- **`fs.ModeDirDefault`**: `0755` (`rwxr-xr-x`) — Standard directory permissions.
- **`fs.ModeFileDefault`**: `0644` (`rw-r--r--`) — Standard readable file permissions.
- **`fs.ModeFilePrivate`**: `0600` (`rw-------`) — Owner-only restricted file permissions.
- **`fs.ModeDirPrivate`**: `0700` (`rwx------`) — Owner-only restricted directory permissions.

### The `FileSystem` Contract

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

- **`fs.Default`**: Points to `fs.OS{}` for production workloads.
- **`fs.NewMemory()`**: Instantiates an isolated in-memory filesystem for tests.

---

## 3. Usage Example: Unit Testing Without Disk I/O

```go
package main_test

import (
    "testing"
    "github.com/krewire/krewire/pkg/fs"
)

func TestFileWriter(t *testing.T) {
    // Isolated in-memory filesystem
    memFS := fs.NewMemory()

    // Perform operations without touching host disk
    _ = memFS.MkdirAll("/data", fs.ModeDirDefault)
    _ = memFS.WriteFile("/data/hello.txt", []byte("world"), fs.ModeFileDefault)

    content, err := memFS.ReadFile("/data/hello.txt")
    if err != nil || string(content) != "world" {
        t.Fatalf("unexpected content: %s", content)
    }
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Standardized permission masks (`0600` / `0700`) prevent leaking confidential secrets to world-readable files.
- **Sustainable (S2)**: Pure Go without CGO or third-party vfs packages.
- **Scalable (S3)**: Seamlessly swappable between OS filesystem and memory mock without code changes.
