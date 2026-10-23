package ui

import "html/template"

// Badge is an inline label with semantic variants.
type Badge struct {
	Text    string `json:"text"`
	Variant string `json:"variant"` // primary, neutral, success, info, warning, error; default primary
}

// Render produces <span class="badge badge-{variant}">…</span>.
func (b Badge) Render() template.HTML {
	v := b.Variant
	if v == "" {
		v = "primary"
	}
	return template.HTML(`<span class="badge badge-` + template.HTMLEscapeString(v) +
		`" data-ui-component="badge">` + template.HTMLEscapeString(b.Text) + `</span>`)
}
