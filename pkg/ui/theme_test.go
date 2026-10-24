package ui

import (
	"strings"
	"testing"
)

func TestThemeScriptDefaults(t *testing.T) {
	s := string(Theme{}.Script())
	for _, want := range []string{
		"<script>",
		"krewire-theme",
		"localStorage",
		"(prefers-color-scheme: dark)",
		"data-theme-toggle",
		"auto",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("default script missing %q", want)
		}
	}
}

func TestThemeScriptCustom(t *testing.T) {
	s := string(Theme{StorageKey: "site-theme", Default: "dark"}.Script())
	if !strings.Contains(s, `k="site-theme"`) {
		t.Errorf("script missing custom storage key: %q", s)
	}
	if !strings.Contains(s, `d="dark"`) {
		t.Errorf("script missing custom default: %q", s)
	}
}

func TestThemeButton(t *testing.T) {
	b := string(Theme{}.Button())
	for _, want := range []string{
		`data-theme-toggle`,
		`aria-label="Toggle light/dark theme"`,
		`icon-sun`,
		`icon-moon`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("button missing %q", want)
		}
	}
}

func TestThemeStorageKeyAndDefault(t *testing.T) {
	if got := (Theme{}).StorageKeyOrDefault(); got != "krewire-theme" {
		t.Errorf("default storage key = %q", got)
	}
	if got := (Theme{Default: "bogus"}).DefaultTheme(); got != "auto" {
		t.Errorf("invalid default = %q, want auto", got)
	}
	if got := (Theme{Default: "light"}).DefaultTheme(); got != "light" {
		t.Errorf("default = %q, want light", got)
	}
}

func TestThemeCSS(t *testing.T) {
	th := DefaultTheme()
	css := string(th.CSS())
	if !strings.Contains(css, "--ui-primary") {
		t.Errorf("CSS missing --ui-primary")
	}
	if !strings.Contains(css, ".ui-btn") {
		t.Errorf("CSS missing .ui-btn")
	}
}

func TestThemeCSSScrollbar(t *testing.T) {
	css := string(DefaultTheme().CSS())
	for _, want := range []string{
		"--ui-scrollbar-thumb:",
		"scrollbar-color: var(--ui-scrollbar-thumb) var(--ui-scrollbar-track)",
		"::-webkit-scrollbar",
		"::-webkit-scrollbar-thumb",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("CSS missing scrollbar rule %q", want)
		}
	}
}

func TestThemeTailwindConfigScript(t *testing.T) {
	th := DefaultTheme()
	s := string(th.TailwindConfigScript())
	if !strings.Contains(s, "tailwind.config") {
		t.Errorf("missing tailwind.config in %s", s)
	}
	if !strings.Contains(s, "var(--ui-primary") {
		t.Errorf("missing ui color token mapping in %s", s)
	}
}

// countDecl reports how many times a custom property is declared in the sheet.
func countDecl(css, name string) int {
	return strings.Count(css, name+":")
}

// TestCSSDeclaresTokens verifies the generated stylesheet still carries
// every design token. The rules for light and dark are generated from one shared
// template, so a missing palette field would silently drop a declaration.
func TestCSSDeclaresTokens(t *testing.T) {
	css := string(DefaultTheme().CSS())

	// Mode-independent tokens belong to :root only, so they are declared once.
	for _, token := range []string{
		"--ui-radius", "--ui-pop-border", "--ui-pop-shadow",
		"--ui-pop-shadow-sm", "--ui-font-sans", "--ui-font-mono",
	} {
		if got := countDecl(css, token); got != 1 {
			t.Errorf("%s declared %d times, want 1", token, got)
		}
	}

	// Colour tokens are declared for both modes.
	for _, token := range []string{
		"--ui-primary", "--ui-primary-content", "--ui-secondary",
		"--ui-accent", "--ui-bg", "--ui-surface", "--ui-fg",
		"--ui-muted", "--ui-border", "--ui-success", "--ui-warning",
		"--ui-error",
	} {
		if got := countDecl(css, token); got != 2 {
			t.Errorf("%s declared %d times, want 2 (light and dark)", token, got)
		}
	}
}

