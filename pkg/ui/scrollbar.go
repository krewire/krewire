package ui

// ScrollbarCSS styles the page scrollbar to match the Krewire UI pop-brutalist
// theme. Every value is driven by the --ui-* design tokens, so the
// scrollbar adapts automatically to light/dark mode and to any palette
// override a consumer applies (no separate dark rule is needed). It covers
// both the standard `scrollbar-color`/`scrollbar-width` properties (Firefox)
// and the WebKit pseudo-elements (Chromium, Safari, Edge). Include it once,
// globally, after the palette tokens.
const ScrollbarCSS = `/* Scrollbar — pop-brutalist, theme-driven (auto light/dark). */
:root {
  --ui-scrollbar-size: 12px;
  --ui-scrollbar-track: var(--ui-surface);
  --ui-scrollbar-thumb: var(--ui-primary);
  --ui-scrollbar-thumb-hover: var(--ui-secondary);
  --ui-scrollbar-border: var(--ui-border);
}
html {
  scrollbar-color: var(--ui-scrollbar-thumb) var(--ui-scrollbar-track);
  scrollbar-width: thin;
}
::-webkit-scrollbar {
  width: var(--ui-scrollbar-size);
  height: var(--ui-scrollbar-size);
}
::-webkit-scrollbar-track {
  background: var(--ui-scrollbar-track);
  border-left: 2px solid var(--ui-scrollbar-border);
}
::-webkit-scrollbar-thumb {
  background: var(--ui-scrollbar-thumb);
  border: 2px solid var(--ui-scrollbar-border);
  border-radius: 0;
}
::-webkit-scrollbar-thumb:hover {
  background: var(--ui-scrollbar-thumb-hover);
}
::-webkit-scrollbar-corner {
  background: var(--ui-scrollbar-track);
}`
