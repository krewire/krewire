package node_test

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/ui/node"
	"github.com/krewire/krewire/pkg/ui/widget"
)

func TestTextEscaping(t *testing.T) {
	ctx := context.Background()
	malicious := "<script>alert('xss') & \"quoted\"</script>"
	txt := node.Text(malicious)

	out, err := node.RenderToString(ctx, txt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "<script>") {
		t.Errorf("expected script tag to be escaped, got %s", out)
	}
	expected := template.HTMLEscapeString(malicious)
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestRawHTML(t *testing.T) {
	ctx := context.Background()
	svg := `<svg class="w-4 h-4"><circle cx="2" cy="2" r="2"/></svg>`
	raw := node.RawHTML(svg)

	out, err := node.RenderToString(ctx, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != svg {
		t.Errorf("expected %q, got %q", svg, out)
	}
}

func TestElementVoid(t *testing.T) {
	ctx := context.Background()
	inp := node.Element("input", node.Attrs{
		"type":     "text",
		"name":     "username",
		"disabled": true,
	})

	out, err := node.RenderToString(ctx, inp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "<input") || !strings.HasSuffix(out, " />") {
		t.Errorf("expected self-closing input tag, got %s", out)
	}
	if !strings.Contains(out, "disabled") {
		t.Errorf("expected boolean attribute disabled in %s", out)
	}
	if !strings.Contains(out, `name="username"`) {
		t.Errorf("expected name attribute in %s", out)
	}
}

func TestElementAttrsEscaping(t *testing.T) {
	ctx := context.Background()
	el := node.Element("div", node.Attrs{
		"title":       `<hello & "world">`,
		"data-active": false,
		"data-ready":  true,
		"data-count":  42,
	})

	out, err := node.RenderToString(ctx, el)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "<hello") {
		t.Errorf("expected attribute value to be escaped, got %s", out)
	}
	if strings.Contains(out, "data-active") {
		t.Errorf("expected false boolean attribute to be omitted, got %s", out)
	}
	if !strings.Contains(out, "data-ready") {
		t.Errorf("expected true boolean attribute to be present, got %s", out)
	}
	if !strings.Contains(out, `data-count="42"`) {
		t.Errorf("expected numeric attribute formatted as 42, got %s", out)
	}
}

func TestSafeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://krewire.com", "https://krewire.com"},
		{"/docs/overview", "/docs/overview"},
		{"mailto:hello@krewire.com", "mailto:hello@krewire.com"},
		{"tel:+1234567890", "tel:+1234567890"},
		{"#section-1", "#section-1"},
		{"javascript:alert(1)", "#"},
		{"javascript:void(0)", "#"},
		{"data:text/html,<script>alert(1)</script>", "#"},
		{"vbscript:msgbox(1)", "#"},
		{"java\tscript:alert(1)", "#"},
		{"", ""},
	}

	for _, tc := range tests {
		got := node.SafeURL(tc.input)
		if got != tc.expected {
			t.Errorf("SafeURL(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestFragment(t *testing.T) {
	ctx := context.Background()
	frag := node.Fragment(
		node.Text("Hello"),
		nil, // nil node should be filtered safely
		node.Text(" "),
		node.Text("World"),
	)

	out, err := node.RenderToString(ctx, frag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", out)
	}
}

func TestSlotResolution(t *testing.T) {
	ctx := context.Background()

	// 1. Fallback when slot is not provided
	fallbackTree := node.Element("div", nil,
		node.Slot("header", node.Text("Default Header")),
		node.Text("Body Content"),
	)
	out, err := node.RenderToString(ctx, fallbackTree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Default Header") {
		t.Errorf("expected fallback header in %s", out)
	}

	// 2. Override when slot is provided via WithSlots
	customTree := node.WithSlots(fallbackTree, node.SlotMap{
		"header": node.Element("h1", nil, node.Text("Custom Header")),
	})
	outCustom, err := node.RenderToString(ctx, customTree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outCustom, "<h1>Custom Header</h1>") {
		t.Errorf("expected custom header in %s", outCustom)
	}
	if strings.Contains(outCustom, "Default Header") {
		t.Errorf("expected fallback header to be replaced in %s", outCustom)
	}
}

func TestLayoutComponents(t *testing.T) {
	ctx := context.Background()

	// Container
	container := node.Container(node.ContainerProps{
		MaxWidth: "max-w-4xl",
		Padding:  "px-6 py-12",
		Centered: true,
	}, node.Text("Container Content"))

	cOut, err := node.RenderToString(ctx, container)
	if err != nil {
		t.Fatalf("container error: %v", err)
	}
	if !strings.Contains(cOut, "kiw-container") || !strings.Contains(cOut, "mx-auto") || !strings.Contains(cOut, "max-w-4xl") {
		t.Errorf("unexpected container output: %s", cOut)
	}

	// VStack and HStack
	vstack := node.VStack(node.StackProps{
		Gap:   "gap-4",
		Align: node.AlignCenter,
	}, node.Text("Item 1"), node.Text("Item 2"))

	vOut, err := node.RenderToString(ctx, vstack)
	if err != nil {
		t.Fatalf("vstack error: %v", err)
	}
	if !strings.Contains(vOut, "flex-col") || !strings.Contains(vOut, "gap-4") || !strings.Contains(vOut, "items-center") {
		t.Errorf("unexpected vstack output: %s", vOut)
	}

	hstack := node.HStack(node.StackProps{
		Gap:     "gap-2",
		Justify: node.JustifyBetween,
	}, node.Text("Left"), node.Text("Right"))

	hOut, err := node.RenderToString(ctx, hstack)
	if err != nil {
		t.Fatalf("hstack error: %v", err)
	}
	if !strings.Contains(hOut, "flex-row") || !strings.Contains(hOut, "justify-between") {
		t.Errorf("unexpected hstack output: %s", hOut)
	}

	// Grid
	grid := node.Grid(node.GridProps{
		Columns: 3,
		Gap:     "gap-6",
	}, node.Text("Col 1"), node.Text("Col 2"), node.Text("Col 3"))

	gOut, err := node.RenderToString(ctx, grid)
	if err != nil {
		t.Fatalf("grid error: %v", err)
	}
	if !strings.Contains(gOut, "grid-cols-3") || !strings.Contains(gOut, "gap-6") {
		t.Errorf("unexpected grid output: %s", gOut)
	}

	// Section
	sec := node.Section(node.SectionProps{
		Title:    "Features",
		Subtitle: "Everything you need",
	}, node.Text("Section Body"))

	sOut, err := node.RenderToString(ctx, sec)
	if err != nil {
		t.Fatalf("section error: %v", err)
	}
	if !strings.Contains(sOut, "<section") || !strings.Contains(sOut, "Features") || !strings.Contains(sOut, "Everything you need") {
		t.Errorf("unexpected section output: %s", sOut)
	}

	// Box
	box := node.Box(node.BoxProps{
		Border:     true,
		Rounded:    true,
		Background: "bg-white dark:bg-black",
	}, node.Text("Box Content"))

	bOut, err := node.RenderToString(ctx, box)
	if err != nil {
		t.Fatalf("box error: %v", err)
	}
	if !strings.Contains(bOut, "border-2") || !strings.Contains(bOut, "rounded-lg") {
		t.Errorf("unexpected box output: %s", bOut)
	}
}

func TestPrimitiveComponents(t *testing.T) {
	ctx := context.Background()

	// Button
	btn := node.Button(node.ButtonProps{
		Variant: node.VariantPrimary,
		Size:    node.SizeLarge,
		Type:    "submit",
	}, node.Text("Sign In"))

	bOut, err := node.RenderToString(ctx, btn)
	if err != nil {
		t.Fatalf("button error: %v", err)
	}
	if !strings.Contains(bOut, "kiw-btn-primary") || !strings.Contains(bOut, `type="submit"`) || !strings.Contains(bOut, "Sign In") {
		t.Errorf("unexpected button output: %s", bOut)
	}

	// Button as Link
	linkBtn := node.Button(node.ButtonProps{
		Variant: node.VariantOutline,
		URL:     "https://krewire.com/docs",
	}, node.Text("Documentation"))

	lOut, err := node.RenderToString(ctx, linkBtn)
	if err != nil {
		t.Fatalf("link button error: %v", err)
	}
	if !strings.Contains(lOut, "<a") || !strings.Contains(lOut, `href="https://krewire.com/docs"`) {
		t.Errorf("unexpected link button output: %s", lOut)
	}

	// Input
	inp := node.Input(node.InputProps{
		Type:        node.InputEmail,
		Name:        "email",
		Placeholder: "user@example.com",
		Required:    true,
	})

	iOut, err := node.RenderToString(ctx, inp)
	if err != nil {
		t.Fatalf("input error: %v", err)
	}
	if !strings.Contains(iOut, `type="email"`) || !strings.Contains(iOut, `name="email"`) || !strings.Contains(iOut, "required") {
		t.Errorf("unexpected input output: %s", iOut)
	}

	// Heading
	h := node.Heading(node.HeadingProps{Level: node.Heading1}, node.Text("Welcome"))
	hOut, err := node.RenderToString(ctx, h)
	if err != nil {
		t.Fatalf("heading error: %v", err)
	}
	if !strings.Contains(hOut, "<h1>Welcome</h1>") && !strings.Contains(hOut, "<h1") {
		t.Errorf("unexpected heading output: %s", hOut)
	}

	// Badge
	badge := node.Badge(node.BadgeProps{Variant: node.VariantSuccess}, node.Text("Active"))
	bgOut, err := node.RenderToString(ctx, badge)
	if err != nil {
		t.Fatalf("badge error: %v", err)
	}
	if !strings.Contains(bgOut, "kiw-badge-success") || !strings.Contains(bgOut, "Active") {
		t.Errorf("unexpected badge output: %s", bgOut)
	}

	// Alert
	alert := node.Alert(node.AlertProps{
		Variant: node.VariantError,
		Title:   "Validation Failed",
	}, node.Text("Please correct the errors below."))
	aOut, err := node.RenderToString(ctx, alert)
	if err != nil {
		t.Fatalf("alert error: %v", err)
	}
	if !strings.Contains(aOut, "kiw-alert-error") || !strings.Contains(aOut, `role="alert"`) || !strings.Contains(aOut, "Validation Failed") {
		t.Errorf("unexpected alert output: %s", aOut)
	}

	// Card with slots
	card := node.Card(node.CardProps{
		Title:    "Account Settings",
		Subtitle: "Manage your profile",
		Slots: node.SlotMap{
			"footer": node.Button(node.ButtonProps{Variant: node.VariantSecondary}, node.Text("Save")),
		},
	}, node.Text("Card Body Here"))

	cdOut, err := node.RenderToString(ctx, card)
	if err != nil {
		t.Fatalf("card error: %v", err)
	}
	if !strings.Contains(cdOut, "Account Settings") || !strings.Contains(cdOut, "Card Body Here") || !strings.Contains(cdOut, "Save") {
		t.Errorf("unexpected card output: %s", cdOut)
	}

	// Divider
	div := node.Divider(node.DividerProps{Label: "OR"})
	dOut, err := node.RenderToString(ctx, div)
	if err != nil {
		t.Fatalf("divider error: %v", err)
	}
	if !strings.Contains(dOut, "OR") || !strings.Contains(dOut, "kiw-divider-with-label") {
		t.Errorf("unexpected divider output: %s", dOut)
	}

	// Modal
	modal := node.Modal(node.ModalProps{
		ID:    "confirm-dialog",
		Title: "Confirm Action",
		Open:  true,
	}, node.Text("Are you sure?"))
	mOut, err := node.RenderToString(ctx, modal)
	if err != nil {
		t.Fatalf("modal error: %v", err)
	}
	if !strings.Contains(mOut, "<dialog") || !strings.Contains(mOut, `id="confirm-dialog"`) || !strings.Contains(mOut, "open") {
		t.Errorf("unexpected modal output: %s", mOut)
	}

	// Table
	tbl := node.Table(node.TableProps{
		Headers: []string{"ID", "Name", "Status"},
		Rows: [][]node.Node{
			{node.Text("1"), node.Text("Alice"), node.Badge(node.BadgeProps{Variant: node.VariantSuccess}, node.Text("Active"))},
			{node.Text("2"), node.Text("Bob"), node.Badge(node.BadgeProps{Variant: node.VariantWarning}, node.Text("Pending"))},
		},
		Striped: true,
	})
	tOut, err := node.RenderToString(ctx, tbl)
	if err != nil {
		t.Fatalf("table error: %v", err)
	}
	if !strings.Contains(tOut, "<table") || !strings.Contains(tOut, "Alice") || !strings.Contains(tOut, "Bob") {
		t.Errorf("unexpected table output: %s", tOut)
	}
}

// Composite component definition test
func UserProfileCard(name, email, role string) node.Node {
	return node.Card(node.CardProps{
		Title: name,
		Slots: node.SlotMap{
			"footer": node.HStack(node.StackProps{Justify: node.JustifyEnd},
				node.Button(node.ButtonProps{Variant: node.VariantOutline, Size: node.SizeSmall}, node.Text("Edit Profile")),
			),
		},
	},
		node.VStack(node.StackProps{Gap: "gap-2"},
			node.Paragraph(node.TextProps{Muted: true}, node.Text(email)),
			node.Badge(node.BadgeProps{Variant: node.VariantPrimary}, node.Text(role)),
		),
	)
}

func TestCompositeComponent(t *testing.T) {
	ctx := context.Background()
	userCard := UserProfileCard("Alice Doe", "alice@krewire.com", "Admin")

	out, err := node.RenderToString(ctx, userCard)
	if err != nil {
		t.Fatalf("composite component render failed: %v", err)
	}

	for _, expected := range []string{"Alice Doe", "alice@krewire.com", "Admin", "Edit Profile"} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in composite output, got %s", expected, out)
		}
	}
}

