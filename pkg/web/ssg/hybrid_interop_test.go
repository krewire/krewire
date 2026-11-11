package ssg_test

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/ui"
	"github.com/krewire/krewire/pkg/ui/node"
	"github.com/krewire/krewire/pkg/web/ssg"
)

// UserMetricsProps defines typed props for a Go component.
type UserMetricsProps struct {
	UserID string `json:"userId"`
	Period string `json:"period"`
}

// UserMetricsCard is a native Go component returning a composable node.Node tree.
func UserMetricsCard(props UserMetricsProps, children ...node.Node) node.Node {
	return node.Card(
		node.CardProps{
			Title: "User Analytics: " + props.UserID,
			Slots: node.SlotMap{
				"footer": node.Badge(node.BadgeProps{Variant: node.VariantSuccess}, node.Text("Period: "+props.Period)),
			},
		},
		node.VStack(
			node.StackProps{Gap: "gap-2"},
			node.Paragraph(node.TextProps{Bold: true}, node.Text("Live Metrics Summary")),
			node.Fragment(children...),
		),
	)
}

func TestHybridGoInKiw(t *testing.T) {
	site := ssg.New()
	site.Layout(ssg.Layout{
		Name: "default",
		Body: "<!DOCTYPE html><html><body>{{.Content}}</body></html>",
	})
	reg := site.GetRegistry()

	// Register Go component with inspectable metadata
	ui.RegisterNode(reg, "UserMetricsCard", UserMetricsCard, ui.ComponentMeta{
		Description: "Displays user analytics and metrics card",
		Props: map[string]ui.PropSchema{
			"userId": {Type: "string", Required: true},
			"period": {Type: "string", Default: "monthly"},
		},
	})

	// Define a .kiw page that invokes the Go component
	dashboardPage := &ssg.Page{
		Path:   "/dashboard",
		Root:   "DashboardPage",
		Layout: "default",
	}
	site.Page(*dashboardPage)

	site.Component(ssg.Component{
		Name: "DashboardPage",
		Body: `<div class="dashboard">
  <h1>Dashboard Overview</h1>
  {{component "UserMetricsCard" (dict "userId" "usr_999" "period" "weekly" "Body" "<p>Custom body from .kiw</p>")}}
</div>`,
	})

	out, err := site.RenderPage(dashboardPage)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	outStr := string(out)
	for _, expected := range []string{
		"Dashboard Overview",
		"User Analytics: usr_999",
		"Period: weekly",
		"Live Metrics Summary",
		"Custom body from .kiw",
		"kiw-card",
		"kiw-badge-success",
	} {
		if !strings.Contains(outStr, expected) {
			t.Errorf("expected %q in rendered output, got:\n%s", expected, outStr)
		}
	}
}

func TestHybridKiwInGo(t *testing.T) {
	site := ssg.New()

	// Register a template component
	site.Component(ssg.Component{
		Name: "KiwNotification",
		Body: `<div class="kiw-notification" data-status="{{.Status}}">
  <span class="msg">{{.Message}}</span>
  <div class="content">{{.Body}}</div>
</div>`,
	})
	// Connect site's template components to node.Kiw
	site.BridgeToNode()

	// Now build a Go node tree that embeds the .kiw component
	goPage := node.Container(
		node.ContainerProps{MaxWidth: "max-w-xl", Centered: true},
		node.Heading(node.HeadingProps{Level: node.Heading2}, node.Text("System Status")),
		node.Kiw("KiwNotification", map[string]any{
			"Status":  "online",
			"Message": "All systems operational",
		}, node.Text("Additional diagnostics detail.")),
	)

	html, err := node.RenderToHTML(context.Background(), goPage)
	if err != nil {
		t.Fatalf("unexpected node RenderToHTML error: %v", err)
	}

	htmlStr := string(html)
	for _, expected := range []string{
		"System Status",
		"kiw-notification",
		`data-status="online"`,
		"All systems operational",
		"Additional diagnostics detail.",
	} {
		if !strings.Contains(htmlStr, expected) {
			t.Errorf("expected %q in rendered output, got:\n%s", expected, htmlStr)
		}
	}
}

func TestHybridGoRejectsUnknownProps(t *testing.T) {
	site := ssg.New()
	site.Layout(ssg.Layout{
		Name: "default",
		Body: "<!DOCTYPE html><html><body>{{.Content}}</body></html>",
	})
	reg := site.GetRegistry()

	ui.RegisterNode(reg, "StrictUserCard", UserMetricsCard)

	p := &ssg.Page{
		Path:   "/strict-test",
		Root:   "StrictPage",
		Layout: "default",
	}
	site.Page(*p)

	site.Component(ssg.Component{
		Name: "StrictPage",
		Body: `<div class="test">
  {{component "StrictUserCard" (dict "userId" "123" "unknownTypoField" "invalid")}}
</div>`,
	})

	_, err := site.RenderPage(p)
	if err == nil {
		t.Fatal("expected error due to unknown prop key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("expected error message mentioning unknown field, got: %v", err)
	}
}
