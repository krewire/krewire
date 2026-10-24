package form

import (
	"fmt"
	"html/template"
	"strings"
)

// TextareaField represents a multi-line text input.
type TextareaField struct {
	BaseField
	Rows int
}

// NewTextarea creates a textarea field.
func NewTextarea(name, label string) *TextareaField {
	return &TextareaField{BaseField: BaseField{FieldName: name, FieldLabel: label}, Rows: 4}
}

// Render produces textarea markup.
func (f *TextareaField) Render(value string, errors []string) template.HTML {
	var b strings.Builder
	openFieldGroup(&b, &f.BaseField)
	b.WriteString(fmt.Sprintf(`<textarea class="`+fieldTextareaCls+`" id="%s" name="%s" rows="%d">%s</textarea>`,
		fieldID(f.FieldName), template.HTMLEscapeString(f.FieldName), f.Rows, template.HTMLEscapeString(value)))
	writeErrors(&b, errors)
	closeFieldGroup(&b)
	return template.HTML(b.String())
}