func TestKiwAdapter(t *testing.T) {
	ctx := context.Background()

	// 1. Unregistered .kiw component renders clean diagnostic placeholder
	unreg := node.Kiw("CustomWidget", map[string]any{"propA": "val1"}, node.Text("Child Content"))
	out, err := node.RenderToString(ctx, unreg)
	if err != nil {
		t.Fatalf("unregistered kiw error: %v", err)
	}
	if !strings.Contains(out, `data-kiw-component="CustomWidget"`) || !strings.Contains(out, "Child Content") {
		t.Errorf("unexpected unresolved kiw output: %s", out)
	}

	// 2. Registered .kiw component executes registered handler
	node.RegisterKiwComponent("TestCounter", func(ctx context.Context, props map[string]any, children []node.Node, w io.Writer) error {
		start := props["start"]
		_, err := fmt.Fprintf(w, `<div class="kiw-counter" data-start="%v">`, start)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, `</div>`)
		return err
	})

	reg := node.Kiw("TestCounter", map[string]any{"start": 10}, node.Text("Click me"))
	regOut, err := node.RenderToString(ctx, reg)
	if err != nil {
		t.Fatalf("registered kiw error: %v", err)
	}
	if !strings.Contains(regOut, `<div class="kiw-counter" data-start="10">Click me</div>`) {
		t.Errorf("unexpected registered kiw output: %s", regOut)
	}
}

func TestWidgetAdapter(t *testing.T) {
	ctx := context.Background()

	// Existing legacy widget
	btnWidget := widget.NewButton("Legacy Button").Primary()
	adapted := node.FromWidget(btnWidget)

	out, err := node.RenderToString(ctx, adapted)
	if err != nil {
		t.Fatalf("from widget error: %v", err)
	}
	if !strings.Contains(out, "Legacy Button") {
		t.Errorf("expected 'Legacy Button' in adapted output, got %s", out)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	tree := node.Container(node.ContainerProps{}, node.Text("Should not render"))
	_, err := node.RenderToString(ctx, tree)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRenderToHTML(t *testing.T) {
	ctx := context.Background()
	tree := node.Element("span", nil, node.Text("Safe HTML"))
	html, err := node.RenderToHTML(ctx, tree)
	if err != nil {
		t.Fatalf("RenderToHTML error: %v", err)
	}
	if html != "<span>Safe HTML</span>" {
		t.Errorf("expected <span>Safe HTML</span>, got %q", html)
	}
}
