package ui

import (
	"html/template"
	"strings"
)

// Footer is a bottom bar with Left, Center, and Right slots.
type Footer struct {
	Left   template.HTML `json:"left"`
	Center template.HTML `json:"center"`
	Right  template.HTML `json:"right"`
}

// Render produces <footer><div class="footer-inner">…</div></footer>.
func (f Footer) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<footer class="ui-footer" data-ui-component="footer">`)
	b.WriteString(`<div class="footer-inner">`)
	if f.Left != "" {
		b.WriteString(`<div class="footer-left">`)
		b.WriteString(string(f.Left))
		b.WriteString(`</div>`)
	}
	if f.Center != "" {
		b.WriteString(`<div class="footer-center">`)
		b.WriteString(string(f.Center))
		b.WriteString(`</div>`)
	}
	if f.Right != "" {
		b.WriteString(`<div class="footer-right">`)
		b.WriteString(string(f.Right))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</footer>`)
	return template.HTML(b.String())
}
