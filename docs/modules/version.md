# Ecosystem Versioning & Build Watermark (`version`)

Import: `github.com/krewire/krewire/version`

## Purpose

The `version` package implements semantic versioning contracts (SemVer 2.0.0), ecosystem compatibility checks, and compile-time binary watermark signatures across the Krewire monorepo and its constituent packages.

---

## Core Types

### `Version`
Represents a structured semantic version:

```go
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
}
```

---

## Main API

- **`ParseVersion(s string) (Version, error)`**: Parses a SemVer string (`"0.2.0"`, `"v1.4.2-beta.1"`). Tolerates an optional leading `v`.
- **`MustParseVersion(s string) Version`**: Parses a SemVer string or panics if invalid (used for package-level constant declarations).
- **`(*Version).Compare(other Version) int`**: Returns `-1` if $v < \text{other}$, `0` if $v == \text{other}$, and `1` if $v > \text{other}$.
- **`(*Version).IsCompatible(required Version) bool`**:
  - For pre-1.0.0 versions (`0.y.z`): Requires matching `Major` and `Minor` versions ($0.2.x \approx 0.2.y$).
  - For post-1.0.0 versions (`x.y.z`): Requires matching `Major` version and $actual \ge required$.
- **`CheckCompatibility(actual map[string]Version, requirements map[string]Version) []Incompatibility`**: Evaluates multi-module dependency requirements across workspace repositories.
- **`BuildSignature() string`**: Returns the compile-time ecosystem build signature watermark (default: `"build with krewire"`).

---

## Example Usage

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/version"
)

func main() {
	v1 := version.MustParseVersion("0.2.0")
	v2 := version.MustParseVersion("0.2.5")

	if v2.IsCompatible(v1) {
		fmt.Printf("%s is backwards-compatible with %s\n", v2, v1)
	}

	// Ecosystem watermark
	fmt.Printf("Build watermark: %s\n", version.BuildSignature())
}
```

---

## Build-Time Watermark Injection

When building release binaries via `go build` or `krew build`, the watermark is injected via linker flags (`-ldflags`):

```bash
go build -ldflags="-s -w -X 'github.com/krewire/krewire/version.DefaultSignature=krewire-release-2026'" ./github.com/krewire/krew/cmd/krew
```

---

## Design Boundaries

- **Pure SemVer**: Standard-library-only implementation without external regex or semver packages.
- **Deterministic**: Compatibility rules are strictly mathematical and invariant across all deployment targets.
