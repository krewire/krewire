package widget

import (
	"html/template"
	"strings"
)

// Badge represents an inline status or label tag.
type Badge struct {
	Text    string
	Variant string // primary, secondary, success, warning, error, default
	Icon    string
}

// NewBadge creates a new badge with text.
func NewBadge(text string) *Badge {
	return &Badge{Text: text, Variant: VariantDefault}
}

// Primary sets badge variant to primary.
func (bg *Badge) Primary() *Badge { bg.Variant = VariantPrimary; return bg }

// Secondary sets badge variant to secondary.
func (bg *Badge) Secondary() *Badge { bg.Variant = VariantSecondary; return bg }

// Success sets badge variant to success.
func (bg *Badge) Success() *Badge { bg.Variant = VariantSuccess; return bg }

// Warning sets badge variant to warning.
func (bg *Badge) Warning() *Badge { bg.Variant = VariantWarning; return bg }

// Error sets badge variant to error. `danger` was removed; see Alert.Error.
func (bg *Badge) Error() *Badge { bg.Variant = VariantError; return bg }

// WithIcon adds an icon to the badge.
func (bg *Badge) WithIcon(icon string) *Badge { bg.Icon = icon; return bg }

// Render produces the badge markup.
func (bg *Badge) Render() template.HTML {
	var buf strings.Builder
	buf.WriteString(`<span class="` + badgeClassPrefix + ` ` + modifier(badgeClassPrefix, bg.Variant) + `">`)
	if bg.Icon != "" {
		buf.WriteString(`<span class="badge-icon">` + template.HTMLEscapeString(bg.Icon) + `</span>`)
	}
	buf.WriteString(template.HTMLEscapeString(bg.Text))
	buf.WriteString(`</span>`)
	return template.HTML(buf.String())
}
