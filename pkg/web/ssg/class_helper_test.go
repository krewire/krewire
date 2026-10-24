package ssg

import (
	"bytes"
	"html/template"
	"reflect"
	"strings"
	"testing"
)

// FRK-DSL-501: class "base" prefix value default … trailing… composes base + conditional modifiers + trailing tokens.
func TestClassHelper_FRK_DSL_501_Composition(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want string
	}{
		{
			name: "base with explicit prefix triple and trailing",
			args: []any{"btn", "btn-", "secondary", "primary", "custom-class"},
			want: "btn btn-secondary custom-class",
		},
		{
			name: "base with explicit prefix default and multiple trailing",
			args: []any{"card", "card-", "", "default", "shadow-lg", "rounded"},
			want: "card card-default shadow-lg rounded",
		},
		{
			name: "base with implicit pairs",
			args: []any{"btn", "outline", "primary", "lg", "md", "shadow-sm"},
			want: "btn btn-outline btn-lg shadow-sm",
		},
		{
			name: "empty base with explicit prefix",
			args: []any{"", "text-", "red", "black"},
			want: "text-red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassHelper(tt.args...)
			if got != tt.want {
				t.Errorf("ClassHelper() = %q, want %q", got, tt.want)
			}
		})
	}
}

// FRK-DSL-502: A group emits prefix+value when value is non-empty, else prefix+default; empty groups contribute nothing.
func TestClassHelper_FRK_DSL_502_ModifierRules(t *testing.T) {
	// Value provided -> prefix+value
	if got := ClassHelper("btn", "secondary", "primary"); got != "btn btn-secondary" {
		t.Errorf("value provided: got %q, want %q", got, "btn btn-secondary")
	}

	// Value empty -> prefix+default
	if got := ClassHelper("btn", "", "primary"); got != "btn btn-primary" {
		t.Errorf("value empty: got %q, want %q", got, "btn btn-primary")
	}

	// Value nil -> prefix+default
	if got := ClassHelper("btn", nil, "primary"); got != "btn btn-primary" {
		t.Errorf("value nil: got %q, want %q", got, "btn btn-primary")
	}

	// Empty group (both value and default empty) contributes nothing
	if got := ClassHelper("btn", "", ""); got != "btn" {
		t.Errorf("empty group: got %q, want %q", got, "btn")
	}

	// Multiple empty groups contribute nothing
	if got := ClassHelper("btn", "", "", nil, false); got != "btn" {
		t.Errorf("multiple empty groups: got %q, want %q", got, "btn")
	}
}

// FRK-DSL-503: Output has no leading, trailing, or doubled spaces (collapse).
func TestClassHelper_FRK_DSL_503_WhitespaceCollapse(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want string
	}{
		{
			name: "extra spaces in base, modifier, and trailing",
			args: []any{"  btn  ", "  outline  ", "primary", "   extra-1    extra-2   "},
			want: "btn btn-outline extra-1 extra-2",
		},
		{
			name: "no tokens",
			args: []any{"", "", ""},
			want: "",
		},
		{
			name: "only trailing with whitespace",
			args: []any{"", "   single-token   "},
			want: "single-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassHelper(tt.args...)
			if got != tt.want {
				t.Errorf("ClassHelper() = %q, want %q", got, tt.want)
			}
			if strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") || strings.Contains(got, "  ") {
				t.Errorf("whitespace not collapsed: %q", got)
			}
		})
	}
}

// FRK-DSL-504: The helper returns a plain string, NOT template.HTML.
func TestClassHelper_FRK_DSL_504_ReturnsPlainString(t *testing.T) {
	res := ClassHelper("btn", "primary", "secondary")
	val := reflect.ValueOf(res)
	if val.Kind() != reflect.String {
		t.Fatalf("expected string kind, got %v", val.Kind())
	}
	if reflect.TypeOf(res) == reflect.TypeOf(template.HTML("")) {
		t.Fatalf("ClassHelper must return plain string, not template.HTML (ADR-K4M2Q violation)")
	}
}

// FRK-DSL-505: Escaping regression test — hostile value renders escaped (no second escaping model).
func TestClassHelper_FRK_DSL_505_EscapingRegression(t *testing.T) {
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"class": ClassHelper,
	}).Parse(`<a class="{{class "btn" .Variant "primary" .Class}}">link</a>`)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	data := map[string]any{
		"Variant": `"><script>alert(1)</script>`,
		"Class":   `evil" onclick="alert(2)`,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	rendered := buf.String()

	// Ensure raw script tag is not injected into the markup
	if strings.Contains(rendered, "<script>") || strings.Contains(rendered, `onclick="alert(2)`) {
		t.Fatalf("XSS regression: raw unescaped input present in rendered markup:\n%s", rendered)
	}

	// Ensure attribute quotes are escaped by html/template
	if !strings.Contains(rendered, "&#34;") {
		t.Errorf("expected html/template attribute escaping in rendered markup:\n%s", rendered)
	}
}

