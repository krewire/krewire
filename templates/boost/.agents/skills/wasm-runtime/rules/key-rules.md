# WASM Runtime — Key Rules

- SSR/client parity is deterministic: `RenderHTML(Comp(props))` must equal island HTML after canonicalization.
- Never re-render SSR text nodes during hydration; hydration mismatches warn per island name/prop key, not crash.
- SSR content is always complete without JS; islands are progressive enhancement.
- Theme preference `localStorage "krewire-theme"` is respected by both SSG `<head>` script and runtime toggle.
- Standard Go WASM only for v1; no TinyGo, no CSS-in-JS runtime authoring.
- Budget violations are blockers: fail the task if hello-world exceeds 800KB gzipped.
