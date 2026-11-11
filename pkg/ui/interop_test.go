package ui_test

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/ui"
	"github.com/krewire/krewire/pkg/ui/node"
)

// MetricCardProps defines the typed props for a Go metric card component.
type MetricCardProps struct {
	Title string  `json:"title"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

func TestMixedComposition(t *testing.T) {
	r := ui.NewRegistry()

	// 1. Register a native Go node component
	ui.RegisterNode(r, "MetricCard", func(props MetricCardProps, children ...node.Node) node.Node {
		return node.Card(node.CardProps{
			Title: props.Title,
			Slots: node.SlotMap{
				"footer": node.HStack(node.StackProps{Justify: node.JustifyEnd}, children...),
			},
		},
			node.Textf("%.1f %s", props.Value, props.Unit),
		)
	}, ui.ComponentMeta{
		Description: "Displays a numeric KPI with unit",
		Props: map[string]ui.PropSchema{
			"title": {Type: "string", Required: true},
			"value": {Type: "number", Required: true},
			"unit":  {Type: "string", Default: "%"},
		},
	})

	// Bridge registry to node adapter
	r.BridgeToNode()

	// 2. Compose a full mixed page in native Go containing both Go nodes and bridged .kiw nodes
	page := node.Container(
		node.ContainerProps{MaxWidth: "max-w-5xl", Centered: true},
		node.VStack(
			node.StackProps{Gap: "gap-6"},
			node.Heading(node.HeadingProps{Level: node.Heading1}, node.Text("Analytics Dashboard")),
			// Invoke the MetricCard through the interoperability node.Kiw contract
			node.Kiw("MetricCard", map[string]any{
				"title": "CPU Utilization",
				"value": 42.5,
				"unit":  "%",
			}, node.Badge(node.BadgeProps{Variant: node.VariantSuccess}, node.Text("Healthy"))),
		),
	)

	// 3. Render through canonical pipeline
	out, err := node.RenderToString(context.Background(), page)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	for _, expected := range []string{
		"Analytics Dashboard",
		"CPU Utilization",
		"42.5 %",
		"Healthy",
		"kiw-badge-success",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in mixed page output: %s", expected, out)
		}
	}
}

func TestComponentMetadataInspectionWithoutExecution(t *testing.T) {
	r := ui.Default()

	ui.RegisterNode(r, "StatCard", func(props MetricCardProps, children ...node.Node) node.Node {
		t.Fatal("component code should not be executed during inspection")
		return nil
	}, ui.ComponentMeta{
		Package:     "@krewire/ui",
		Version:     "1.0.0",
		Description: "Static KPI Card",
		Props: map[string]ui.PropSchema{
			"title": {Type: "string", Required: true},
		},
		Slots: []string{"header", "footer"},
	})

	// Inspect metadata directly
	meta, ok := r.Meta("StatCard")
	if !ok {
		t.Fatal("expected StatCard metadata")
	}
	if meta.Package != "@krewire/ui" || meta.Version != "1.0.0" {
		t.Errorf("unexpected package metadata: %+v", meta)
	}
	if len(meta.Slots) != 2 || meta.Slots[0] != "header" {
		t.Errorf("unexpected slots metadata: %+v", meta.Slots)
	}

	all := r.AllMeta()
	if len(all) == 0 {
		t.Error("expected non-empty AllMeta")
	}
}

func TestUnknownPropsRejectionIdentifiesComponent(t *testing.T) {
	r := ui.NewRegistry()

	type StrictProps struct {
		Name string `json:"name"`
	}

	ui.RegisterNode(r, "StrictUser", func(props StrictProps, children ...node.Node) node.Node {
		return node.Text(props.Name)
	})

	// Pass an unknown prop typo
	_, err := r.Render("StrictUser", map[string]any{
		"name":  "Alice",
		"typoo": "invalid",
	})
	if err == nil {
		t.Fatal("expected error for unknown property key")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("expected unknown field error message, got: %v", err)
	}
}
