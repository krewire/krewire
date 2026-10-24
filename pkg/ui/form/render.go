package form

import (
	"html/template"
	"strings"
)

// Class names for the form elements. The field id prefix is a contract: the
// label's `for` attribute and the control's `id` are derived from it, so a
// mismatch silently breaks click-to-focus and screen-reader association.
const (
	fieldGroupClass  = "forge-field-group"
	fieldLabelClass  = "forge-label"
	fieldRequiredCls = "forge-required"
	fieldInputClass  = "forge-input"
	fieldTextareaCls = "forge-textarea"
	fieldSelectClass = "forge-select"
	fieldErrorClass  = "forge-field-error"
	fieldFormClass   = "forge-form"
	fieldActionsCls  = "forge-form-actions"
	// fieldIDPrefix is prepended to a field name to form the control id.
	fieldIDPrefix = "field-"
)

// fieldID returns the DOM id for a named field.
func fieldID(name string) string {
	return fieldIDPrefix + template.HTMLEscapeString(name)
}

// openFieldGroup writes the wrapper and the opening label, including the
// required marker, so every field type presents its label identically.
func openFieldGroup(b *strings.Builder, f *BaseField) {
	b.WriteString(`<div class="` + fieldGroupClass + `">`)
	b.WriteString(`<label class="` + fieldLabelClass + `" for="` + fieldID(f.FieldName) + `">` +
		template.HTMLEscapeString(f.FieldLabel))
	if f.Required {
		b.WriteString(` <span class="` + fieldRequiredCls + `">*</span>`)
	}
	b.WriteString(`</label>`)
}

// writeErrors renders the validation messages for a field. Each message is
// escaped: messages can embed the rejected value.
func writeErrors(b *strings.Builder, errors []string) {
	for _, e := range errors {
		b.WriteString(`<span class="` + fieldErrorClass + `">` + template.HTMLEscapeString(e) + `</span>`)
	}
}

// closeFieldGroup writes the closing wrapper.
func closeFieldGroup(b *strings.Builder) {
	b.WriteString(`</div>`)
}

// checkedValues are the submitted values a browser sends for a checked box. A
// checkbox is only ever present in the request when checked, so any of these
// means checked and anything else means unchecked.
var checkedValues = map[string]bool{
	"true": true,
	"on":   true,
	"1":    true,
}

// isTruthy reports whether a submitted checkbox value means checked.
func isTruthy(value string) bool { return checkedValues[value] }

// styleCheckboxRow lays a checkbox out inline with its label, which is the only
// field type whose control follows its label.
const styleCheckboxRow = ` style="flex-direction:row; align-items:center; gap:0.5rem;"`
