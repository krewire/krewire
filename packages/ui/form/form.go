package form

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

// Form represents a programmatic form builder.
type Form struct {
	Action      string
	Method      string
	Fields      []Field
	SubmitLabel string
	ValuesMap   map[string]string
	ErrorsMap   map[string][]string
	Class       string
}

// New creates a new form with action and method.
func New(action, method string) *Form {
	if method == "" {
		method = "POST"
	}
	return &Form{
		Action:      action,
		Method:      method,
		SubmitLabel: "Submit",
		ValuesMap:   make(map[string]string),
		ErrorsMap:   make(map[string][]string),
	}
}

// Add appends fields to the form.
func (f *Form) Add(fields ...Field) *Form {
	f.Fields = append(f.Fields, fields...)
	return f
}

// WithSubmitLabel sets the submit button text.
func (f *Form) WithSubmitLabel(label string) *Form {
	f.SubmitLabel = label
	return f
}

// SetValue sets the current value for a field.
func (f *Form) SetValue(name, val string) *Form {
	f.ValuesMap[name] = val
	return f
}

// Bind loads values from url.Values (query or form post).
func (f *Form) Bind(vals url.Values) *Form {
	for k := range vals {
		f.ValuesMap[k] = vals.Get(k)
	}
	return f
}

// BindRequest parses and binds form data from an http.Request.
func (f *Form) BindRequest(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	f.Bind(r.Form)
	return nil
}

// Validate executes all field validators and returns true if clean.
func (f *Form) Validate() bool {
	f.ErrorsMap = make(map[string][]string)
	isValid := true
	for _, field := range f.Fields {
		val := f.ValuesMap[field.Name()]
		errs := field.Validate(val)
		if len(errs) > 0 {
			f.ErrorsMap[field.Name()] = errs
			isValid = false
		}
	}
	return isValid
}

// Value returns the string value for a given field name.
func (f *Form) Value(name string) string {
	return f.ValuesMap[name]
}

// Values returns all form values as a map.
func (f *Form) Values() map[string]string {
	res := make(map[string]string, len(f.ValuesMap))
	for k, v := range f.ValuesMap {
		res[k] = v
	}
	return res
}

// Errors returns all validation errors.
func (f *Form) Errors() map[string][]string {
	return f.ErrorsMap
}

// Render produces the complete form HTML.
func (f *Form) Render() template.HTML {
	var b strings.Builder
	cls := "forge-form"
	if f.Class != "" {
		cls += " " + f.Class
	}
	b.WriteString(`<form class="` + template.HTMLEscapeString(cls) + `" action="` + template.HTMLEscapeString(f.Action) + `" method="` + template.HTMLEscapeString(f.Method) + `">`)
	for _, field := range f.Fields {
		val := f.ValuesMap[field.Name()]
		errs := f.ErrorsMap[field.Name()]
		b.WriteString(string(field.Render(val, errs)))
	}
	b.WriteString(`<div class="forge-form-actions" style="margin-top:0.5rem;">`)
	b.WriteString(`<button type="submit" class="forge-btn forge-btn-primary">` + template.HTMLEscapeString(f.SubmitLabel) + `</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</form>`)
	return template.HTML(b.String())
}
