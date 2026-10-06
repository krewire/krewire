package form

import (
	"html/template"
	"strings"
)

// Field represents a form input field.
type Field interface {
	Name() string
	Label() string
	Validate(value string) []string
	Render(value string, errors []string) template.HTML
}

// BaseField provides shared attributes for inputs.
type BaseField struct {
	FieldName   string
	FieldLabel  string
	Help        string
	Required    bool
	Placeholder string
	Validators  []func(string) string
}

func (b *BaseField) Name() string  { return b.FieldName }
func (b *BaseField) Label() string { return b.FieldLabel }

func (b *BaseField) Validate(value string) []string {
	var errs []string
	if b.Required && strings.TrimSpace(value) == "" {
		errs = append(errs, b.FieldLabel+" is required")
		return errs
	}
	for _, v := range b.Validators {
		if msg := v(value); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}
