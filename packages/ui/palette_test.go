package ui

import (
	"strings"
	"testing"
)

func TestPaletteDefaults(t *testing.T) {
	if DefaultLightPalette.Primary != "#39D353" {
		t.Errorf("expected primary #39D353, got %q", DefaultLightPalette.Primary)
	}
	if DefaultDarkPalette.Primary != "#39D353" {
		t.Errorf("expected primary #39D353, got %q", DefaultDarkPalette.Primary)
	}
	if DefaultLightPalette.Base1 != "#FAF8F4" {
		t.Errorf("expected base1 #FAF8F4, got %q", DefaultLightPalette.Base1)
	}
	if DefaultDarkPalette.Base1 != "#0B1F3B" {
		t.Errorf("expected dark base1 #0B1F3B, got %q", DefaultDarkPalette.Base1)
	}
}

func TestPaletteCSSVars(t *testing.T) {
	p := Palette{Primary: "#123456"}
	vars := p.CSSVars(DefaultLightPalette)
	if !strings.Contains(vars, "--primary: #123456;") {
		t.Errorf("expected custom primary var, got %s", vars)
	}
	if !strings.Contains(vars, "--base-1: #FAF8F4;") {
		t.Errorf("expected default base-1 fallback, got %s", vars)
	}
}
