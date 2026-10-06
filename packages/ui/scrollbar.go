package ui

// ScrollbarCSS styles the page scrollbar to match the Forge pop-brutalist
// theme. Every value is driven by the --forge-* design tokens, so the
// scrollbar adapts automatically to light/dark mode and to any palette
// override a consumer applies (no separate dark rule is needed). It covers
// both the standard `scrollbar-color`/`scrollbar-width` properties (Firefox)
// and the WebKit pseudo-elements (Chromium, Safari, Edge). Include it once,
// globally, after the palette tokens.
const ScrollbarCSS = `/* Scrollbar — pop-brutalist, theme-driven (auto light/dark). */
:root {
  --forge-scrollbar-size: 12px;
  --forge-scrollbar-track: var(--forge-surface);
  --forge-scrollbar-thumb: var(--forge-primary);
  --forge-scrollbar-thumb-hover: var(--forge-secondary);
  --forge-scrollbar-border: var(--forge-border);
}
html {
  scrollbar-color: var(--forge-scrollbar-thumb) var(--forge-scrollbar-track);
  scrollbar-width: thin;
}
::-webkit-scrollbar {
  width: var(--forge-scrollbar-size);
  height: var(--forge-scrollbar-size);
}
::-webkit-scrollbar-track {
  background: var(--forge-scrollbar-track);
  border-left: 2px solid var(--forge-scrollbar-border);
}
::-webkit-scrollbar-thumb {
  background: var(--forge-scrollbar-thumb);
  border: 2px solid var(--forge-scrollbar-border);
  border-radius: 0;
}
::-webkit-scrollbar-thumb:hover {
  background: var(--forge-scrollbar-thumb-hover);
}
::-webkit-scrollbar-corner {
  background: var(--forge-scrollbar-track);
}`
