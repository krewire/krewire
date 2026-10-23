package ui

import "html/template"

// ThemeToggle wraps the theme button in a stable container.
type ThemeToggle struct {
	Theme *Theme `json:"theme"`
}

// Render returns the theme-switcher button wrapped in a stable container div.
func (t ThemeToggle) Render() template.HTML {
	if t.Theme == nil {
		t.Theme = &Theme{}
	}
	return template.HTML(`<div class="theme-toggle-wrap" data-ui-component="theme-toggle">` +
		string(t.Theme.Button()) + `</div>`)
}
