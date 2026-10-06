# `fs`

Import: `github.com/krewire/libs/fs`

## Purpose

`fs` provides a small, testable filesystem boundary on top of the standard library's `io/fs` and `os` primitives. The standard library exposes filesystem primitives, but this package provides one repository-level contract for read, write, metadata, directory creation, rename, and removal.

## Main API

- `FileSystem` — filesystem contract used by higher-level packages.
- `OS` — host operating system implementation.
- `Memory` — concurrency-safe in-memory implementation for tests and ephemeral work.
- `Default` — default `FileSystem` backed by `OS`.
- `Clean(root, name)` — joins a root and relative name while rejecting path traversal outside the root.

## Example

```go
path, err := fs.Clean(root, "objects/item.txt")
if err != nil { return err }
data, err := fs.Default.ReadFile(path)
```

## Design boundary

`fs` does not define storage semantics, object keys, serialization, or business policy. `file` and `storage` build those responsibilities on top of this boundary. Implement `FileSystem` with an in-memory fake when testing higher-level code.
