package book

import (
	"html/template"
	"reflect"
	"strconv"
	"strings"
)

const defaultThemeKey = "krewire-theme"

const ThemeModeVarsCSS = `:root { --show-sun: block; --show-moon: none; }
:root[data-theme="dark"] { --show-sun: none; --show-moon: block; }
@media (prefers-color-scheme: dark) { :root:not([data-theme]) { --show-sun: none; --show-moon: block; } }`

const ThemeToggleCSS = `.theme-toggle { border: var(--pop-border-thin, 2px solid #0B1F3B); box-shadow: var(--pop-shadow-sm, 2px 2px 0px #0B1F3B); border-radius: 8px; background: var(--base-2, #f0ede5); color: var(--ink, #0B1F3B); width: 36px; height: 36px; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.15s ease; }
.theme-toggle:hover { background: var(--base-3, #e3dfd5); transform: translate(-1px, -1px); box-shadow: 3px 3px 0px var(--border, #0B1F3B); }
.theme-toggle:active { transform: translate(1px, 1px); box-shadow: 1px 1px 0px var(--border, #0B1F3B); }
.theme-toggle svg { width: 17px; height: 17px; display: block; }
.theme-toggle .icon-sun { display: var(--show-sun); }
.theme-toggle .icon-moon { display: var(--show-moon); }`

type Color string

type Palette struct {
	Base1            Color `css:"base-1"`
	Base1Content     Color `css:"base-1-content"`
	Base2            Color `css:"base-2"`
	Base2Content     Color `css:"base-2-content"`
	Base3            Color `css:"base-3"`
	Base3Content     Color `css:"base-3-content"`
	Primary          Color `css:"primary"`
	PrimaryContent   Color `css:"primary-content"`
	Secondary        Color `css:"secondary"`
	SecondaryContent Color `css:"secondary-content"`
	Accent           Color `css:"accent"`
	AccentContent    Color `css:"accent-content"`
	Ghost            Color `css:"ghost"`
	GhostContent     Color `css:"ghost-content"`
	Neutral          Color `css:"neutral"`
	NeutralContent   Color `css:"neutral-content"`
	Success          Color `css:"success"`
	SuccessContent   Color `css:"success-content"`
	Info             Color `css:"info"`
	InfoContent      Color `css:"info-content"`
	Warning          Color `css:"warning"`
	WarningContent   Color `css:"warning-content"`
	Error            Color `css:"error"`
	ErrorContent     Color `css:"error-content"`
}

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
	Error:            "#f87171",
	ErrorContent:     "#260a0a",
}

func (p Palette) cssVars(defaults Palette) string {
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
		b.WriteString("--")
		b.WriteString(kiw.Type().Field(i).Tag.Get("css"))
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString(";")
	}
	return b.String()
}

type Theme struct {
	StorageKey string
	Default    string
	Light      Palette
	Dark       Palette
}

// DefaultTheme returns a Theme configured for automatic light/dark switching.
func DefaultTheme() *Theme {
	return &Theme{
		Default: "auto",
		Light:   DefaultLightPalette,
		Dark:    DefaultDarkPalette,
	}
}

func (t Theme) storageKey() string {
	if t.StorageKey == "" {
		return defaultThemeKey
	}
	return t.StorageKey
}

func (t Theme) defaultTheme() string {
	switch t.Default {
	case "light", "dark":
		return t.Default
	default:
		return "auto"
	}
}

func (t Theme) Style() template.CSS {
	light := ":root{" + t.Light.cssVars(DefaultLightPalette) + "}"
	dark := ":root[data-theme=\"dark\"]{" + t.Dark.cssVars(DefaultDarkPalette) + "}"
	fallback := "@media (prefers-color-scheme: dark){:root:not([data-theme]){" +
		t.Dark.cssVars(DefaultDarkPalette) + "}}"
	return template.CSS("<style>" + light + dark + fallback + "</style>")
}

func (t Theme) Script() template.HTML {
	return template.HTML(string(t.Style()) + "<script>" + themeJS(t.storageKey(), t.defaultTheme()) + "</script>")
}

func (t Theme) Button() template.HTML {
	return template.HTML(`<button type="button" class="theme-toggle" data-theme-toggle aria-label="Toggle light/dark theme" title="Toggle theme"><svg class="icon-sun" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg><svg class="icon-moon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg></button>`)
}

func themeJS(key, def string) string {
	var b strings.Builder
	b.WriteString(`(function(){var k=` + strconv.Quote(key) + `,d=` + strconv.Quote(def) + `,t=d;try{t=localStorage.getItem(k)||d}catch(e){}var m=window.matchMedia('(prefers-color-scheme: dark)');function apply(){var dark=t==='dark'||(t==='auto'&&m.matches);var r=document.documentElement;r.dataset.theme=dark?'dark':'light';r.style.colorScheme=dark?'dark':'light';var meta=document.querySelector('meta[name="color-scheme"]');if(meta)meta.content=dark?'dark':'light'}apply();if(m.addEventListener){m.addEventListener('change',function(){if(t==='auto')apply()})}window.krewireTheme={get:function(){return t},set:function(x){t=x;try{localStorage.setItem(k,x)}catch(e){}apply()},toggle:function(){var dark=t==='dark'||(t==='auto'&&m.matches);window.krewireTheme.set(dark?'light':'dark')}};if(!window._krewireThemeBound){window._krewireThemeBound=true;document.addEventListener('click',function(e){var el=e.target&&e.target.closest?e.target.closest('[data-theme-toggle]'):null;if(el){e.preventDefault();window.krewireTheme.toggle()}})}})();`)
	return b.String()
}
