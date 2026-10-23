package ui

import (
	"html/template"
	"strings"
)

// Hero is a prominent top section with headline, subtext, and CTA slots.
type Hero struct {
	Badge    template.HTML `json:"badge"`
	Title    template.HTML `json:"title"`
	Subtitle template.HTML `json:"subtitle"`
	CTALeft  template.HTML `json:"cta_left"`
	CTARight template.HTML `json:"cta_right"`
	Visual   template.HTML `json:"visual"` // optional illustration/terminal
	Extra    template.HTML `json:"extra"`
}

// Render produces <section class="hero">…</section>.
func (h Hero) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<section class="hero" data-ui-component="hero">`)
	b.WriteString(`<div class="hero-inner">`)
	b.WriteString(`<div class="hero-text">`)
	if h.Badge != "" {
		b.WriteString(`<div class="hero-badge">`)
		b.WriteString(string(h.Badge))
		b.WriteString(`</div>`)
	}
	if h.Title != "" {
		b.WriteString(`<h1 class="hero-title">`)
		b.WriteString(string(h.Title))
		b.WriteString(`</h1>`)
	}
	if h.Subtitle != "" {
		b.WriteString(`<p class="hero-subtitle">`)
		b.WriteString(string(h.Subtitle))
		b.WriteString(`</p>`)
	}
	if h.CTALeft != "" || h.CTARight != "" {
		b.WriteString(`<div class="hero-cta">`)
		if h.CTALeft != "" {
			b.WriteString(string(h.CTALeft))
		}
		if h.CTARight != "" {
			b.WriteString(string(h.CTARight))
		}
		b.WriteString(`</div>`)
	}
	if h.Extra != "" {
		b.WriteString(string(h.Extra))
	}
	b.WriteString(`</div>`)
	if h.Visual != "" {
		b.WriteString(`<div class="hero-visual">`)
		b.WriteString(string(h.Visual))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}
