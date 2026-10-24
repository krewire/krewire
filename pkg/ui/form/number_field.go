package form

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

type NumberField struct {
	BaseField
	Min *float64
	Max *float64
}

// NewNumber creates a numeric input field.
func NewNumber(name, label string) *NumberField {
	nf := &NumberField{BaseField: BaseField{FieldName: name, FieldLabel: label}}
	nf.Validators = append(nf.Validators, func(val string) string {
		if val == "" {
			return ""
		}
		num, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return "Must be a valid number"
		}
		if nf.Min != nil && num < *nf.Min {
			return fmt.Sprintf("Must be at least %v", *nf.Min)
		}
		if nf.Max != nil && num > *nf.Max {
			return fmt.Sprintf("Must be at most %v", *nf.Max)
		}
		return ""
	})
	return nf
}

// SetMin sets the minimum value.
func (f *NumberField) SetMin(min float64) *NumberField {
	f.Min = &min
	return f
}

// SetMax sets the maximum value.
func (f *NumberField) SetMax(max float64) *NumberField {
	f.Max = &max
	return f
}

// Render produces the numeric input markup.
func (f *NumberField) Render(value string, errors []string) template.HTML {
	var b strings.Builder
	openFieldGroup(&b, &f.BaseField)
	b.WriteString(`<input class="` + fieldInputClass + `" id="` + fieldID(f.FieldName) + `" name="` + template.HTMLEscapeString(f.FieldName) + `" type="number" value="` + template.HTMLEscapeString(value) + `"`)
	if f.Min != nil {
		b.WriteString(fmt.Sprintf(` min="%v"`, *f.Min))
	}
	if f.Max != nil {
		b.WriteString(fmt.Sprintf(` max="%v"`, *f.Max))
	}
	b.WriteString(` />`)
	writeErrors(&b, errors)
	closeFieldGroup(&b)
	return template.HTML(b.String())
}
