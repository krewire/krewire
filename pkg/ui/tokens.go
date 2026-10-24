package ui

import "fmt"

// Design-token values that are not derived from a Palette. Naming them keeps a
// change to the visual language to a single edit instead of a search through the
// generated stylesheet.
const (
	// radiusToken is the base corner radius for UI surfaces.
	radiusToken = "10px"
	// popBorderToken is the pop-brutalist outline applied to surfaces.
	popBorderToken = "2px solid var(--ui-border, var(--forge-border))"
	// popShadowToken and popShadowSmallToken are the hard offset shadows.
	popShadowToken      = "3px 3px 0px var(--ui-border, var(--forge-border))"
	popShadowSmallToken = "2px 2px 0px var(--ui-border, var(--forge-border))"
	// fontSansToken and fontMonoToken are the default font stacks.
	fontSansToken = "Inter, system-ui, -apple-system, sans-serif"
	fontMonoToken = "ui-monospace, SFMono-Regular, Menlo, monospace"
)

// colorTokenDecls is the --ui-* (and legacy --forge-*) colour declaration list for one palette.
// Light and dark declare the identical variable set, so both render from this
// single template instead of maintaining two copies that can drift apart.
const colorTokenDecls = `
  --ui-primary: var(--primary, %[1]s);
  --forge-primary: var(--ui-primary);
  --ui-primary-content: var(--primary-content, %[2]s);
  --forge-primary-content: var(--ui-primary-content);
  --ui-secondary: var(--secondary, %[3]s);
  --forge-secondary: var(--ui-secondary);
  --ui-secondary-content: var(--secondary-content, %[4]s);
  --forge-secondary-content: var(--ui-secondary-content);
  --ui-accent: var(--accent, %[5]s);
  --forge-accent: var(--ui-accent);
  --ui-bg: var(--base-1, %[6]s);
  --forge-bg: var(--ui-bg);
  --ui-surface: var(--base-2, %[7]s);
  --forge-surface: var(--ui-surface);
  --ui-fg: var(--base-1-content, %[8]s);
  --forge-fg: var(--ui-fg);
  --ui-muted: var(--neutral-content, %[9]s);
  --forge-muted: var(--ui-muted);
  --ui-border: var(--base-1-content, %[10]s);
  --forge-border: var(--ui-border);
  --ui-success: var(--success, %[11]s);
  --forge-success: var(--ui-success);
  --ui-warning: var(--warning, %[12]s);
  --forge-warning: var(--ui-warning);
  --ui-error: var(--error, %[13]s);
  --forge-error: var(--ui-error);
`

// geometryTokenDecls holds the mode-independent tokens. They are declared once
// on :root and inherited by the dark block, so they are written only once.
const geometryTokenDecls = `
  --ui-radius: ` + radiusToken + `;
  --forge-radius: var(--ui-radius);
  --ui-pop-border: ` + popBorderToken + `;
  --forge-pop-border: var(--ui-pop-border);
  --ui-pop-shadow: ` + popShadowToken + `;
  --forge-pop-shadow: var(--ui-pop-shadow);
  --ui-pop-shadow-sm: ` + popShadowSmallToken + `;
  --forge-pop-shadow-sm: var(--ui-pop-shadow-sm);
  --ui-font-sans: ` + fontSansToken + `;
  --forge-font-sans: var(--ui-font-sans);
  --ui-font-mono: ` + fontMonoToken + `;
  --forge-font-mono: var(--ui-font-mono);
`

// paletteArgs returns the ordered fallback colours that colorTokenDecls expects.
func paletteArgs(p Palette) []any {
	return []any{
		p.Primary, p.PrimaryContent, p.Secondary, p.SecondaryContent, p.Accent,
		p.Base1, p.Base2, p.Base1Content, p.NeutralContent, p.Base1Content,
		p.Success, p.Warning, p.Error,
	}
}

// uiTokenCSS renders the custom-property declarations for both modes: the
// geometry tokens on :root, then the per-mode colour tokens.
func uiTokenCSS() string {
	return ":root {" + geometryTokenDecls + fmt.Sprintf(colorTokenDecls, paletteArgs(DefaultLightPalette)...) + "}\n" +
		"\n[data-theme=\"dark\"], .dark {" + fmt.Sprintf(colorTokenDecls, paletteArgs(DefaultDarkPalette)...) + "}\n"
}
