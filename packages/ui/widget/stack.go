package widget

import (
	"html/template"
	"strings"
)

// Stack represents a layout container that arranges widgets horizontally or vertically.
type Stack struct {
	Direction string // vertical or horizontal
	Widgets   []Widget
	Gap       string
	Align     string
}

// VStack creates a vertical stack.
func VStack(widgets ...Widget) *Stack {
	return &Stack{Direction: DirectionVertical, Widgets: widgets, Gap: defaultStackGap}
}

// HStack creates a horizontal stack.
func HStack(widgets ...Widget) *Stack {
	return &Stack{Direction: DirectionHorizontal, Widgets: widgets, Gap: defaultStackGap, Align: defaultStackAlign}
}

// WithGap sets the gap size.
func (s *Stack) WithGap(gap string) *Stack {
	s.Gap = gap
	return s
}

// Render produces the stack layout.
func (s *Stack) Render() template.HTML {
	var buf strings.Builder
	dirCls := modifier(stackClassPrefix, "v")
	if s.Direction == DirectionHorizontal {
		dirCls = modifier(stackClassPrefix, "h")
	}
	style := ""
	if s.Gap != "" {
		style = ` style="gap:` + template.HTMLEscapeString(s.Gap) + `;"`
	}
	buf.WriteString(`<div class="` + dirCls + `"` + style + `>`)
	for _, w := range s.Widgets {
		if w != nil {
			buf.WriteString(string(w.Render()))
		}
	}
	buf.WriteString(`</div>`)
	return template.HTML(buf.String())
}
