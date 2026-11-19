# Package: `runtime` (Client-Side WASM Engine, VDOM & Island Hydration)

The `pkg/runtime` library provides the client-side WebAssembly (WASM) execution engine, Virtual DOM (VDOM) diffing and reconciliation algorithm, Fiber component lifecycle scheduler, and multi-framework island hydration contracts.

It allows developers to write interactive, reactive client-side widgets in 100% pure Go compiled to WebAssembly (`GOOS=js GOARCH=wasm`), or embed lightweight modular frontend framework islands (Svelte 5, Vue 3) while keeping initial page loads static and zero-JS by default.

---

## 1. Architectural Overview & Subsystem Map

```text
pkg/runtime/
├── mount/      # Progressive island hydration contracts (data-kiw-mount, load, idle, visible)
├── vdom/       # Virtual DOM tree representation, diffing engine & browser DOM patcher
├── component/  # Reactive component lifecycle, Fiber scheduler, state hooks & event dispatch
├── schedule/   # requestAnimationFrame & requestIdleCallback browser scheduler
├── boot/       # Client WASM runtime initialization, DOM scanner & event listeners
├── build/      # Automated pure-Go WASM compiler pipeline (GOOS=js GOARCH=wasm)
└── widgets/    # Standard reactive client widgets (Counters, Modals, Autocomplete, Tabs)
```

---

## 2. Progressive Island Hydration Contract (`pkg/runtime/mount`)

Krewire implements the **Islands Architecture**: pages are rendered statically on the server (SSG/SSR) with zero JavaScript. Interactive islands are hydrated selectively on the client.

### Hydration Attributes

Mount points emit standard HTML markers:

```html
<div data-kiw-mount="Counter"
     data-kiw-framework="wasm"
     data-kiw-hydrate="visible"
     data-kiw-props='{"initial":10}'>
  <!-- Server-rendered fallback HTML -->
  <button class="btn">Count: 10</button>
</div>
```

### Hydration Modes (`mount.Hydrate`)

- **`load`** (`mount.Load`): Hydrates immediately as soon as the client module or WASM bundle boots.
- **`idle`** (`mount.Idle`): Hydrates during browser idle periods via `window.requestIdleCallback`.
- **`visible`** (`mount.Visible`): Hydrates only when the island scrolls into the browser viewport via `IntersectionObserver`.

### Multi-Framework Islands Support

Through `pkg/runtime/mount` and `pkg/hub/plugin`, mount points support:
- **`wasm`** (Default): Pure-Go compiled WebAssembly island.
- **`svelte`**: Svelte 5 reactive component island.
- **`vue`**: Vue 3 reactive component island.

---

## 3. Virtual DOM Engine (`pkg/runtime/vdom`)

The `vdom` package implements a lightweight tree representation and Myers/Snabbdom-inspired reconciliation diffing algorithm:

### Node Primitives

```go
package vdom

type VNode struct {
    Tag      string
    Props    map[string]any
    Children []*VNode
    Text     string
    Key      string
    DOMNode  any // js.Value when compiled with GOOS=js
}

func H(tag string, props map[string]any, children ...*VNode) *VNode
func Text(content string) *VNode
func Diff(oldVNode, newVNode *VNode) []Patch
```

### Fast Patch Application

When state changes occur:
1. The component renders a new `*VNode` tree.
2. `vdom.Diff` compares the previous and new virtual trees, calculating the minimal set of DOM patches (`PatchText`, `PatchProp`, `InsertChild`, `RemoveChild`, `ReorderKeys`).
3. Patches are flushed to the real browser DOM in a single `requestAnimationFrame` tick.

---

## 4. Component Lifecycle & State Hooks (`pkg/runtime/component`)

Pure-Go client components implement reactive state hooks and lifecycle contracts:

```go
package main

import (
    "syscall/js"
    "github.com/krewire/krewire/pkg/runtime/component"
    "github.com/krewire/krewire/pkg/runtime/vdom"
)

type CounterWidget struct {
    component.Base
    count int
}

func (c *CounterWidget) Init(props map[string]any) {
    if initVal, ok := props["initial"].(float64); ok {
        c.count = int(initVal)
    }
}

func (c *CounterWidget) Render() *vdom.VNode {
    return vdom.H("div", map[string]any{"class": "counter-card"},
        vdom.H("span", nil, vdom.Text(fmt.Sprintf("Current: %d", c.count))),
        vdom.H("button", map[string]any{
            "class": "btn btn-primary",
            "onclick": func(this js.Value, args []js.Value) any {
                c.count++
                c.Update() // Triggers scheduled VDOM re-render
                return nil
            },
        }, vdom.Text("+ Increment")),
    )
}
```

---

## 5. Client Boot Scanner (`pkg/runtime/boot`)

The client runtime automatically boots and mounts components upon DOM ready:

```go
package main

import (
    "github.com/krewire/krewire/pkg/runtime/boot"
    "github.com/krewire/krewire/pkg/runtime/component"
)

func main() {
    // 1. Register interactive Go components
    component.Register("Counter", func() component.Component {
        return &CounterWidget{}
    })

    // 2. Scan DOM for [data-kiw-mount] markers and mount islands
    boot.Run()
}
```

---

## 6. Alignment with the Three Pillars (3S)

- **Secure (S1)**:
  - WASM executes inside the browser's hermetic WebAssembly sandbox with no direct unsafe memory access.
  - VDOM auto-escapes all text nodes and attribute values to eliminate Cross-Site Scripting (XSS).
- **Sustainable (S2)**:
  - Zero JavaScript dependencies for Go WASM workloads; ships with standard Go WASM exec runtime (`wasm_exec.js`).
  - Modular island hydration ensures users download interactive code only for parts of the page that actually require interactivity.
- **Scalable (S3)**:
  - Unifies Go WASM, Svelte 5, and Vue 3 under a single progressive hydration contract (`data-kiw-hydrate`).
