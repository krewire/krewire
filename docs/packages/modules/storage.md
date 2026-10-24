# `storage`

Import: `github.com/krewire/krewire/packages/storage`

## Purpose

`storage` defines a backend-independent object storage contract and provides a local filesystem implementation. Consumers depend on `Store` instead of coupling application code to a concrete backend.

## Main API

- `Store` — `Put`, `Get`, `Delete`, and `Stat` operations.
- `ContextStore` — context-aware extension of `Store` without changing the base interface.
- **Drivers**:
  - `Local` — local filesystem storage with path-traversal protection and atomic replacement.
  - `MemoryStore` (`NewMemoryStore`) — thread-safe in-memory object store.
  - `S3Store` (`NewS3Store`) — S3-compatible cloud storage driver (AWS S3, MinIO, Cloudflare R2, Wasabi) with pure-Go AWS SigV4 authorization.
- `ObjectInfo` — key, size, and mode metadata.

Object keys are resolved below the configured root. Path traversal attempts are rejected.

## Example

```go
store, err := storage.NewLocal("var/objects", nil)
if err != nil { return err }
if err := store.Put("images/avatar.png", data); err != nil { return err }
contents, err := store.Get("images/avatar.png")
```

## Design boundary

`storage` owns object addressing and backend abstraction. It does not define serialization, authentication, lifecycle, replication, or remote cloud adapters. Additional backends can implement `Store` without changing consumers.
