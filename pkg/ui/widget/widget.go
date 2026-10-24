package widget

import (
	"html/template"
	"strings"
)

// Widget represents any renderable Forge UI component.
type Widget interface {
	Render() template.HTML
}

// HTML converts a raw HTML string into a Widget.
type HTML string

// Render returns the HTML content.
func (h HTML) Render() template.HTML {
	return template.HTML(h)
}

// safeURLSchemes are the URL schemes permitted in a rendered href. Executable
// schemes such as javascript:, vbscript:, and data: are refused so an
// attacker-supplied link target cannot execute script when clicked.
var safeURLSchemes = map[string]bool{
	"http": true, "https": true, "mailto": true, "tel": true, "ftp": true,
}

// SafeURL returns a value suitable for an href or src attribute. Relative and
// scheme-less URLs are returned unchanged; a URL with a scheme outside
// safeURLSchemes is replaced with "#". The result is already HTML-escaped, so
// callers must not escape it a second time.
func SafeURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Strip control characters so "java\tscript:" cannot bypass the check.
	cleaned := strings.Map(func(r rune) rune {
		if r <= 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, trimmed)
	idx := strings.Index(cleaned, ":")
	if idx <= 0 {
		return template.HTMLEscapeString(trimmed)
	}
	prefix := cleaned[:idx]
	// A separator before the colon means it is a relative path, not a scheme.
	if strings.ContainsAny(prefix, "/?#") {
		return template.HTMLEscapeString(trimmed)
	}
	if !safeURLSchemes[strings.ToLower(prefix)] {
		return "#"
	}
	return template.HTMLEscapeString(trimmed)
}
