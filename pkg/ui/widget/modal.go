package widget

import (
	"html/template"
	"strings"
)

// Modal represents a popup dialog.
type Modal struct {
	ID      string
	Title   string
	Content Widget
	Footer  Widget
}

// NewModal creates a new modal dialog.
func NewModal(id, title string) *Modal {
	return &Modal{ID: id, Title: title}
}

// WithContent sets the modal content.
func (m *Modal) WithContent(content Widget) *Modal {
	m.Content = content
	return m
}

// WithFooter sets the modal action footer.
func (m *Modal) WithFooter(footer Widget) *Modal {
	m.Footer = footer
	return m
}

// Render produces the modal markup.
func (m *Modal) Render() template.HTML {
	var buf strings.Builder
	buf.WriteString(`<dialog class="` + modalClassPrefix + `" id="` + template.HTMLEscapeString(m.ID) + `" style="border:var(--ui-pop-border); border-radius:var(--ui-radius); background:var(--ui-bg); box-shadow:var(--ui-pop-shadow); padding:0; max-width:540px; width:90%;">`)
	buf.WriteString(`<div style="padding:1rem 1.25rem; border-bottom:1.5px solid var(--ui-border); display:flex; justify-content:space-between; align-items:center;">`)
	buf.WriteString(`<h3 style="margin:0; font-size:1.15rem; font-weight:800;">` + template.HTMLEscapeString(m.Title) + `</h3>`)
	buf.WriteString(`<button onclick="this.closest('dialog').close()" style="border:none; background:none; font-size:1.5rem; line-height:1; cursor:pointer;">×</button>`)
	buf.WriteString(`</div>`)
	if m.Content != nil {
		buf.WriteString(`<div style="padding:1.25rem;">` + string(m.Content.Render()) + `</div>`)
	}
	if m.Footer != nil {
		buf.WriteString(`<div style="padding:0.75rem 1.25rem; border-top:1.5px solid var(--ui-border); background:var(--ui-surface); display:flex; justify-content:flex-end; gap:0.5rem;">` + string(m.Footer.Render()) + `</div>`)
	}
	buf.WriteString(`</dialog>`)
	return template.HTML(buf.String())
}
