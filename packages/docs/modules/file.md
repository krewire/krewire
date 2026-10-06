# `file`

Import: `github.com/krewire/libs/file`

## Purpose

`file` provides file-oriented operations with consistent errors, parent-directory creation, and atomic replacement.

## Main API

- `Read(fsys, name)` — reads a complete file.
- `Write(fsys, name, data, perm)` — creates parent directories and replaces a file.
- `WriteAtomic(fsys, name, data, perm)` — writes beside the destination and renames into place.
- `Copy(fsys, dst, src, perm)` — copies complete file contents.
- `ReadContext`, `WriteContext`, `WriteAtomicContext`, `CopyContext` — context-aware variants.

Pass `nil` for `fsys` to use `fs.Default`.

## Example

```go
err := file.WriteAtomic(nil, "var/data/config.json", data, 0o600)
if err != nil { return err }
```

## Design boundary

`file` owns file operations, not object-store behavior, serialization formats, locking, or retention policies. Atomic replacement is intended to prevent readers from observing a partially written file; it is not a distributed transaction.
