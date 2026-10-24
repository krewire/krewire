package ui

import (
	"html/template"
	"strings"
)

// Card is a bordered container with optional header, body, and footer.
type Card struct {
	Header template.HTML `json:"header"`
	Body   template.HTML `json:"body"`
	Footer template.HTML `json:"footer"`
}

// Render produces <article class="card">…</article>.
func (c Card) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<article class="card" data-ui-component="card">`)
	if c.Header != "" {
		b.WriteString(`<div class="card-header">`)
		b.WriteString(string(c.Header))
		b.WriteString(`</div>`)
	}
	if c.Body != "" {
		b.WriteString(`<div class="card-body">`)
		b.WriteString(string(c.Body))
		b.WriteString(`</div>`)
	}
	if c.Footer != "" {
		b.WriteString(`<div class="card-footer">`)
		b.WriteString(string(c.Footer))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</article>`)
	return template.HTML(b.String())
}
