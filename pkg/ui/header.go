package ui

import (
	"html/template"
	"strings"
)

// Header is a top bar with Brand, Nav, and Actions slots.
type Header struct {
	Brand   template.HTML `json:"brand"`
	Nav     template.HTML `json:"nav"`
	Actions template.HTML `json:"actions"`
}

// Render produces <header><div class="header-inner">…</div></header>.
func (h Header) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<header class="ui-header" data-ui-component="header">`)
	b.WriteString(`<div class="header-inner">`)
	if h.Brand != "" {
		b.WriteString(`<div class="header-brand">`)
		b.WriteString(string(h.Brand))
		b.WriteString(`</div>`)
	}
	if h.Nav != "" {
		b.WriteString(`<div class="header-nav">`)
		b.WriteString(string(h.Nav))
		b.WriteString(`</div>`)
	}
	if h.Actions != "" {
		b.WriteString(`<div class="header-actions">`)
		b.WriteString(string(h.Actions))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</header>`)
	return template.HTML(b.String())
}
