package widget

import (
	"html/template"
	"strconv"
	"strings"
)

// Grid arranges items in a responsive grid.
type Grid struct {
	Columns int
	Widgets []Widget
}

// NewGrid creates a grid with column count (2, 3, 4).
func NewGrid(columns int, widgets ...Widget) *Grid {
	if columns < minGridColumns || columns > maxGridColumns {
		columns = defaultGridColumns
	}
	return &Grid{Columns: columns, Widgets: widgets}
}

// Render produces the grid layout.
func (g *Grid) Render() template.HTML {
	var buf strings.Builder
	cols := strconv.Itoa(g.Columns)
	if g.Columns < minGridColumns || g.Columns > maxGridColumns {
		cols = strconv.Itoa(defaultGridColumns)
	}
	cls := modifier(gridClassPrefix, cols)
	buf.WriteString(`<div class="` + cls + `">`)
	for _, w := range g.Widgets {
		if w != nil {
			buf.WriteString(string(w.Render()))
		}
	}
	buf.WriteString(`</div>`)
	return template.HTML(buf.String())
}
