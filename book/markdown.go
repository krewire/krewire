package book

import (
	"github.com/krewire/krewire/markdown"
)

// renderMarkdown converts a Markdown body to an HTML fragment, prefixing
// absolute links with the site's base path via pkg/markdown.
func renderMarkdown(body []byte, base string) (string, error) {
	return markdown.RenderWithBase(body, base)
}

// prefixLinks rewrites absolute href/src links so they resolve under base,
// delegating to pkg/markdown.PrefixLinks.
func prefixLinks(html, base string) string {
	if base == "" || base == "/" {
		return html
	}
	return markdown.PrefixLinks(html, base)
}
