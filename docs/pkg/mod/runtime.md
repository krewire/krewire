# Package Reference: `runtime`

Import: `github.com/krewire/krewire/pkg/runtime`

## 1. Purpose & Architecture

The `pkg/runtime` package delivers the client-side WebAssembly (WASM) execution engine, Virtual DOM (VDOM) reconciliation, Fiber component lifecycle scheduler, and multi-framework island hydration contracts (`mount.Hydrate`).

For full details on client WASM architecture and hydration contracts, see [Client-Side WASM & Island Hydration Guide](../web-runtime/runtime.md).

---

## 2. Main API Reference

- **`mount.Scan(doc)`**: Discovers server-rendered mount points (`data-kiw-mount`) and extracts JSON props.
- **`mount.Hydrate`**: Progressive hydration strategies (`Load`, `Idle`, `Visible`).
- **`vdom.H(tag, props, children...)`**: Constructs virtual DOM nodes.
- **`vdom.Diff(oldTree, newTree)`**: Computes the minimal DOM patch set.
- **`component.Register(name, factory)`**: Registers client-side reactive components.
- **`boot.Run()`**: Initializes client event loops and mounts interactive islands.

---

## 3. Usage Example

```go
package main

import (
    "github.com/krewire/krewire/pkg/runtime/boot"
    "github.com/krewire/krewire/pkg/runtime/component"
)

func main() {
    // Register reactive components in client WASM binary
    component.Register("Counter", NewCounterComponent)

    // Scan DOM and hydrate islands
    boot.Run()
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Runs in browser's sandboxed WebAssembly execution environment; auto-escapes DOM text to prevent XSS.
- **Sustainable (S2)**: Zero npm/Node.js runtime required; compiled with standard Go WASM toolchain.
- **Scalable (S3)**: Unified hydration contract supporting Go WASM, Svelte 5, and Vue 3 islands.
