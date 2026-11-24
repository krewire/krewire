# Package Reference: `file`

Import: `github.com/krewire/krewire/file`

## 1. Purpose & Architecture

The `file` package provides high-level, atomic file operations built on top of `fs`. It prevents data corruption during unexpected process crashes and protects against symlink traversal attacks (CWE-59, CWE-61) and predictable temporary file race conditions (CWE-377).

---

## 2. Main API Reference

- **`file.WriteAtomic(fsys fs.FileSystem, path string, data []byte, perm os.FileMode) error`**:
  Writes data to an unguessable temporary file in the same directory using 16 cryptographic random bytes (`tempSuffixEntropyBytes`), syncs to disk, and replaces the target file via atomic `os.Rename`.
- **`file.ErrSymlinkTarget`**: Returned if the target file is a symlink, preventing link redirection attacks.
- **`file.EnsureDir(fsys fs.FileSystem, dir string) error`**: Creates parent directories with secure permissions (`0700` / `ModeDirPrivate`).
- **`file.Read(fsys fs.FileSystem, path string) ([]byte, error)`**: Reads complete contents safely.
- **`file.Exists(fsys fs.FileSystem, path string) bool`**: Checks file existence without race panics.
- **`file.Copy(fsys fs.FileSystem, src, dst string) error`**: Copies files with permission preservation.

---

## 3. Usage Example: Crash-Resistant Atomic Writes

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/file"
)

func SaveConfig(configBytes []byte) error {
    targetFile := "/etc/app/config.json"

    // 1. Refuses to write through symlinks
    // 2. Uses atomic rename
    // 3. Defaults to 0600 private permissions
    err := file.WriteAtomic(nil, targetFile, configBytes, 0o600)
    if err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    return nil
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Refusal to follow write symlinks eliminates symlink hijacking; 16-byte random temporary names prevent TOCTOU file races.
- **Sustainable (S2)**: Zero external dependencies; standard library `crypto/rand` and `os`.
- **Scalable (S3)**: Fully integrated with `fs.FileSystem` for deterministic mocking in tests.
