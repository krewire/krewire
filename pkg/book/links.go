package book

import (
	"strings"
)

// link prefixes a site path with the book's base path and keeps chapter
// links in extensionless form.
func (b *Book) link(path string) string {
	path = pageLink(path)
	if b.base == "" || b.base == "/" {
		return path
	}
	return strings.TrimRight(b.base, "/") + path
}

// pageLink normalizes a chapter page path to the extensionless form its
// sibling .html file is served at, stripping any trailing slash; asset and
// root paths pass through unchanged.
func pageLink(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" || !isPagePath(trimmed) {
		return path
	}
	return trimmed
}

// isPagePath reports whether path points to a rendered page file rather than
// the site root or an asset file.
func isPagePath(path string) bool {
	if path == "" || path == "/" {
		return false
	}
	last := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		last = path[i+1:]
	}
	return last != "" && !strings.Contains(last, ".")
}

// normalizeBase normalizes a base URL into leading-and-trailing-slash form
// ("/", "/guide/", …).
func normalizeBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 {
		base = strings.TrimRight(base, "/") + "/"
	}
	return base
}
