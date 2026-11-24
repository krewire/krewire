# `hub`

Import: `github.com/krewire/hub`

## Purpose

Plugin management, package resolution, semantic version constraint solving, and dynamic capability loading.

## Subpackages

- **`hub/resolver`**: Resolves Go package paths, remote git URLs, and semantic versions (`@latest`, `@v0.2.0`). Implements the cascading `DefaultChain()`: `PluginResolver` → `GoResolver` → `NpmResolver`.
- **`hub/plugin`**: Lifecycle interface and registry for opt-in framework plugins:
  - **`tailwind`** (aliases: `twcss`, `tailwindcss`): Standalone/npm Tailwind CSS compiler.
  - **`svelte`** (alias: `sveltejs`): Modular Svelte 5 component compiler and island hydration bridge.
  - **`vue`** (aliases: `vuejs`, `vue3`): Modular Vue 3 component compiler and island hydration bridge.

## Example

```go
package main

import (
	"fmt"
	"github.com/krewire/hub/plugin"
	"github.com/krewire/hub/resolver"
)

func main() {
	// 1. Resolve package specification
	pkg, ver := resolver.ParsePackageSpec("svelte@latest")
	fmt.Printf("Package: %s, Version: %s\n", pkg, ver)

	// 2. Discover modular plugin
	p := plugin.Find("svelte")
	if p != nil {
		fmt.Printf("Plugin found: %s (v%s)\n", p.Name(), p.Version())
	}
}
```

## Design Boundary

Powers `krew add` and `krew remove`. Allows developers to opt in to modern frontend ecosystems (Tailwind, Svelte, Vue) without breaking pure-Go defaults.
