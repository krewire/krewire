package ui

import (
	"fmt"
	"html/template"
)

// Container is a max-width wrapper with responsive padding.
type Container struct {
	Content template.HTML `json:"content"`
}

// Render produces <div class="container">…</div>.
func (c Container) Render() template.HTML {
	return template.HTML(`<div class="container" data-ui-component="container">` + string(c.Content) + `</div>`)
}

// Grid is a responsive CSS grid.
type Grid struct {
	Columns int           `json:"columns"` // 1..6, defaults to 4 when <= 0
	Gap     string        `json:"gap"`     // optional CSS gap, e.g. "1.5rem"
	Content template.HTML `json:"content"`
}

// Render produces <div class="grid grid-{n}">…</div>.
func (g Grid) Render() template.HTML {
	cols := g.Columns
	if cols <= 0 || cols > 6 {
		cols = 4
	}
	style := ""
	if g.Gap != "" {
		style = ` style="gap:` + template.HTMLEscapeString(g.Gap) + `"`
	}
	return template.HTML(fmt.Sprintf(`<div class="grid grid-%d" data-ui-component="grid"%s>`, cols, style) + string(g.Content) + `</div>`)
}
