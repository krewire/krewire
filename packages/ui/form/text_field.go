package form

import (
	"fmt"
	"html/template"
	"net/mail"
	"strings"
)

// TextField represents a single-line text input.
type TextField struct {
	BaseField
	InputType string // text, email, password, etc.
}

// NewText creates a new text input field.
func NewText(name, label string) *TextField {
	return &TextField{
		BaseField: BaseField{FieldName: name, FieldLabel: label},
		InputType: "text",
	}
}

// MakeRequired marks the field as mandatory.
func (f *TextField) MakeRequired() *TextField {
	f.Required = true
	return f
}

// WithPlaceholder sets placeholder text.
func (f *TextField) WithPlaceholder(ph string) *TextField {
	f.Placeholder = ph
	return f
}

// WithHelp sets field help text.
func (f *TextField) WithHelp(help string) *TextField {
	f.Help = help
	return f
}

// AsEmail configures the field as email type.
func (f *TextField) AsEmail() *TextField {
	f.InputType = "email"
	f.Validators = append(f.Validators, func(val string) string {
		if val == "" {
			return ""
		}
		if _, err := mail.ParseAddress(val); err != nil {
			return "Invalid email address"
		}
		return ""
	})
	return f
}

// AsPassword configures the field as password type.
func (f *TextField) AsPassword() *TextField {
	f.InputType = "password"
	return f
}

// MinLength enforces minimum character length.
func (f *TextField) MinLength(min int) *TextField {
	f.Validators = append(f.Validators, func(val string) string {
		if val != "" && len(val) < min {
			return fmt.Sprintf("Must be at least %d characters", min)
		}
		return ""
	})
	return f
}

// Render produces the field markup.
func (f *TextField) Render(value string, errors []string) template.HTML {
	var b strings.Builder
	openFieldGroup(&b, &f.BaseField)

	b.WriteString(`<input class="` + fieldInputClass + `" id="` + fieldID(f.FieldName) + `" name="` + template.HTMLEscapeString(f.FieldName) + `" type="` + template.HTMLEscapeString(f.InputType) + `" value="` + template.HTMLEscapeString(value) + `"`)
	if f.Placeholder != "" {
		b.WriteString(` placeholder="` + template.HTMLEscapeString(f.Placeholder) + `"`)
	}
	if f.Required {
		b.WriteString(` required`)
	}
	b.WriteString(` />`)

	if f.Help != "" {
		b.WriteString(`<span class="forge-field-help">` + template.HTMLEscapeString(f.Help) + `</span>`)
	}
	writeErrors(&b, errors)
	closeFieldGroup(&b)
	return template.HTML(b.String())
}

// NumberField represents a numeric input.
