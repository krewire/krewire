package ui

import "fmt"

// Design-token values that are not derived from a Palette. Naming them keeps a
// change to the visual language to a single edit instead of a search through the
// generated stylesheet.
const (
	// radiusToken is the base corner radius for Forge surfaces.
	radiusToken = "10px"
	// popBorderToken is the pop-brutalist outline applied to surfaces.
	popBorderToken = "2px solid var(--forge-border)"
	// popShadowToken and popShadowSmallToken are the hard offset shadows.
	popShadowToken      = "3px 3px 0px var(--forge-border)"
	popShadowSmallToken = "2px 2px 0px var(--forge-border)"
	// fontSansToken and fontMonoToken are the default font stacks.
	fontSansToken = "Inter, system-ui, -apple-system, sans-serif"
	fontMonoToken = "ui-monospace, SFMono-Regular, Menlo, monospace"
)

// colorTokenDecls is the --forge-* colour declaration list for one palette.
// Light and dark declare the identical variable set, so both render from this
// single template instead of maintaining two copies that can drift apart.
const colorTokenDecls = `
  --forge-primary: var(--primary, %[1]s);
  --forge-primary-content: var(--primary-content, %[2]s);
  --forge-secondary: var(--secondary, %[3]s);
  --forge-secondary-content: var(--secondary-content, %[4]s);
  --forge-accent: var(--accent, %[5]s);
  --forge-bg: var(--base-1, %[6]s);
  --forge-surface: var(--base-2, %[7]s);
  --forge-fg: var(--base-1-content, %[8]s);
  --forge-muted: var(--neutral-content, %[9]s);
  --forge-border: var(--base-1-content, %[10]s);
  --forge-success: var(--success, %[11]s);
  --forge-warning: var(--warning, %[12]s);
  --forge-error: var(--error, %[13]s);
`

// geometryTokenDecls holds the mode-independent tokens. They are declared once
// on :root and inherited by the dark block, so they are written only once.
const geometryTokenDecls = `
  --forge-radius: ` + radiusToken + `;
  --forge-pop-border: ` + popBorderToken + `;
  --forge-pop-shadow: ` + popShadowToken + `;
  --forge-pop-shadow-sm: ` + popShadowSmallToken + `;
  --forge-font-sans: ` + fontSansToken + `;
  --forge-font-mono: ` + fontMonoToken + `;
`

// paletteArgs returns the ordered fallback colours that colorTokenDecls expects.
func paletteArgs(p Palette) []any {
	return []any{
		p.Primary, p.PrimaryContent, p.Secondary, p.SecondaryContent, p.Accent,
		p.Base1, p.Base2, p.Base1Content, p.NeutralContent, p.Base1Content,
		p.Success, p.Warning, p.Error,
	}
}

// forgeTokenCSS renders the custom-property declarations for both modes: the
// geometry tokens on :root, then the per-mode colour tokens.
func forgeTokenCSS() string {
	return ":root {" + geometryTokenDecls + fmt.Sprintf(colorTokenDecls, paletteArgs(DefaultLightPalette)...) + "}\n" +
		"\n[data-theme=\"dark\"], .dark {" + fmt.Sprintf(colorTokenDecls, paletteArgs(DefaultDarkPalette)...) + "}\n"
}
