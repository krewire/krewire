package ui

import (
	"html/template"

	"github.com/krewire/krewire/packages/ui/widget"
)

// Button is a styled link or button element.
type Button struct {
	Text    string `json:"text"`
	URL     string `json:"url"`     // when set, renders <a>
	Variant string `json:"variant"` // primary, ghost, outline; default primary
	Target  string `json:"target"`
	Rel     string `json:"rel"`
}

// Render produces an <a> or <button> element.
// Link hrefs are sanitised through widget.SafeURL so executable URL schemes
// cannot reach the rendered attribute.
func (b Button) Render() template.HTML {
	cls := "btn btn-" + b.Variant
	if b.Variant == "" {
		cls = "btn btn-primary"
	}
	if b.URL != "" {
		attrs := ` href="` + widget.SafeURL(b.URL) + `"`
		if b.Target != "" {
			attrs += ` target="` + template.HTMLEscapeString(b.Target) + `"`
		}
		if b.Rel != "" {
			attrs += ` rel="` + template.HTMLEscapeString(b.Rel) + `"`
		}
		return template.HTML(`<a class="` + cls + `" data-ui-component="button"` + attrs + `>` +
			template.HTMLEscapeString(b.Text) + `</a>`)
	}
	return template.HTML(`<button type="button" class="` + cls + `" data-ui-component="button">` +
		template.HTMLEscapeString(b.Text) + `</button>`)
}
