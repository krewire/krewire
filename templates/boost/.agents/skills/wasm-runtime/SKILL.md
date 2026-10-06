---
name: wasm-runtime
description: Use for Krewire frontend work — Go→WASM build, VDOM, components/hooks, hydration islands, widgets, and theme bridging. Triggers: "WASM", "frontend", "hydration", "island", "widget", "VDOM", "runtime".
---

# WASM Runtime — Go-Native Frontend

Build reactive frontends without leaving Go. This skill covers `framework/runtime` (spec `KWF-T4X9P`).

## 1. Detect the context

- Read `docs/project-vision.md` and `docs/specs/framework/KWF-WASM-T4X9P-wasm-client-runtime.md`.
- Confirm the project kind: `site`/`app` projects use islands; check `web/ssg` for `client:load`/`idle`/`visible` markers.
- Identify whether the task is **build pipeline**, **VDOM**, **component/hooks**, **hydration**, **widgets**, or **theming**.

## 2. Build pipeline

- Use standard Go WASM: `GOOS=js GOARCH=wasm go build`; never TinyGo for v1.
- `runtime/build.BuildWASM` → outputs `.wasm` + JS glue to `site/_assets/runtime.*` with content hash.
- `krewire build` must run WASM build before emitting `site/` when islands exist.

## 3. VDOM & component model

- `VNode{Tag, Props, Children, Key, ComponentType}`; `Diff` in O(n) with keyed children; `RenderHTML` (server) and `PatchDOM` (client) share prop normalization.
- `Component` interface: `Render() VNode` + optional `OnMount`/`OnUnmount`/`ShouldUpdate`.
- Hooks `UseState[T]`/`UseEffect`/`UseRef[T]`/`UseMemo[T]` — enforce stable call order; panic on conditional hook violation in tests.

## 4. Hydration islands

- SSG emits `data-kiw-island` + props JSON for `client:load`/`idle`/`visible`.
- Client boot scans markers, instantiates via `component.Registry`, attaches listeners without re-rendering SSR text nodes.
- Graceful degradation: SSR HTML is always complete and readable without JS; mismatched text warns, not crashes.

## 5. Widgets & theming

- Starter catalog: `Container`, `Row`/`Column`/`Stack`/`Expanded`/`SizedBox`, `Text`/`Image`/`Icon`, `Button`/`Input`/`Checkbox`/`Switch`, `Scaffold`/`AppBar`/`Card`, `ListView`/`ListTile`.
- Theme: `framework/ui` vars → CSS custom props under `data-kiw-theme`; scoped CSS `data-kiw-component` must match server and client computed styles.

## 6. Budgets & gates

- Hello-world + one island ≤ 800KB gzipped; first hydration < 500ms on CI fixture.
- Gates: `gofmt -l .`, `go vet ./...`, `go test ./framework/runtime/...` (VDOM/hydration coverage), `krewire build` + `curl` spot-check.
- Use `runtime` subagent for focused work; `build` orchestrates cross-cutting frontend+backend tasks.

## Rules

Read `rules/key-rules.md` before acting.
