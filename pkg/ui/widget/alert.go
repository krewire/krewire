package widget

import "html/template"

// Alert renders an informational or status callout box.
type Alert struct {
	Message string
	Variant string // info, success, warning, error
	Icon    string
}

// NewAlert creates a new alert box.
func NewAlert(message string) *Alert {
	return &Alert{Message: message, Variant: VariantInfo, Icon: iconInfo}
}

// Success sets variant to success.
func (a *Alert) Success() *Alert { a.Variant = VariantSuccess; a.Icon = iconSuccess; return a }

// Warning sets variant to warning.
func (a *Alert) Warning() *Alert { a.Variant = VariantWarning; a.Icon = iconWarning; return a }

// Error sets variant to error — the canonical name for the destructive
// status. `danger` was removed; `error` matches the `--error` theme token and
// the vocabulary framework/ui already uses.
func (a *Alert) Error() *Alert { a.Variant = VariantError; a.Icon = iconError; return a }

// Render produces the alert HTML.
func (a *Alert) Render() template.HTML {
	return template.HTML(`<div class="` + modifier(alertClassPrefix, a.Variant) + `">` +
		`<span class="alert-icon"` + styleAlertIcon + `>` + template.HTMLEscapeString(a.Icon) + `</span>` +
		`<span>` + template.HTMLEscapeString(a.Message) + `</span>` +
		`</div>`)
}
