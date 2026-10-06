package widget

import (
	"html/template"
	"strings"
)

// Card represents a structured container panel.
type Card struct {
	Title       string
	Description string
	Body        Widget
	Footer      Widget
	Actions     Widget
}

// NewCard creates a new card with title.
func NewCard(title string) *Card {
	return &Card{Title: title}
}

// WithDesc sets a card description.
func (c *Card) WithDesc(desc string) *Card {
	c.Description = desc
	return c
}

// WithBody sets the card's inner content.
func (c *Card) WithBody(body Widget) *Card {
	c.Body = body
	return c
}

// WithFooter sets the card footer.
func (c *Card) WithFooter(footer Widget) *Card {
	c.Footer = footer
	return c
}

// WithActions sets header action widgets.
func (c *Card) WithActions(actions Widget) *Card {
	c.Actions = actions
	return c
}

// Render produces the card HTML.
func (c *Card) Render() template.HTML {
	var buf strings.Builder
	buf.WriteString(`<div class="` + cardClassPrefix + `">`)

	if c.Title != "" || c.Description != "" || c.Actions != nil {
		buf.WriteString(`<div class="` + cardClassPrefix + `-header"` + styleCardHeader + `>`)
		buf.WriteString(`<div>`)
		if c.Title != "" {
			buf.WriteString(`<h3 class="` + cardClassPrefix + `-title">` + template.HTMLEscapeString(c.Title) + `</h3>`)
		}
		if c.Description != "" {
			buf.WriteString(`<p class="` + cardClassPrefix + `-desc">` + template.HTMLEscapeString(c.Description) + `</p>`)
		}
		buf.WriteString(`</div>`)
		if c.Actions != nil {
			buf.WriteString(`<div>` + string(c.Actions.Render()) + `</div>`)
		}
		buf.WriteString(`</div>`)
	}

	if c.Body != nil {
		buf.WriteString(`<div class="` + cardClassPrefix + `-body">` + string(c.Body.Render()) + `</div>`)
	}

	if c.Footer != nil {
		buf.WriteString(`<div class="` + cardClassPrefix + `-footer"` + styleCardFooter + `>` + string(c.Footer.Render()) + `</div>`)
	}

	buf.WriteString(`</div>`)
	return template.HTML(buf.String())
}
