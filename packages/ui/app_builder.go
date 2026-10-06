package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/krewire/krewire/packages/ui/components"
	"github.com/krewire/krewire/packages/ui/form"
	"github.com/krewire/krewire/packages/ui/panel"
	"github.com/krewire/krewire/packages/ui/widget"
)

// ComponentsFS embeds all built-in UI components provided by Krewire Forge.
var ComponentsFS = components.FS

// DefaultTailwindCDN is the standard TailwindCSS Play CDN URL used by Forge by default.
const DefaultTailwindCDN = "https://cdn.tailwindcss.com"

// Document shell constants. Naming the markup fragments keeps RenderShell
// readable and confines the envelope layout to this one place.
const (
	// defaultDocumentLang is the lang attribute on the root element.
	defaultDocumentLang = "en"
	// titleSeparator joins the page name and the app title in <title>.
	titleSeparator = " — "
	// cssSuffix identifies a prebuilt stylesheet URL rather than a CDN script.
	cssSuffix = ".css"
	// bodyClass is applied to the root body element.
	bodyClass = "forge-app"
	// headerOpenTag opens the top navigation bar. The break belongs after
	// </header>, where RenderShell writes it, so the tag itself stays on one line.
	headerOpenTag = `<header style="background:var(--forge-surface); border-bottom:var(--forge-pop-border); padding:0.85rem 1.5rem; display:flex; align-items:center; justify-content:space-between;">`
	// brandMarkTag is the app mark shown at the head of the navigation bar.
	brandMarkTag = "<span style=\"background:var(--forge-primary); color:var(--forge-primary-content); width:28px; height:28px; display:inline-grid; place-items:center; border-radius:6px; font-weight:900; border:var(--forge-pop-border); box-shadow:var(--forge-pop-shadow-sm);\">◈</span>"
	// navOpenTag opens the navigation link list.
	navOpenTag = `<nav style="display:flex; align-items:center; gap:0.5rem;">`
	// mainOpenTag opens the main content canvas.
	mainOpenTag = `<main style="max-width:1200px; margin:0 auto; padding:2rem 1.5rem;">`
)

// Fallback palette values for the Tailwind config bridge. Each entry points at
// the CSS variable the theme already defines and repeats its value as a literal,
// so Tailwind's generated utilities keep their colour even when the variables
// resolve later. These are the Krewire defaults and live here so the shell does
// not inline hex values into a script string.
const (
	fallbackPrimary          = "#39D353"
	fallbackPrimaryContent   = "#0B1F3B"
	fallbackSecondary        = "#00D1C1"
	fallbackSecondaryContent = "#0B1F3B"
	fallbackAccent           = "#FF3B2E"
	fallbackSurface          = "#f0ede5"
	fallbackMuted            = "#475569"
)

// tailwindConfigScript configures Tailwind's dark-mode strategy and maps the
// theme's CSS variables onto Tailwind colour utilities.
const tailwindConfigScript = `  <script>
    tailwind.config = {
      darkMode: ['class', '[data-theme="dark"]'],
      theme: {
        extend: {
          colors: {
            primary: 'var(--forge-primary, ` + fallbackPrimary + `)',
            'primary-content': 'var(--forge-primary-content, ` + fallbackPrimaryContent + `)',
            secondary: 'var(--forge-secondary, ` + fallbackSecondary + `)',
            'secondary-content': 'var(--forge-secondary-content, ` + fallbackSecondaryContent + `)',
            accent: 'var(--forge-accent, ` + fallbackAccent + `)',
            surface: 'var(--forge-surface, ` + fallbackSurface + `)',
            muted: 'var(--forge-muted, ` + fallbackMuted + `)',
          }
        }
      }
    }
  </script>` + "\n"

// NavItem represents a link in the app navigation bar.
type NavItem struct {
	Label string
	URL   string
}

// App represents a programmatic web application built with Forge.
type App struct {
	Title       string
	Theme       *Theme
	Nav         []NavItem
	TailwindURL string // Defaults to DefaultTailwindCDN ("https://cdn.tailwindcss.com").
	mux         *http.ServeMux
	routes      map[string]bool
}

// New creates a new Forge programmatic application builder.
// By default, Krewire Forge uses TailwindCSS for modern utility styling.
func New(title string) *App {
	return &App{
		Title:       title,
		Theme:       DefaultTheme(),
		TailwindURL: DefaultTailwindCDN,
		mux:         http.NewServeMux(),
		routes:      make(map[string]bool),
	}
}

// WithTheme sets custom theme tokens.
func (a *App) WithTheme(t *Theme) *App {
	a.Theme = t
	return a
}

// WithTailwindURL configures a custom Tailwind stylesheet or script URL.
// Pass a local stylesheet like "/assets/tailwind.css" or a custom CDN script URL.
func (a *App) WithTailwindURL(url string) *App {
	a.TailwindURL = url
	return a
}

// DisableTailwind disables automatic TailwindCSS injection.
func (a *App) DisableTailwind() *App {
	a.TailwindURL = ""
	return a
}

// AddNav adds a navigation item.
func (a *App) AddNav(label, url string) *App {
	a.Nav = append(a.Nav, NavItem{Label: label, URL: url})
	return a
}

