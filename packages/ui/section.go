package ui

import "html/template"

// Section is a vertical-rhythm container.
type Section struct {
	Background string        `json:"background"` // optional CSS background value
	Content    template.HTML `json:"content"`
}

// Render produces <section>…</section>.
func (s Section) Render() template.HTML {
	if s.Background != "" {
		return template.HTML(`<section class="section" data-ui-component="section" style="background:` +
			template.HTMLEscapeString(s.Background) + `">` + string(s.Content) + `</section>`)
	}
	return template.HTML(`<section class="section" data-ui-component="section">` + string(s.Content) + `</section>`)
}
