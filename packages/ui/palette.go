// Package ui provides the theming system for the Krewire ecosystem.
// It manages color palettes, light/dark mode switching, design tokens,
// and CSS variables for web applications and UI components.
package ui

import (
	"reflect"
	"strings"
)

// Color is a CSS color value, e.g. "#0b58a1", "rgb(11 88 161)", or
// "oklch(0.5 0.2 250)".
type Color string

// Palette is a named color schema for a single mode (light or dark). Every
// base color has a -content companion: the color of text and icons rendered
// on top of it. Empty fields fall back to the mode's default palette, so a
// site can override just a few colors.
type Palette struct {
	Base1            Color `css:"base-1" json:"base_1,omitempty"`
	Base1Content     Color `css:"base-1-content" json:"base_1_content,omitempty"`
	Base2            Color `css:"base-2" json:"base_2,omitempty"`
	Base2Content     Color `css:"base-2-content" json:"base_2_content,omitempty"`
	Base3            Color `css:"base-3" json:"base_3,omitempty"`
	Base3Content     Color `css:"base-3-content" json:"base_3_content,omitempty"`
	Primary          Color `css:"primary" json:"primary,omitempty"`
	PrimaryContent   Color `css:"primary-content" json:"primary_content,omitempty"`
	Secondary        Color `css:"secondary" json:"secondary,omitempty"`
	SecondaryContent Color `css:"secondary-content" json:"secondary_content,omitempty"`
	Accent           Color `css:"accent" json:"accent,omitempty"`
	AccentContent    Color `css:"accent-content" json:"accent_content,omitempty"`
	Ghost            Color `css:"ghost" json:"ghost,omitempty"`
	GhostContent     Color `css:"ghost-content" json:"ghost_content,omitempty"`
	Neutral          Color `css:"neutral" json:"neutral,omitempty"`
	NeutralContent   Color `css:"neutral-content" json:"neutral_content,omitempty"`
	Success          Color `css:"success" json:"success,omitempty"`
	SuccessContent   Color `css:"success-content" json:"success_content,omitempty"`
	Info             Color `css:"info" json:"info,omitempty"`
	InfoContent      Color `css:"info-content" json:"info_content,omitempty"`
	Warning          Color `css:"warning" json:"warning,omitempty"`
	WarningContent   Color `css:"warning-content" json:"warning_content,omitempty"`
	Error            Color `css:"error" json:"error,omitempty"`
	ErrorContent     Color `css:"error-content" json:"error_content,omitempty"`
}

// DefaultLightPalette is the palette applied when no light override is set.
var DefaultLightPalette = Palette{
	Base1:            "#FAF8F4",
	Base1Content:     "#0B1F3B",
	Base2:            "#f0ede5",
	Base2Content:     "#475569",
	Base3:            "#e3dfd5",
	Base3Content:     "#334155",
	Primary:          "#39D353",
	PrimaryContent:   "#0B1F3B",
	Secondary:        "#00D1C1",
	SecondaryContent: "#0B1F3B",
	Accent:           "#FF3B2E",
	AccentContent:    "#ffffff",
	Ghost:            "#e9e6dc",
	GhostContent:     "#475569",
	Neutral:          "#e3dfd5",
	NeutralContent:   "#475569",
	Success:          "#39D353",
	SuccessContent:   "#0B1F3B",
	Info:             "#00D1C1",
	InfoContent:      "#0B1F3B",
	Warning:          "#f59e0b",
	WarningContent:   "#0B1F3B",
	Error:            "#FF3B2E",
	ErrorContent:     "#ffffff",
}

// DefaultDarkPalette is the palette applied when no dark override is set.
var DefaultDarkPalette = Palette{
	Base1:            "#0B1F3B",
	Base1Content:     "#FAF8F4",
	Base2:            "#112646",
	Base2Content:     "#8fa2bc",
	Base3:            "#1a355d",
	Base3Content:     "#c4d1e2",
	Primary:          "#39D353",
	PrimaryContent:   "#0B1F3B",
	Secondary:        "#00D1C1",
	SecondaryContent: "#0B1F3B",
	Accent:           "#FF3B2E",
	AccentContent:    "#ffffff",
	Ghost:            "#162e52",
	GhostContent:     "#8fa2bc",
	Neutral:          "#1a355d",
	NeutralContent:   "#8fa2bc",
	Success:          "#39D353",
	SuccessContent:   "#0B1F3B",
	Info:             "#00D1C1",
	InfoContent:      "#0B1F3B",
	Warning:          "#fbbf24",
	WarningContent:   "#0B1F3B",
	Error:            "#FF3B2E",
	ErrorContent:     "#ffffff",
}

// CSSVars emits "name: value;" declarations for every set color, filling in
// non-empty fields from overrides and empty fields from the defaults palette.
func (p Palette) CSSVars(defaults Palette) string {
	var b strings.Builder
	kiw, rdd := reflect.ValueOf(p), reflect.ValueOf(defaults)
	for i := 0; i < kiw.NumField(); i++ {
		v := kiw.Field(i).String()
		if v == "" {
			v = rdd.Field(i).String()
		}
		if v == "" {
			continue
		}
		tag := kiw.Type().Field(i).Tag.Get("css")
		if tag == "" {
			continue
		}
		b.WriteString("--")
		b.WriteString(tag)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString(";")
	}
	return b.String()
}
