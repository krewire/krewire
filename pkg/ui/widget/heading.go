package widget

import (
	"fmt"
	"html/template"
)

// Heading renders an h1-h6 tag.
type Heading struct {
	Level   int
	Content string
}

// NewHeading creates a heading of specified level (1..6).
func NewHeading(level int, content string) *Heading {
	if level < minHeadingLevel || level > maxHeadingLevel {
		level = defaultHeadingLevel
	}
	return &Heading{Level: level, Content: content}
}

// Render produces the <hN> element.
func (h *Heading) Render() template.HTML {
	return template.HTML(fmt.Sprintf(`<h%d class="`+headingClassPrefix+`"`+styleHeading+`>%s</h%d>`,
		h.Level, template.HTMLEscapeString(h.Content), h.Level))
}
