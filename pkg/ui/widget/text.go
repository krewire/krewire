package widget

import "html/template"

// Text renders safe escaped text.
type Text struct {
	Content string
	Bold    bool
	Muted   bool
}

// NewText returns a new Text widget.
func NewText(content string) *Text {
	return &Text{Content: content}
}

// Strong marks the text as bold.
func (t *Text) Strong() *Text {
	t.Bold = true
	return t
}

// Subtle marks the text as muted.
func (t *Text) Subtle() *Text {
	t.Muted = true
	return t
}

// Render produces the text markup.
func (t *Text) Render() template.HTML {
	escaped := template.HTMLEscapeString(t.Content)
	if t.Bold {
		escaped = "<strong>" + escaped + "</strong>"
	}
	if t.Muted {
		return template.HTML(`<span ` + styleMutedText + `>` + escaped + `</span>`)
	}
	return template.HTML(escaped)
}
