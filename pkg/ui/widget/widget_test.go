package widget

import (
	"strings"
	"testing"
)

func TestButtonRender(t *testing.T) {
	btn := NewButton("Click Me").Primary().WithIcon("🚀")
	html := string(btn.Render())
	if !strings.Contains(html, "forge-btn-primary") {
		t.Errorf("expected primary class, got %s", html)
	}
	if !strings.Contains(html, "Click Me") {
		t.Errorf("expected text Click Me, got %s", html)
	}
	if !strings.Contains(html, "🚀") {
		t.Errorf("expected icon, got %s", html)
	}

	link := NewButton("Go").Link("https://example.com").Error()
	linkHTML := string(link.Render())
	if !strings.Contains(linkHTML, "<a href=\"https://example.com\"") {
		t.Errorf("expected <a> tag, got %s", linkHTML)
	}
	if !strings.Contains(linkHTML, "forge-btn-error") {
		t.Errorf("expected error class, got %s", linkHTML)
	}
}

func TestBadgeRender(t *testing.T) {
	badge := NewBadge("Active").Success()
	html := string(badge.Render())
	if !strings.Contains(html, "forge-badge-success") {
		t.Errorf("expected success badge, got %s", html)
	}
	if !strings.Contains(html, "Active") {
		t.Errorf("expected badge text Active, got %s", html)
	}
}

func TestCardRender(t *testing.T) {
	card := NewCard("Project Alpha").
		WithDesc("Internal operations tool").
		WithBody(NewText("Content details").Strong()).
		WithFooter(NewButton("Details").Small())

	html := string(card.Render())
	if !strings.Contains(html, "Project Alpha") {
		t.Errorf("expected title, got %s", html)
	}
	if !strings.Contains(html, "Internal operations tool") {
		t.Errorf("expected description, got %s", html)
	}
	if !strings.Contains(html, "<strong>Content details</strong>") {
		t.Errorf("expected strong body, got %s", html)
	}
	if !strings.Contains(html, "forge-btn-sm") {
		t.Errorf("expected footer button, got %s", html)
	}
}

func TestAlertRender(t *testing.T) {
	alert := NewAlert("Operation succeeded").Success()
	html := string(alert.Render())
	if !strings.Contains(html, "forge-alert-success") {
		t.Errorf("expected alert success, got %s", html)
	}
	if !strings.Contains(html, "Operation succeeded") {
		t.Errorf("expected alert message, got %s", html)
	}
}

func TestTableRender(t *testing.T) {
	tbl := NewTable("ID", "Name", "Role")
	tbl.AddTextRow("1", "Alice", "Admin")
	tbl.AddTextRow("2", "Bob", "Developer")

	html := string(tbl.Render())
	if !strings.Contains(html, "<th>ID</th>") || !strings.Contains(html, "<td>Alice</td>") {
		t.Errorf("table missing content: %s", html)
	}
}

func TestStackAndGridRender(t *testing.T) {
	stack := VStack(NewText("First"), NewText("Second"))
	stackHTML := string(stack.Render())
	if !strings.Contains(stackHTML, "forge-stack-v") {
		t.Errorf("expected vertical stack class, got %s", stackHTML)
	}

	grid := NewGrid(3, NewCard("A"), NewCard("B"))
	gridHTML := string(grid.Render())
	if !strings.Contains(gridHTML, "forge-grid-3") {
		t.Errorf("expected grid-3 class, got %s", gridHTML)
	}
}
