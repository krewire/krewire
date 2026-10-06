package panel

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/ui/widget"
)

func TestPanelRender(t *testing.T) {
	p := New("Sales Overview").
		WithDesc("Monthly transactional metrics").
		WithActions(widget.NewButton("Export").Small()).
		WithBody(widget.NewText("Detailed stats table")).
		WithFooter(widget.NewText("Last updated 5m ago").Subtle())

	html := string(p.Render())
	if !strings.Contains(html, "Sales Overview") {
		t.Errorf("missing panel title in %s", html)
	}
	if !strings.Contains(html, "Monthly transactional metrics") {
		t.Errorf("missing panel desc in %s", html)
	}
	if !strings.Contains(html, "forge-btn-sm") {
		t.Errorf("missing action button in %s", html)
	}
	if !strings.Contains(html, "Detailed stats table") {
		t.Errorf("missing panel body in %s", html)
	}
}

func TestStatAndDashboardRender(t *testing.T) {
	stat1 := NewStat("Revenue", "$48,250").WithChange("18.4%", true).WithIcon("💰")
	stat2 := NewStat("Active Users", "1,420").WithChange("2.1%", false).WithIcon("👥")

	dash := NewDashboard("Analytics").
		AddStat(stat1, stat2).
		AddPanel(New("Recent Events").WithBody(widget.NewText("No incidents reported.")))

	html := string(dash.Render())
	if !strings.Contains(html, "Analytics") {
		t.Errorf("missing dashboard title in %s", html)
	}
	if !strings.Contains(html, "$48,250") {
		t.Errorf("missing stat revenue in %s", html)
	}
	if !strings.Contains(html, "forge-stat-change up") {
		t.Errorf("missing positive trend in %s", html)
	}
	if !strings.Contains(html, "forge-stat-change down") {
		t.Errorf("missing negative trend in %s", html)
	}
	if !strings.Contains(html, "Recent Events") {
		t.Errorf("missing panel in %s", html)
	}
}
