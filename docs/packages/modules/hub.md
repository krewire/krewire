# `hub`

Import: `github.com/krewire/krewire/packages/hub`

## Purpose

Plugin management, package resolution, semantic version constraint solving, and dynamic capability loading.

## Subpackages

- **`hub/resolver`**: Resolves Go package paths, remote git URLs, and semantic versions (`@latest`, `@v0.1.0`).
- **`hub/plugin`**: Lifecycle interface and registry for opt-in framework plugins (e.g., Tailwind CSS).

## Example

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/packages/hub/resolver"
)

func main() {
	pkg, ver := resolver.ParsePackageSpec("github.com/krewire/krewire@v0.1.0")
	fmt.Printf("Package: %s, Version: %s\n", pkg, ver)
}
```

## Design boundary

Powers `kiw add` and `kiw remove`. Independent of specific package implementations.
