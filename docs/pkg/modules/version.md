# Ecosystem Versioning & Signature (`packages/version`)

The `packages/version` package implements semantic versioning contracts, ecosystem compatibility matrix checks, and binary signature watermarking.

## Features

- **Semantic Version Parsing & Comparison**: `version.ParseVersion`, `version.MustParseVersion`, `v.Compare`, `v.IsCompatible`.
- **Ecosystem Compatibility Matrix**: `version.CheckCompatibility(actual, requirements)`.
- **Build Signature**:
  ```go
  import "github.com/krewire/krewire/pkg/version"

  sig := version.BuildSignature() // "build with krewire"
  ```
