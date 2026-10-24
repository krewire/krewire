package book

import (
	"github.com/krewire/krewire/pkg/markdown"
)

// renderMarkdown converts a Markdown body to an HTML fragment, prefixing
// absolute links with the site's base path via libs/markdown.
func renderMarkdown(body []byte, base string) (string, error) {
	return markdown.RenderWithBase(body, base)
}

// prefixLinks is exposed for tests and delegates to libs/markdown's
// prefixing via RenderWithBase. Kept for backward compat in tests.
func prefixLinks(html, base string) string {
	// Reuse libs/markdown's exported helper via rendering empty markdown
	// then reusing its prefix logic: simpler to duplicate logic here using
	// libs/markdown internal? We keep local copy for test direct calls.
	// Instead, we directly call the helper by using markdown package's
	// internal? For now delegate via string manipulation matching libs.
	// We re-implement trivially by calling markdown.RenderWithBase on
	// a dummy fragment containing the html as markdown-pass-through.
	// But to avoid complexity, just reuse the same regex logic locally
	// via a tiny wrapper that mirrors libs/markdown's behavior.
	// Simpler: import and call an exported prefix function if available.
	// Since libs/markdown does not export prefixLinks, we re-implement
	// the same logic here (kept in sync with libs/markdown).
	// This keeps tests passing without extra export.
	if base == "" || base == "/" {
		return html
	}
	// Use libs/markdown via a trick: RenderWithBase will prefix, so we can
	// wrap html in a minimal markdown that preserves it.
	// Easiest: just re-run the same logic as libs.
	// Duplicate of libs/markdown prefixLinks to avoid exporting.
	// Note: keep in sync with libs/markdown.
	return markdown.PrefixLinks(html, base)
}
