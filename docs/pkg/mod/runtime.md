# `runtime` (Client WASM)

Import: `github.com/krewire/krewire/pkg/runtime`

## Purpose

Interactive client-side WebAssembly runtime (`GOOS=js GOARCH=wasm`) providing reactive Virtual DOM components without JavaScript build dependencies.

## Subpackages

- **`runtime/vdom`**: High-speed, diff-and-patch Virtual DOM tree.
- **`runtime/mount`**: Attaches reactive Go component trees to HTML DOM elements.
- **`runtime/widgets`**: Built-in stateful UI widgets (counters, forms, tabs, dropdowns).
- **`runtime/boot`**: Initializes WASM event loop and browser event bridging.

## Example

```go
//go:build js && wasm

package main

import (
	"github.com/krewire/krewire/pkg/runtime/boot"
	"github.com/krewire/krewire/pkg/runtime/mount"
	"github.com/krewire/krewire/pkg/runtime/vdom"
)

func main() {
	boot.Init()
	mount.Mount("#app", vdom.Div(
		vdom.Class("hero"),
		vdom.H1(vdom.Text("Reactive WebAssembly")),
	))
	select {}
}
```

## Design boundary

Targeted at browser client execution (`GOOS=js GOARCH=wasm`). See the full [Runtime Reference](../web-runtime/runtime.md).
