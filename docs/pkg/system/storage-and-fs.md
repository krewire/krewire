# Package: `storage`, `fs` & `file` (Storage & Filesystem)

This package trio abstracts persistent object storage, testable filesystem operations, and atomic file transactions.

---

## 1. Object Storage (`pkg/storage`)

A provider-agnostic interface for storing and retrieving binary objects (media, build artifacts, backups):

```go
type Bucket interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
```
Supported backends: Local disk filesystem, S3-compatible cloud storage, and in-memory mock storage for unit testing.

---

## 2. Testable Filesystem (`pkg/fs`)

Wraps `io/fs` and `os` behind testable interfaces (`fs.FS`), enabling fast in-memory mock testing without writing temporary files to the host disk.

---

## 3. Atomic File Writes (`pkg/file`)

Ensures zero-downtime, safe file replacements: writes content to a temporary sibling file and performs an atomic rename (`os.Rename`), preventing file corruption during power cuts or process interruptions.
