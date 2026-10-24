package widget

import (
	"html/template"
	"strings"
)

// Fragment renders multiple widgets sequentially.
func Fragment(widgets ...Widget) template.HTML {
	var b strings.Builder
	for _, w := range widgets {
		if w != nil {
			b.WriteString(string(w.Render()))
		}
	}
	return template.HTML(b.String())
}
