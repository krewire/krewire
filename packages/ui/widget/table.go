package widget

import (
	"html/template"
	"strings"
)

// Table represents a data table widget.
type Table struct {
	Headers []string
	Rows    [][]template.HTML
}

// NewTable creates a new table with headers.
func NewTable(headers ...string) *Table {
	return &Table{Headers: headers}
}

// AddRow appends a row of cells to the table.
func (t *Table) AddRow(cells ...template.HTML) *Table {
	t.Rows = append(t.Rows, cells)
	return t
}

// AddTextRow appends a row of plain strings.
func (t *Table) AddTextRow(cells ...string) *Table {
	row := make([]template.HTML, len(cells))
	for i, c := range cells {
		row[i] = template.HTML(template.HTMLEscapeString(c))
	}
	t.Rows = append(t.Rows, row)
	return t
}

// Render produces the table HTML.
func (t *Table) Render() template.HTML {
	var buf strings.Builder
	buf.WriteString(`<div class="` + tableClassPrefix + `-wrap">`)
	buf.WriteString(`<table class="` + tableClassPrefix + `">`)
	if len(t.Headers) > 0 {
		buf.WriteString(`<thead><tr>`)
		for _, h := range t.Headers {
			buf.WriteString(`<th>` + template.HTMLEscapeString(h) + `</th>`)
		}
		buf.WriteString(`</tr></thead>`)
	}
	buf.WriteString(`<tbody>`)
	for _, row := range t.Rows {
		buf.WriteString(`<tr>`)
		for _, cell := range row {
			buf.WriteString(`<td>` + string(cell) + `</td>`)
		}
		buf.WriteString(`</tr>`)
	}
	buf.WriteString(`</tbody>`)
	buf.WriteString(`</table>`)
	buf.WriteString(`</div>`)
	return template.HTML(buf.String())
}
