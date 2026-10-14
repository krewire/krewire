# Package: `runtime` (Client-Side WebAssembly Engine)

The `packages/runtime` package delivers an interactive client-side WebAssembly runtime (`GOOS=js GOARCH=wasm`) that renders reactive UI components without requiring Node.js or JavaScript build tools.

---

## 1. Key Modules

- **`runtime/vdom`**: High-speed, diff-and-patch Virtual DOM written in pure Go.
- **`runtime/mount`**: Attaches reactive Go component trees to HTML DOM nodes.
- **`runtime/widgets`**: Built-in stateful widgets (forms, tabs, dropdowns, modal dialogs).
- **`runtime/boot`**: Initializes the WASM event loop and browser window event listeners.

---

## 2. Component Implementation Example

```go
//go:build js && wasm

package main

import (
	"github.com/krewire/krewire/packages/runtime/boot"
	"github.com/krewire/krewire/packages/runtime/mount"
	"github.com/krewire/krewire/packages/runtime/vdom"
)

func CounterComponent() vdom.Node {
	count := 0
	return vdom.Div(
		vdom.Class("counter-box"),
		vdom.H2(vdom.Text("Interactive WASM Island")),
		vdom.Button(
			vdom.Text("Increment"),
			vdom.OnClick(func() {
				count++
			}),
		),
	)
}

func main() {
	boot.Init()
	mount.Mount("#counter-island", CounterComponent())
	select {} // Keep WASM runtime alive
}
```
