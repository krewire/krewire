---
description: Krewire frontend — Go→WASM build, VDOM, hydration islands, widgets. Optionally give the task: /runtime [build|island|widget|theme].
agent: build
---

Execute a Krewire frontend task. $ARGUMENTS is the optional slice (build/island/widget/theme); if empty, infer.

1. Load the `vision` skill to get the workload matrix, then `wasm-runtime` skill and `KWF-T4X9P` spec.
2. Delegate to the `runtime` subagent for focused WASM work — build pipeline, VDOM, component/hooks, hydration islands, widgets, or theming.
3. Verify SSR vs hydrate parity and budgets (≤ 800KB gzipped, hydration < 500ms). For cross-cutting frontend+backend, let `build` orchestrate.

Report artifacts, parity evidence, and gates.
