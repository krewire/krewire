package ui

import "fmt"

// Design-token values that are not derived from a Palette. Naming them keeps a
// change to the visual language to a single edit instead of a search through the
// generated stylesheet.
const (
	// radiusToken is the base corner radius for UI surfaces.
	radiusToken = "10px"
	// popBorderToken is the pop-brutalist outline applied to surfaces.
	popBorderToken = "2px solid var(--ui-border)"
	// popShadowToken and popShadowSmallToken are the hard offset shadows.
	popShadowToken      = "3px 3px 0px var(--ui-border)"
	popShadowSmallToken = "2px 2px 0px var(--ui-border)"
	// fontSansToken and fontMonoToken are the default font stacks.
	fontSansToken = "Inter, system-ui, -apple-system, sans-serif"
	fontMonoToken = "ui-monospace, SFMono-Regular, Menlo, monospace"
)

// colorTokenDecls is the --ui-* colour declaration list for one palette.
// Light and dark declare the identical variable set, so both render from this
// single template instead of maintaining two copies that can drift apart.
const colorTokenDecls = `
  --ui-primary: var(--primary, %[1]s);
  --ui-primary-content: var(--primary-content, %[2]s);
  --ui-secondary: var(--secondary, %[3]s);
  --ui-secondary-content: var(--secondary-content, %[4]s);
  --ui-accent: var(--accent, %[5]s);
  --ui-bg: var(--base-1, %[6]s);
  --ui-surface: var(--base-2, %[7]s);
  --ui-fg: var(--base-1-content, %[8]s);
  --ui-muted: var(--neutral-content, %[9]s);
  --ui-border: var(--base-1-content, %[10]s);
  --ui-success: var(--success, %[11]s);
  --ui-warning: var(--warning, %[12]s);
  --ui-error: var(--error, %[13]s);
`

// geometryTokenDecls holds the mode-independent tokens. They are declared once
// on :root and inherited by the dark block, so they are written only once.
const geometryTokenDecls = `
  --ui-radius: ` + radiusToken + `;
  --ui-pop-border: ` + popBorderToken + `;
  --ui-pop-shadow: ` + popShadowToken + `;
  --ui-pop-shadow-sm: ` + popShadowSmallToken + `;
  --ui-font-sans: ` + fontSansToken + `;
  --ui-font-mono: ` + fontMonoToken + `;
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
