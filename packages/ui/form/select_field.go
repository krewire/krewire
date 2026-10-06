package form

import (
	"html/template"
	"strings"
)

// SelectOption represents a key-value choice.
type SelectOption struct {
	Value string
	Label string
}

// SelectField represents a dropdown choice input.
type SelectField struct {
	BaseField
	Options []SelectOption
}

// NewSelect creates a dropdown select field.
func NewSelect(name, label string, options ...SelectOption) *SelectField {
	return &SelectField{BaseField: BaseField{FieldName: name, FieldLabel: label}, Options: options}
}

// AddOption adds a choice.
func (f *SelectField) AddOption(value, label string) *SelectField {
	f.Options = append(f.Options, SelectOption{Value: value, Label: label})
	return f
}

// Render produces select markup.
func (f *SelectField) Render(value string, errors []string) template.HTML {
	var b strings.Builder
	openFieldGroup(&b, &f.BaseField)
	b.WriteString(`<select class="` + fieldSelectClass + `" id="` + fieldID(f.FieldName) + `" name="` + template.HTMLEscapeString(f.FieldName) + `">`)
	for _, opt := range f.Options {
		sel := ""
		if opt.Value == value {
			sel = " selected"
		}
		b.WriteString(`<option value="` + template.HTMLEscapeString(opt.Value) + `"` + sel + `>` + template.HTMLEscapeString(opt.Label) + `</option>`)
	}
	b.WriteString(`</select>`)
	writeErrors(&b, errors)
	closeFieldGroup(&b)
	return template.HTML(b.String())
}