// FRK-DSL-506: Equivalence test — {{class "btn" .Variant "primary" .Class}} renders
// identically to hand-written {{if .Variant}}btn-{{.Variant}}{{else}}btn-primary{{end}} {{.Class}}.
func TestClassHelper_FRK_DSL_506_Equivalence(t *testing.T) {
	tHand, err := template.New("hand").Parse(`<a class="btn {{if .Variant}}btn-{{.Variant}}{{else}}btn-primary{{end}}{{if .Class}} {{.Class}}{{end}}"></a>`)
	if err != nil {
		t.Fatalf("parse hand: %v", err)
	}

	tHelper, err := template.New("helper").Funcs(template.FuncMap{
		"class": ClassHelper,
	}).Parse(`<a class="{{class "btn" .Variant "primary" .Class}}"></a>`)
	if err != nil {
		t.Fatalf("parse helper: %v", err)
	}

	testCases := []map[string]any{
		{"Variant": "", "Class": ""},
		{"Variant": "secondary", "Class": ""},
		{"Variant": "", "Class": "extra-class"},
		{"Variant": "accent", "Class": "my-btn pop"},
	}

	for _, tc := range testCases {
		var bufHand, bufHelper bytes.Buffer
		if err := tHand.Execute(&bufHand, tc); err != nil {
			t.Fatalf("execute hand: %v", err)
		}
		if err := tHelper.Execute(&bufHelper, tc); err != nil {
			t.Fatalf("execute helper: %v", err)
		}

		if bufHand.String() != bufHelper.String() {
			t.Errorf("equivalence failure for tc %+v:\nhand:   %q\nhelper: %q", tc, bufHand.String(), bufHelper.String())
		}
	}
}

// FRK-DSL-507: Refactor Button, Card, Alert, Progress, Badge, Link to use class helper.
// Asserts that all six components render byte-accurate classes under both default and custom props.
func TestClassHelper_FRK_DSL_507_RefactoredComponents(t *testing.T) {
	site, err := LoadFromDir(atomicProbeSite(t))
	if err != nil {
		t.Fatalf("load probe site: %v", err)
	}

	tests := []struct {
		name      string
		component string
		props     map[string]any
		wantClass string
	}{
		// Button
		{"Button defaults", "Button", map[string]any{}, "btn btn-primary btn-md"},
		{"Button custom", "Button", map[string]any{"Variant": "secondary", "Size": "lg", "Class": "extra-btn"}, "btn btn-secondary btn-lg extra-btn"},
		// Card
		{"Card defaults", "Card", map[string]any{}, "card card-default"},
		{"Card custom", "Card", map[string]any{"Variant": "featured", "Class": "extra-card"}, "card card-featured extra-card"},
		// Alert
		{"Alert defaults", "Alert", map[string]any{}, "alert alert-info"},
		{"Alert custom", "Alert", map[string]any{"Variant": "success", "Class": "extra-alert"}, "alert alert-success extra-alert"},
		// Progress
		{"Progress defaults", "Progress", map[string]any{}, "kiw-progress kiw-progress-primary kiw-progress-md"},
		{"Progress custom", "Progress", map[string]any{"Variant": "accent", "Size": "sm", "Class": "extra-prog"}, "kiw-progress kiw-progress-accent kiw-progress-sm extra-prog"},
		// Badge
		{"Badge defaults", "Badge", map[string]any{}, "badge badge-neutral badge-md"},
		{"Badge custom", "Badge", map[string]any{"Variant": "error", "Size": "sm", "Class": "extra-badge"}, "badge badge-error badge-sm extra-badge"},
		// Link
		{"Link defaults", "Link", map[string]any{}, "kiw-link kiw-link-default"},
		{"Link custom", "Link", map[string]any{"Variant": "subtle", "Class": "extra-link"}, "kiw-link kiw-link-subtle extra-link"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := site.renderComponent(tt.component, tt.props)
			if err != nil {
				t.Fatalf("%s render error: %v", tt.component, err)
			}
			expectedAttr := `class="` + tt.wantClass + `"`
			if !strings.Contains(string(rendered), expectedAttr) {
				t.Errorf("%s missing expected class attribute %q in output:\n%s", tt.component, expectedAttr, rendered)
			}
		})
	}
}
