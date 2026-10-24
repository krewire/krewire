package ui

import (
	"html/template"
	"strings"

	"github.com/krewire/krewire/pkg/ui/widget"
)

// NavLink is a single navigation entry.
type NavLink struct {
	Text   string `json:"text"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
	Target string `json:"target"` // optional: "_blank"
	Rel    string `json:"rel"`    // optional: "noopener"
	Class  string `json:"class"`  // optional: extra CSS classes on the anchor
}

// Nav renders a navigation bar with active-state support.
type Nav struct {
	Links []NavLink `json:"links"`
	Label string    `json:"label"` // aria-label for the <nav> element
}

// Render produces <nav><ul><li><a>...</a></li></ul></nav>.
// Link hrefs are sanitised through widget.SafeURL so executable URL schemes
// (javascript:, vbscript:, data:) cannot reach the rendered attribute.
func (n Nav) Render() template.HTML {
	label := n.Label
	if label == "" {
		label = "Main navigation"
	}
	var b strings.Builder
	b.WriteString(`<nav class="nav" data-ui-component="nav" aria-label="` + template.HTMLEscapeString(label) + `">`)
	b.WriteString(`<ul class="nav-list">`)
	for _, l := range n.Links {
		b.WriteString(`<li class="nav-item`)
		if l.Active {
			b.WriteString(" is-active")
		}
		b.WriteString(`">`)
		b.WriteString(`<a class="nav-link`)
		if l.Class != "" {
			b.WriteString(` ` + template.HTMLEscapeString(l.Class))
		}
		// SafeURL rejects executable schemes (javascript:, vbscript:, data:).
		b.WriteString(`" href="` + widget.SafeURL(l.URL) + `"`)
		if l.Active {
			b.WriteString(` aria-current="page"`)
		}
		if l.Target != "" {
			b.WriteString(` target="` + template.HTMLEscapeString(l.Target) + `"`)
		}
		if l.Rel != "" {
			b.WriteString(` rel="` + template.HTMLEscapeString(l.Rel) + `"`)
		}
		b.WriteString(`>` + template.HTMLEscapeString(l.Text) + `</a>`)
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	b.WriteString(`</nav>`)
	return template.HTML(b.String())
}