// Page registers a simple static or dynamic content page.
func (a *App) Page(path, title string, content widget.Widget) *App {
	a.routes[path] = true
	a.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := content.Render()
		fmt.Fprint(w, a.RenderShell(title, body))
	})
	return a
}

// Panel registers a panel page.
func (a *App) Panel(path string, p *panel.Panel) *App {
	return a.Page(path, p.Title, p)
}

// Dashboard registers a dashboard page.
func (a *App) Dashboard(path string, d *panel.Dashboard) *App {
	return a.Page(path, d.Title, d)
}

// Form registers an interactive form route with GET (render) and POST (submit) handlers.
func (a *App) Form(path string, f *form.Form, onSubmit func(values map[string]string) (widget.Widget, error)) *App {
	a.routes[path] = true
	a.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := f.BindRequest(r); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if f.Validate() {
				if onSubmit != nil {
					res, err := onSubmit(f.Values())
					if err != nil {
						alert := widget.NewAlert(err.Error()).Error()
						body := template.HTML(string(alert.Render()) + string(f.Render()))
						w.Header().Set("Content-Type", "text/html; charset=utf-8")
						fmt.Fprint(w, a.RenderShell("Error", body))
						return
					}
					if res != nil {
						w.Header().Set("Content-Type", "text/html; charset=utf-8")
						fmt.Fprint(w, a.RenderShell("Success", res.Render()))
						return
					}
				}
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, a.RenderShell(f.Action, f.Render()))
	})
	return a
}

// RenderShell renders the complete HTML envelope with theme and navigation.
func (a *App) RenderShell(pageTitle string, content template.HTML) template.HTML {
	var b strings.Builder

	a.writeHead(&b, pageTitle)
	a.writeHeader(&b)
	b.WriteString(mainOpenTag + "\n")
	b.WriteString(string(content))
	b.WriteString("\n</main>\n")

	b.WriteString("</body>\n</html>")
	return template.HTML(b.String())
}

// documentTitle returns the browser title for a page, qualifying the app title
// with the page name when one is supplied.
func (a *App) documentTitle(pageTitle string) string {
	if pageTitle == "" {
		return a.Title
	}
	return pageTitle + titleSeparator + a.Title
}

// writeHead emits the document head: metadata, title, theme variables, and the
// Tailwind stylesheet or script, then opens the body.
func (a *App) writeHead(b *strings.Builder, pageTitle string) {
	b.WriteString("<!doctype html>\n<html lang=\"" + defaultDocumentLang + "\">\n<head>\n")
	b.WriteString(`  <meta charset="utf-8">` + "\n")
	b.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	b.WriteString("  <title>" + template.HTMLEscapeString(a.documentTitle(pageTitle)) + "</title>\n")
	b.WriteString("  <style>\n" + string(a.Theme.CSS()) + "\n</style>\n")
	a.writeTailwind(b)
	b.WriteString("</head>\n<body class=\"" + bodyClass + "\">\n")
}

// writeTailwind emits the Tailwind entry point. A URL ending in ".css" is a
// prebuilt stylesheet and becomes a stylesheet link; anything else is treated as
// the Play CDN and additionally receives the Krewire token configuration so the
// theme palette reaches the utility classes.
func (a *App) writeTailwind(b *strings.Builder) {
	if a.TailwindURL == "" {
		return
	}
	url := template.HTMLEscapeString(a.TailwindURL)
	if strings.HasSuffix(a.TailwindURL, cssSuffix) {
		b.WriteString(`  <link rel="stylesheet" href="` + url + `">` + "\n")
		return
	}
	b.WriteString(`  <script src="` + url + `"></script>` + "\n")
	b.WriteString(tailwindConfigScript)
}

// writeHeader emits the top navigation bar containing the brand mark, the app
// title, and the configured navigation links.
func (a *App) writeHeader(b *strings.Builder) {
	b.WriteString(headerOpenTag)
	b.WriteString(`<div style="display:flex; align-items:center; gap:0.75rem;">`)
	b.WriteString(brandMarkTag)
	b.WriteString(`<span style="font-weight:900; font-size:1.1rem; letter-spacing:-0.03em;">` +
		template.HTMLEscapeString(a.Title) + `</span>`)
	b.WriteString(`</div>`)
	a.writeNav(b)
	b.WriteString("</header>\n")
}

// writeNav emits the navigation link list, or nothing when no links are
// configured. Link targets go through widget.SafeURL so an executable URL scheme
// cannot reach the rendered href.
func (a *App) writeNav(b *strings.Builder) {
	if len(a.Nav) == 0 {
		return
	}
	b.WriteString(navOpenTag)
	for _, item := range a.Nav {
		b.WriteString(`<a href="` + widget.SafeURL(item.URL) +
			`" style="font-size:0.85rem; font-weight:700; padding:0.4rem 0.75rem; border-radius:6px; color:var(--forge-fg); text-decoration:none;">` +
			template.HTMLEscapeString(item.Label) + `</a>`)
	}
	b.WriteString(`</nav>`)
}

// ServeHTTP dispatches requests to registered routes.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
