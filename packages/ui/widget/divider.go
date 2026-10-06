package widget

import (
	"fmt"
	"html/template"
)

// Divider renders a horizontal or vertical separator.
type Divider struct {
	Label string
}

// NewDivider creates a new divider.
func NewDivider(label ...string) *Divider {
	l := ""
	if len(label) > 0 {
		l = label[0]
	}
	return &Divider{Label: l}
}

// Render produces the divider.
func (d *Divider) Render() template.HTML {
	if d.Label == "" {
		return template.HTML(`<hr` + styleDividerRule + ` />`)
	}
	return template.HTML(fmt.Sprintf(`<div`+styleDividerWrap+`><hr`+styleDividerFill+` /><span`+styleDividerText+`>%s</span><hr`+styleDividerFill+` /></div>`, template.HTMLEscapeString(d.Label)))
}
