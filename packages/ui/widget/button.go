package widget

import "html/template"

// Button represents a clickable button or anchor.
type Button struct {
	Text     string
	URL      string
	Variant  string // primary, secondary, error, ghost, default
	Size     string // sm, md, lg
	Icon     string
	Type     string // button, submit, reset
	Disabled bool
	OnClick  string
	Class    string
}

// NewButton creates a new button with label.
func NewButton(text string) *Button {
	return &Button{
		Text:    text,
		Variant: VariantDefault,
		Size:    SizeMedium,
		Type:    "button",
	}
}

// Primary sets variant to primary.
func (b *Button) Primary() *Button {
	b.Variant = VariantPrimary
	return b
}

// Secondary sets variant to secondary.
func (b *Button) Secondary() *Button {
	b.Variant = VariantSecondary
	return b
}

// Error sets variant to error. `danger` was removed; see Alert.Error.
func (b *Button) Error() *Button {
	b.Variant = VariantError
	return b
}

// Small sets size to sm.
func (b *Button) Small() *Button {
	b.Size = SizeSmall
	return b
}

// Large sets size to lg.
func (b *Button) Large() *Button {
	b.Size = SizeLarge
	return b
}

// Link turns the button into an anchor link.
func (b *Button) Link(url string) *Button {
	b.URL = url
	return b
}

// WithIcon sets a leading icon.
func (b *Button) WithIcon(icon string) *Button {
	b.Icon = icon
	return b
}

// Submit marks button as form submit type.
func (b *Button) Submit() *Button {
	b.Type = "submit"
	return b
}

// Disable marks the button as disabled, which suppresses activation and is
// announced to assistive technology through the attribute rather than styling
// alone. It is named Disable rather than Disabled because the Disabled field is
// part of the exported struct.
func (b *Button) Disable() *Button {
	b.Disabled = true
	return b
}

// WithClickHandler sets an inline onclick handler. The value is attribute-escaped
// on render; inline handlers cannot be sanitized as thoroughly as a script the
// caller controls, so prefer a delegated listener where possible.
func (b *Button) WithClickHandler(handler string) *Button {
	b.OnClick = handler
	return b
}

// WithClass appends an extra CSS class alongside the computed variant classes.
func (b *Button) WithClass(class string) *Button {
	b.Class = class
	return b
}

// Render produces the button HTML.
func (b *Button) Render() template.HTML {
	cls := modifier(buttonClassPrefix, b.Variant)
	if b.Size != SizeMedium && b.Size != "" {
		cls += " " + modifier(buttonClassPrefix, b.Size)
	}
	if b.Class != "" {
		cls += " " + b.Class
	}

	content := template.HTMLEscapeString(b.Text)
	if b.Icon != "" {
		content = `<span class="btn-icon">` + template.HTMLEscapeString(b.Icon) + `</span> ` + content
	}

	if b.URL != "" {
		// SafeURL blocks javascript: and friends; HTML-escaping alone would not.
		return template.HTML(`<a href="` + SafeURL(b.URL) + `" class="` + template.HTMLEscapeString(cls) + `">` + content + `</a>`)
	}

	dis := ""
	if b.Disabled {
		dis = " disabled"
	}
	clk := ""
	if b.OnClick != "" {
		clk = ` onclick="` + template.HTMLEscapeString(b.OnClick) + `"`
	}

	btnType := b.Type
	if btnType == "" {
		btnType = "button"
	}

	return template.HTML(`<button type="` + template.HTMLEscapeString(btnType) + `" class="` + template.HTMLEscapeString(cls) + `"` + clk + dis + `>` + content + `</button>`)
}
