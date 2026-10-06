package form

import (
	"html/template"
	"strings"
)

// CheckboxField represents a boolean toggle checkbox.
type CheckboxField struct {
	BaseField
}

// NewCheckbox creates a checkbox field.
func NewCheckbox(name, label string) *CheckboxField {
	return &CheckboxField{BaseField: BaseField{FieldName: name, FieldLabel: label}}
}

// Render produces checkbox markup.
//
// A checkbox is laid out inline: the control precedes its label, which is the
// opposite of every other field type, so it does not use openFieldGroup and
// assembles the wrapper itself.
func (f *CheckboxField) Render(value string, errors []string) template.HTML {
	chk := ""
	if isTruthy(value) {
		chk = " checked"
	}
	var b strings.Builder
	b.WriteString(`<div class="` + fieldGroupClass + `"` + styleCheckboxRow + `>`)
	b.WriteString(`<input type="checkbox" id="` + fieldID(f.FieldName) + `" name="` +
		template.HTMLEscapeString(f.FieldName) + `" value="true"` + chk + ` />`)
	b.WriteString(`<label class="` + fieldLabelClass + `" for="` + fieldID(f.FieldName) + `">` +
		template.HTMLEscapeString(f.FieldLabel) + `</label>`)
	writeErrors(&b, errors)
	closeFieldGroup(&b)
	return template.HTML(b.String())
}
