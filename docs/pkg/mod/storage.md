# Package Reference: `storage`

Import: `github.com/krewire/krewire/pkg/storage`

## 1. Purpose & Architecture

The `pkg/storage` package defines a clean, backend-independent object storage abstraction boundary. Applications depend strictly on the `storage.Store` or `storage.StreamStore` interface, allowing transparent swapping between local disk, in-memory mocks, and S3-compatible cloud storage.

For detailed storage architecture and filesystem operations, see [Storage & Filesystem Guide](../system/storage-and-fs.md).

---

## 2. Main API Reference

- **`storage.Store`**: Core contract with `Put(key string, data []byte)`, `Get(key string)`, `Delete(key string)`, and `Stat(key string) (ObjectInfo, error)`.
- **`storage.Provider`**: Type alias for `Store` conforming to Krewire dependency naming conventions.
- **`storage.StreamStore`**: Extends `Store` with `PutStream(key string, r io.Reader)` and `GetStream(key string) (io.ReadCloser, error)`.
- **`storage.NewLocal(root string, fsys fs.FileSystem) (*Local, error)`**: Creates local filesystem storage with path-traversal prevention.
- **`storage.NewMemory() *Memory`**: Ephemeral in-memory mock store for unit tests.
- **`storage.NewS3(cfg S3Config) (*S3, error)`**: Cloud object storage driver (AWS S3, Cloudflare R2, MinIO).
- **`storage.NewRedis(client) (*Redis, error)`**: Redis/Valkey backed binary storage.

---

## 3. Usage Example

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/storage"
)

func main() {
    // In production: storage.NewLocal or storage.NewS3
    store, _ := storage.NewLocal("/var/data/uploads", nil)

    // Put object
    _ = store.Put("reports/2026-q1.pdf", []byte("%PDF-1.4..."))

    // Get object
    data, err := store.Get("reports/2026-q1.pdf")
    if err == nil {
        fmt.Printf("Retrieved %d bytes\n", len(data))
    }
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: `Local` driver integrates `SafePath` validation, preventing directory traversal breakouts.
- **Sustainable (S2)**: `Memory` store eliminates disk I/O in automated test suites; sub-millisecond setup.
- **Scalable (S3)**: Reversible driver swapping; switch from single-node local storage to S3 cloud storage via configuration without rewriting business logic.
