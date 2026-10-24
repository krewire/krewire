package book

import (
	"embed"
	"html/template"
)

// content embeds the site templates and shipped assets.
//
//go:embed templates assets
var content embed.FS

// parseTemplates loads the page templates from the embedded filesystem.
func parseTemplates() (*template.Template, error) {
	return template.ParseFS(content, "templates/*.tmpl")
}

// Assets returns the site's shipped assets: name to file body. The theme-mode
// variables and toggle button styles are appended to the reader stylesheet so
// consumers never hardcode them.
func Assets() map[string]string {
	body, err := content.ReadFile("assets/krewire.css")
	if err != nil {
		return map[string]string{"assets/krewire.css": "/* krewire default stylesheet */"}
	}
	css := string(body) + "\n" + ThemeModeVarsCSS + "\n" + ThemeToggleCSS
	return map[string]string{"assets/krewire.css": css}
}
