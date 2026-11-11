package ui

import (
	"html/template"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/ui/node"
)

func TestDefaultRegistryHasStandardComponents(t *testing.T) {
	r := Default()
	for _, name := range []string{
		"Header", "Footer", "Nav", "Container", "Grid", "Card", "Section",
		"Hero", "ThemeToggle", "Badge", "Button",
	} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("default registry missing %q", name)
		}
	}
}

func TestRegistryRenderHeroMatchesBuiltin(t *testing.T) {
	r := Default()
	props := map[string]any{
		"title":    "Go Static",
		"subtitle": "Meta-framework.",
	}
	want := Hero{Title: template.HTML("Go Static"), Subtitle: template.HTML("Meta-framework.")}.Render()
	got, err := r.Render("Hero", props)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("registry Hero = %q, want %q", got, want)
	}
}

func TestRegistryRenderTypedProps(t *testing.T) {
	r := Default()
	h := Hero{Title: template.HTML("Typed"), Badge: template.HTML("v1")}
	got, err := r.Render("Hero", h)
	if err != nil {
		t.Fatal(err)
	}
	if got != h.Render() {
		t.Errorf("typed props Hero = %q, want %q", got, h.Render())
	}
}

func TestRegistryUnknownComponent(t *testing.T) {
	r := Default()
	if _, err := r.Render("Nope", nil); err == nil {
		t.Fatal("expected error for unknown component")
	}
	if _, ok := r.Get("Nope"); ok {
		t.Fatal("Get must return false for unknown component")
	}
}

func TestRegistryRejectsUnknownProps(t *testing.T) {
	r := Default()
	_, err := r.Render("Hero", map[string]any{"title": "x", "nope": "y"})
	if err == nil {
		t.Fatal("expected error for unknown prop key")
	}
}

func TestRegistryOverrideByName(t *testing.T) {
	r := Default()
	r.Register("Hero", typed(func(b Badge) template.HTML { return b.Render() }))
	got, err := r.Render("Hero", map[string]any{"text": "custom", "variant": "success"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `class="badge badge-success"`) {
		t.Errorf("override not applied: %q", got)
	}
}

func TestRegistryRenderCarriesScopeAttr(t *testing.T) {
	r := Default()
	for _, tc := range []struct {
		name  string
		props any
	}{
		{"Hero", map[string]any{"title": "Hi"}},
		{"Grid", map[string]any{"columns": 3}},
		{"Button", map[string]any{"text": "Go", "url": "/docs"}},
	} {
		out, err := r.Render(tc.name, tc.props)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !strings.Contains(string(out), `data-ui-component="`+strings.ToLower(tc.name)+`"`) {
			t.Errorf("%s output missing data-ui-component scope: %q", tc.name, out)
		}
	}
}

func TestRegistryLayoutComponentEquivalence(t *testing.T) {
	r := Default()
	out, err := r.Render("Footer", map[string]any{"right": "<p>© 2026</p>"})
	if err != nil {
		t.Fatal(err)
	}
	want := Footer{Right: template.HTML("<p>© 2026</p>")}.Render()
	if out != want {
		t.Errorf("registry Footer = %q, want %q", out, want)
	}
}

type CustomBadgeProps struct {
	Label   string `json:"label"`
	Variant string `json:"variant"`
}

func TestRegisterNodeComponent(t *testing.T) {
	r := NewRegistry()

	RegisterNode(r, "CustomBadge", func(props CustomBadgeProps, children ...node.Node) node.Node {
		v := node.VariantPrimary
		if props.Variant == "success" {
			v = node.VariantSuccess
		}
		var content []node.Node
		if props.Label != "" {
			content = append(content, node.Text(props.Label))
		}
		content = append(content, children...)
		return node.Badge(node.BadgeProps{Variant: v}, content...)
	}, ComponentMeta{
		Description: "A test custom badge node component",
		Props: map[string]PropSchema{
			"label":   {Type: "string", Required: true},
			"variant": {Type: "string", Default: "primary"},
		},
	})

	// 1. Verify inspectable metadata
	meta, ok := r.Meta("CustomBadge")
	if !ok {
		t.Fatal("expected CustomBadge metadata to exist")
	}
	if meta.Description != "A test custom badge node component" {
		t.Errorf("unexpected description: %s", meta.Description)
	}
	if !r.Has("CustomBadge") {
		t.Error("expected r.Has('CustomBadge') to be true")
	}

	// 2. Render with declarative map[string]any (e.g. from .kiw)
	out, err := r.Render("CustomBadge", map[string]any{
		"label":   "Verified",
		"variant": "success",
		"Body":    " <span>Extra</span>",
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "kiw-badge-success") || !strings.Contains(outStr, "Verified") || !strings.Contains(outStr, "<span>Extra</span>") {
		t.Errorf("unexpected rendered output: %s", outStr)
	}

	// 3. Render with typed struct props
	typedOut, err := r.Render("CustomBadge", CustomBadgeProps{
		Label:   "TypedBadge",
		Variant: "primary",
	})
	if err != nil {
		t.Fatalf("unexpected render error for typed props: %v", err)
	}
	if !strings.Contains(string(typedOut), "TypedBadge") || !strings.Contains(string(typedOut), "kiw-badge-primary") {
		t.Errorf("unexpected typed output: %s", string(typedOut))
	}
}

func TestBridgeToNode(t *testing.T) {
	r := Default()
	r.BridgeToNode()

	// Now invoke a registered component through node.Kiw
	kiwNode := node.Kiw("Badge", map[string]any{
		"text":    "Bridged Badge",
		"variant": "success",
	})

	html, err := node.RenderToHTML(nil, kiwNode)
	if err != nil {
		t.Fatalf("unexpected RenderToHTML error: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, "Bridged Badge") {
		t.Errorf("expected 'Bridged Badge' in bridged output: %s", out)
	}
}
