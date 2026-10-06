package panel

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/krewire/krewire/packages/ui/widget"
)

// Panel represents an administrative panel or dashboard card.
type Panel struct {
	Title       string
	Description string
	Actions     widget.Widget
	Body        widget.Widget
	Footer      widget.Widget
}

// New creates a new panel.
func New(title string) *Panel {
	return &Panel{Title: title}
}

// WithDesc sets panel description.
func (p *Panel) WithDesc(desc string) *Panel {
	p.Description = desc
	return p
}

// WithActions sets header action widgets.
func (p *Panel) WithActions(actions widget.Widget) *Panel {
	p.Actions = actions
	return p
}

// WithBody sets the panel body content.
func (p *Panel) WithBody(body widget.Widget) *Panel {
	p.Body = body
	return p
}

// WithFooter sets the panel footer content.
func (p *Panel) WithFooter(footer widget.Widget) *Panel {
	p.Footer = footer
	return p
}

// Render produces the panel markup.
func (p *Panel) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<div class="forge-panel">`)
	if p.Title != "" || p.Actions != nil {
		b.WriteString(`<div class="forge-panel-head">`)
		b.WriteString(`<div>`)
		if p.Title != "" {
			b.WriteString(`<h3 class="forge-panel-title">` + template.HTMLEscapeString(p.Title) + `</h3>`)
		}
		if p.Description != "" {
			b.WriteString(`<p style="margin:0.25rem 0 0; font-size:0.8rem; color:var(--forge-muted)">` + template.HTMLEscapeString(p.Description) + `</p>`)
		}
		b.WriteString(`</div>`)
		if p.Actions != nil {
			b.WriteString(`<div>` + string(p.Actions.Render()) + `</div>`)
		}
		b.WriteString(`</div>`)
	}

	if p.Body != nil {
		b.WriteString(`<div class="forge-panel-body">` + string(p.Body.Render()) + `</div>`)
	}

	if p.Footer != nil {
		b.WriteString(`<div style="padding:0.75rem 1.25rem; border-top:1.5px solid var(--forge-border); background:var(--forge-surface)">` + string(p.Footer.Render()) + `</div>`)
	}

	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// Stat represents a KPI metric display widget.
type Stat struct {
	Label  string
	Value  string
	Change string
	Up     bool
	Icon   string
}

// NewStat creates a new metric stat card.
func NewStat(label, value string) *Stat {
	return &Stat{Label: label, Value: value}
}

// WithChange sets the delta percentage.
func (s *Stat) WithChange(change string, up bool) *Stat {
	s.Change = change
	s.Up = up
	return s
}

// WithIcon sets a stat icon.
func (s *Stat) WithIcon(icon string) *Stat {
	s.Icon = icon
	return s
}

// Render produces the stat card HTML.
func (s *Stat) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<div class="forge-stat">`)
	b.WriteString(`<div style="display:flex; justify-content:space-between; align-items:center;">`)
	b.WriteString(`<span class="forge-stat-label">` + template.HTMLEscapeString(s.Label) + `</span>`)
	if s.Icon != "" {
		b.WriteString(`<span style="font-size:1.25rem">` + template.HTMLEscapeString(s.Icon) + `</span>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div class="forge-stat-value">` + template.HTMLEscapeString(s.Value) + `</div>`)
	if s.Change != "" {
		cls := "forge-stat-change down"
		arrow := "↓"
		if s.Up {
			cls = "forge-stat-change up"
			arrow = "↑"
		}
		b.WriteString(`<div class="` + cls + `">` + arrow + ` ` + template.HTMLEscapeString(s.Change) + `</div>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// Dashboard represents an organized collection of panels and stats.
type Dashboard struct {
	Title   string
	Stats   []*Stat
	Panels  []*Panel
	Actions widget.Widget
}

// NewDashboard creates an application dashboard.
func NewDashboard(title string) *Dashboard {
	return &Dashboard{Title: title}
}

// AddStat adds a metric card.
func (d *Dashboard) AddStat(stats ...*Stat) *Dashboard {
	d.Stats = append(d.Stats, stats...)
	return d
}

// AddPanel adds a content panel.
func (d *Dashboard) AddPanel(panels ...*Panel) *Dashboard {
	d.Panels = append(d.Panels, panels...)
	return d
}

// WithActions adds top actions.
func (d *Dashboard) WithActions(actions widget.Widget) *Dashboard {
	d.Actions = actions
	return d
}

// Render produces the full dashboard markup.
func (d *Dashboard) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<div class="forge-dashboard" style="display:flex; flex-direction:column; gap:1.5rem;">`)

	// Head
	b.WriteString(`<div style="display:flex; justify-content:space-between; align-items:center;">`)
	b.WriteString(`<h1 style="font-size:1.75rem; font-weight:900; margin:0;">` + template.HTMLEscapeString(d.Title) + `</h1>`)
	if d.Actions != nil {
		b.WriteString(`<div>` + string(d.Actions.Render()) + `</div>`)
	}
	b.WriteString(`</div>`)

	// Stats row
	if len(d.Stats) > 0 {
		cols := len(d.Stats)
		if cols > 4 {
			cols = 4
		}
		b.WriteString(fmt.Sprintf(`<div class="forge-grid-%d">`, cols))
		for _, s := range d.Stats {
			b.WriteString(string(s.Render()))
		}
		b.WriteString(`</div>`)
	}

	// Panels
	for _, p := range d.Panels {
		b.WriteString(string(p.Render()))
	}

	b.WriteString(`</div>`)
	return template.HTML(b.String())
}