// TestCSSUsesPaletteFallbacks verifies the token block resolves through
// the default palettes, so a palette change is reflected without editing CSS.
func TestCSSUsesPaletteFallbacks(t *testing.T) {
	css := string(DefaultTheme().CSS())
	if !strings.Contains(css, "var(--primary, "+string(DefaultLightPalette.Primary)+")") {
		t.Error("the light primary colour must come from DefaultLightPalette")
	}
	if !strings.Contains(css, "var(--primary, "+string(DefaultDarkPalette.Primary)+")") {
		t.Error("the dark primary colour must come from DefaultDarkPalette")
	}
	if !strings.Contains(css, `[data-theme="dark"]`) {
		t.Error("the dark-mode selector is missing")
	}
}

// TestCSSNilReceiver verifies CSS on a nil Theme falls back to the
// defaults instead of panicking, since App.RenderShell calls it directly.
func TestCSSNilReceiver(t *testing.T) {
	var nilTheme *Theme
	if css := string(nilTheme.CSS()); !strings.Contains(css, "--ui-primary") {
		t.Error("a nil Theme must render the default token block")
	}
}

// TestStorageKeyAndDefaultMode covers the storage-key fallback and the
// mode coercion rules.
func TestStorageKeyAndDefaultMode(t *testing.T) {
	if got := (Theme{}).StorageKeyOrDefault(); got != defaultThemeKey {
		t.Errorf("StorageKeyOrDefault = %q, want %q", got, defaultThemeKey)
	}
	if got := (Theme{StorageKey: "custom"}).StorageKeyOrDefault(); got != "custom" {
		t.Errorf("StorageKeyOrDefault = %q, want custom", got)
	}
	for _, mode := range []string{"light", "dark", "auto"} {
		if got := (Theme{Default: mode}).DefaultTheme(); got != mode {
			t.Errorf("DefaultTheme(%q) = %q", mode, got)
		}
	}
	// An unset or unrecognised value falls back to "auto".
	for _, mode := range []string{"", "sepia", "DARK"} {
		if got := (Theme{Default: mode}).DefaultTheme(); got != "auto" {
			t.Errorf("DefaultTheme(%q) = %q, want auto", mode, got)
		}
	}
}

// TestBuildersAreChainable covers the fluent setters.
func TestBuildersAreChainable(t *testing.T) {
	th := DefaultTheme().WithStorageKey("k").WithDefault("dark")
	if th.StorageKey != "k" || th.Default != "dark" {
		t.Errorf("builders produced %+v", th)
	}
	if th.Name != "krewire" {
		t.Errorf("New must default the name, got %q", th.Name)
	}
	if th.Light.Primary == "" || th.Dark.Primary == "" {
		t.Error("New must populate both palettes")
	}
}

// TestScriptQuotesStorageKey verifies the runtime script receives the
// configured key, safely quoted so a key containing a quote cannot break out of
// the JavaScript string literal.
func TestScriptQuotesStorageKey(t *testing.T) {
	script := string(Theme{StorageKey: `we"ird`, Default: "dark"}.Script())
	if !strings.Contains(script, `we\"ird`) {
		t.Errorf("the storage key must be safely quoted in the script:\n%s", script)
	}
	if !strings.Contains(script, `"dark"`) {
		t.Error("the default mode must appear in the script")
	}
	if !strings.Contains(script, "<style>") {
		t.Error("Script must embed the palette style")
	}
}

// TestCSSIncludesSharedBlocks verifies the shared constants are actually
// wired into the sheet rather than declared and forgotten.
func TestCSSIncludesSharedBlocks(t *testing.T) {
	css := string(DefaultTheme().CSS())
	if !strings.Contains(css, ThemeModeVarsCSS) {
		t.Error("ThemeModeVarsCSS must be included")
	}
	if !strings.Contains(css, ".theme-toggle") {
		t.Error("ThemeToggleCSS must be included")
	}
	if !strings.Contains(css, ScrollbarCSS) {
		t.Error("ScrollbarCSS must be included")
	}
}
