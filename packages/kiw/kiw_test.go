// Tests for the .kiw module parser.
// Scope: Unit libs/kiw.
// Moved here from kiw/dsl: the parser is a leaf concern, and framework/web/ssg
// must be able to parse a .kiw file without importing the devtool.
package kiw

import (
	"strings"
	"testing"
)

func TestParseKiw_FrontmatterAndTemplate(t *testing.T) {
	src := "---\ntitle: Landing\nlayout: Base\n---\n<h1>{{.Title}}</h1>\n<style>h1{color:red}</style>\n<script>console.log(1)</script>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Frontmatter["title"] != "Landing" {
		t.Errorf("frontmatter title = %v want Landing", mod.Frontmatter["title"])
	}
	if mod.Body != "<h1>{{.Title}}</h1>" {
		t.Errorf("body = %q want <h1>{{.Title}}</h1>", mod.Body)
	}
	if len(mod.Styles) != 1 || mod.Styles[0] != "h1{color:red}" {
		t.Errorf("styles = %v", mod.Styles)
	}
	if len(mod.Scripts) != 1 || mod.Scripts[0] != "console.log(1)" {
		t.Errorf("scripts = %v", mod.Scripts)
	}
}

func TestParseKiw_NoFrontmatter(t *testing.T) {
	src := "<p>hello</p>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Body != "<p>hello</p>" {
		t.Errorf("body = %q", mod.Body)
	}
}

func TestParseKiw_JSParseable(t *testing.T) {
	src := "---\ntitle: Landing\n---\n<div>{{title}}</div>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Frontmatter["title"] != "Landing" {
		t.Errorf("frontmatter")
	}
	if mod.Body != "<div>{{title}}</div>" {
		t.Errorf("body")
	}
}

func TestGoClientTier_NoNeedJS(t *testing.T) {
	// Go WASM as primary client, no need to write JS
	src := `<div>hi</div><script lang="go" hydrate="load">var c = $props.initial</script><script lang="ts" hydrate="idle">let x=1</script><script lang="go" server>func Load(){}</script>`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(mod.ScriptBlocks) != 3 {
		t.Fatalf("scriptBlocks = %d want 3", len(mod.ScriptBlocks))
	}
	if mod.ScriptBlocks[0].Lang != "go" || mod.ScriptBlocks[0].Hydrate != "load" || mod.ScriptBlocks[0].Server {
		t.Errorf("go client block = %+v", mod.ScriptBlocks[0])
	}
	if mod.ScriptBlocks[1].Lang != "ts" || mod.ScriptBlocks[1].Hydrate != "idle" {
		t.Errorf("ts client block = %+v", mod.ScriptBlocks[1])
	}
	if !mod.ScriptBlocks[2].Server || mod.ScriptBlocks[2].Lang != "go" {
		t.Errorf("go server block = %+v", mod.ScriptBlocks[2])
	}
	// default <script> without lang → js load
	mod2, _ := ParseKiw(`<script>console.log(1)</script>`)
	if mod2.ScriptBlocks[0].Lang != "js" || mod2.ScriptBlocks[0].Hydrate != "load" {
		t.Errorf("default script block = %+v", mod2.ScriptBlocks[0])
	}
}

func TestStyleScoped(t *testing.T) {
	src := `<style scoped>.btn{color:red}</style><style>h1{}</style>`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(mod.StyleBlocks) != 2 {
		t.Fatalf("styleBlocks = %d", len(mod.StyleBlocks))
	}
	if !mod.StyleBlocks[0].Scoped {
		t.Errorf("first style should be scoped")
	}
	if mod.StyleBlocks[1].Scoped {
		t.Errorf("second style should not be scoped")
	}
}

func TestParseKiw_ComponentMustacheAndJSXEquivalence(t *testing.T) {
	// Case 1: Simple {{component "Navbar"}} vs <Navbar />
	srcMustache := `<div>{{component "Navbar"}}</div>`
	srcJSX := `<div><Navbar /></div>`

	mod1, err := ParseKiw(srcMustache)
	if err != nil {
		t.Fatalf("ParseKiw mustache error: %v", err)
	}
	mod2, err := ParseKiw(srcJSX)
	if err != nil {
		t.Fatalf("ParseKiw JSX error: %v", err)
	}

	// Both should yield the exact same Body for the template engine
	if mod1.Body != mod2.Body {
		t.Errorf("body mismatch: mustache=%q, jsx=%q", mod1.Body, mod2.Body)
	}
	if len(mod2.ComponentNames) != 1 || mod2.ComponentNames[0] != "Navbar" {
		t.Errorf("expected component 'Navbar', got %v", mod2.ComponentNames)
	}
	if len(mod1.ComponentNames) != 1 || mod1.ComponentNames[0] != "Navbar" {
		t.Errorf("expected component 'Navbar', got %v", mod1.ComponentNames)
	}
}

func TestParseKiw_ComponentWithProps(t *testing.T) {
	src := `
<Navbar ShowSidebarToggle=true Version=.Version />
<ThemeSwitch />
<Button Variant="primary" Count=42 Active={true}>Save Changes</Button>
`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatalf("ParseKiw error: %v", err)
	}

	if len(mod.ComponentNames) != 3 {
		t.Fatalf("expected 3 components, got %d: %v", len(mod.ComponentNames), mod.ComponentNames)
	}

	// Verify desugared body contains proper Go template calls
	if !strings.Contains(mod.Body, `{{component "Navbar" (dict "ShowSidebarToggle" true "Version" .Version)}}`) {
		t.Errorf("expected desugared Navbar with dict props, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `{{component "ThemeSwitch"}}`) {
		t.Errorf("expected desugared ThemeSwitch, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `{{component "Button" (dict "Variant" "primary" "Count" 42 "Active" true "Body" "Save Changes")}}`) {
		t.Errorf("expected desugared Button with body, got:\n%s", mod.Body)
	}
}

func TestDesugarTemplate_Direct(t *testing.T) {
	out, err := DesugarTemplate(`<Card Title="My Card" />`)
	if err != nil {
		t.Fatal(err)
	}
	want := `{{component "Card" (dict "Title" "My Card")}}`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// A component nested inside another component's body must be desugared too.
//
// The body is embedded as a quoted string literal, so a child tag left as-is
// would be emitted into the page as literal text — the child component would
// silently never render. This is the regression guard for that.
//
// The child's quotes appear escaped (`{{component \"BreadcrumbItem\" ...}}`)
// because the desugared call is now a *string argument* inside the parent's
// dict. ssg re-executes such a body at render time; see ssg.renderFragment.
func TestDesugarTemplate_NestedComponentInBody(t *testing.T) {
	out, err := DesugarTemplate(`<Breadcrumb><BreadcrumbItem Label="Home" /></Breadcrumb>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `BreadcrumbItem`) {
		t.Errorf("nested component was not desugared, it would render as literal text:\n%s", out)
	}
	// The parent must carry the child as Body, not drop it.
	if !strings.Contains(out, `"Body"`) {
		t.Errorf("nested component is not passed through Body:\n%s", out)
	}
	// A child that was NOT desugared would still contain its raw tag.
	if strings.Contains(out, `<BreadcrumbItem`) {
		t.Errorf("child tag survived as raw markup:\n%s", out)
	}
}

// Nesting must recurse: a grandchild is as invisible as a child was.
func TestDesugarTemplate_DeeplyNestedComponentInBody(t *testing.T) {
	out, err := DesugarTemplate(`<Outer><Mid><Inner Label="x" /></Mid></Outer>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Outer", "Mid", "Inner"} {
		if !strings.Contains(out, name) {
			t.Errorf("missing %s in:\n%s", name, out)
		}
	}
	if strings.Contains(out, "<Inner") {
		t.Errorf("grandchild tag survived as raw markup:\n%s", out)
	}
}

func TestParseKiw_PreservesExternalScriptTags(t *testing.T) {
	src := `<div>content</div><script src="/assets/app.js" defer></script>`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mod.Body, `<script src="/assets/app.js" defer></script>`) {
		t.Errorf("expected external script to be preserved in body, got %q", mod.Body)
	}
	if len(mod.Scripts) != 0 {
		t.Errorf("expected 0 inline scripts extracted, got %d", len(mod.Scripts))
	}
}

func TestDesugar_Interpolation(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<h1>{title}</h1>", "<h1>{{.Title}}</h1>"},
		{"<p>Hello, {user.name}!</p>", "<p>Hello, {{.User.Name}}!</p>"},
		{"<div>{{.AlreadyDotted}}</div>", "<div>{{.AlreadyDotted}}</div>"},
		{"<code>{ \"key\": 1 }</code>", "<code>{ \"key\": 1 }</code>"},
	}

	for _, tc := range cases {
		out, err := DesugarTemplate(tc.in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.in, err)
		}
		if out != tc.want {
			t.Errorf("input %q => got %q, want %q", tc.in, out, tc.want)
		}
	}
}

func TestDesugar_Conditionals(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			`{#if role == 'admin'}<h1>Admin</h1>{/if}`,
			`{{if eq .Role "admin"}}<h1>Admin</h1>{{end}}`,
		},
		{
			`{#if count > 0}<p>Count</p>{:else}<p>Empty</p>{/if}`,
			`{{if gt .Count 0}}<p>Count</p>{{else}}<p>Empty</p>{{end}}`,
		},
		{
			`{#if role == 'admin' && active}<p>Active Admin</p>{/if}`,
			`{{if and (eq .Role "admin") .Active}}<p>Active Admin</p>{{end}}`,
		},
		{
			`{#if !isGuest}<p>User</p>{/if}`,
			`{{if not .IsGuest}}<p>User</p>{{end}}`,
		},
		{
			`{#if role == 'admin' || role == 'editor'}<p>Staff</p>{/if}`,
			`{{if or (eq .Role "admin") (eq .Role "editor")}}<p>Staff</p>{{end}}`,
		},
		{
			`{#if status == 'a'}<p>A</p>{:else if status == 'b'}<p>B</p>{:else}<p>C</p>{/if}`,
			`{{if eq .Status "a"}}<p>A</p>{{else if eq .Status "b"}}<p>B</p>{{else}}<p>C</p>{{end}}`,
		},
	}

	for _, tc := range cases {
		out, err := DesugarTemplate(tc.in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.in, err)
		}
		if out != tc.want {
			t.Errorf("input %q => got %q, want %q", tc.in, out, tc.want)
		}
	}
}

func TestDesugar_Iteration(t *testing.T) {
	in := `<ul>{#each items as item}<li>{item.title} by {author.name}</li>{/each}</ul>`
	want := `<ul>{{range $item := .Items}}<li>{{$item.Title}} by {{$.Author.Name}}</li>{{end}}</ul>`
	out, err := DesugarTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}

	inIndex := `{#each tags as tag, i}<span>{i}: {tag}</span>{/each}`
	wantIndex := `{{range $i, $tag := .Tags}}<span>{{$i}}: {{$tag}}</span>{{end}}`
	outIndex, err := DesugarTemplate(inIndex)
	if err != nil {
		t.Fatal(err)
	}
	if outIndex != wantIndex {
		t.Errorf("got %q, want %q", outIndex, wantIndex)
	}
}

func TestDesugar_ClassDirectives(t *testing.T) {
	in := `<button class="btn" class:btn-primary={variant == 'primary'} class:active={isActive}>Click</button>`
	want := `<button class="btn{{if eq .Variant "primary"}} btn-primary{{end}}{{if .IsActive}} active{{end}}">Click</button>`
	out, err := DesugarTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}

	inStandalone := `<div class:hidden={!show}>Content</div>`
	wantStandalone := `<div class="{{if not .Show}}hidden{{end}}">Content</div>`
	outStandalone, err := DesugarTemplate(inStandalone)
	if err != nil {
		t.Fatal(err)
	}
	if outStandalone != wantStandalone {
		t.Errorf("got %q, want %q", outStandalone, wantStandalone)
	}
}

func TestDesugar_PropPunning(t *testing.T) {
	in := `<Card {title} {author} count=42 />`
	out, err := DesugarTemplate(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `{{component "Card"`) {
		t.Fatalf("expected component call in %q", out)
	}
	if !strings.Contains(out, `"title" .title`) || !strings.Contains(out, `"author" .author`) || !strings.Contains(out, `"count" 42`) {
		t.Errorf("expected punned and explicit props, got:\n%s", out)
	}
}

func TestParseKiw_FullModernSFC(t *testing.T) {
	src := `---
title: Modern Dashboard
---
<div class="container" class:active={isActive}>
  <h1>{title}</h1>
  {#if user.role == 'admin'}
    <p>Welcome admin {user.name}!</p>
  {:else}
    <p>Welcome guest!</p>
  {/if}
  <ul>
    {#each tags as tag}
      <li>{tag}</li>
    {/each}
  </ul>
  <Card {title} variant="primary" />
</div>
<style scoped>
.container { padding: 1rem; }
</style>
<script>
console.log("Ready");
</script>`

	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}

	if mod.Frontmatter["title"] != "Modern Dashboard" {
		t.Errorf("unexpected frontmatter: %v", mod.Frontmatter)
	}
	if !strings.Contains(mod.Body, `{{.Title}}`) {
		t.Errorf("expected {title} to be desugared to {{.Title}}, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `{{if eq .User.Role "admin"}}`) {
		t.Errorf("expected {#if} to be desugared to {{if eq .User.Role \"admin\"}}, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `{{range $tag := .Tags}}`) {
		t.Errorf("expected {#each} to be desugared to range, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `class="container{{if .IsActive}} active{{end}}"`) {
		t.Errorf("expected class directive to be desugared, got:\n%s", mod.Body)
	}
	if !strings.Contains(mod.Body, `{{component "Card" (dict "title" .title "variant" "primary")}}`) {
		t.Errorf("expected Card to be desugared with punned title prop, got:\n%s", mod.Body)
	}
	if len(mod.Styles) != 1 {
		t.Errorf("expected 1 style block, got %d", len(mod.Styles))
	}
	if len(mod.Scripts) != 1 {
		t.Errorf("expected 1 script block, got %d", len(mod.Scripts))
	}
}

func TestDesugar_IslandHydrationDirectives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hydrate  string
		comp     string
	}{
		{
			name:     "client:load with props",
			input:    `<Counter client:load initial=0 step=1 />`,
			expected: `{{mount "Counter" "load" (dict "initial" 0 "step" 1)}}`,
			hydrate:  "load",
			comp:     "Counter",
		},
		{
			name:     "client:idle with string prop",
			input:    `<Search client:idle placeholder="Search docs..." />`,
			expected: `{{mount "Search" "idle" (dict "placeholder" "Search docs...")}}`,
			hydrate:  "idle",
			comp:     "Search",
		},
		{
			name:     "client:visible without props",
			input:    `<Comments client:visible />`,
			expected: `{{mount "Comments" "visible" (dict)}}`,
			hydrate:  "visible",
			comp:     "Comments",
		},
		{
			name:     "explicit hydrate attribute",
			input:    `<Analytics hydrate="idle" siteId="kw-100" />`,
			expected: `{{mount "Analytics" "idle" (dict "siteId" "kw-100")}}`,
			hydrate:  "idle",
			comp:     "Analytics",
		},
		{
			name:     "client:hydrate attribute",
			input:    `<Widget client:hydrate="visible" active />`,
			expected: `{{mount "Widget" "visible" (dict "active" true)}}`,
			hydrate:  "visible",
			comp:     "Widget",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := DesugarTemplate(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tc.expected {
				t.Errorf("got %q, want %q", res, tc.expected)
			}

			mod, err := ParseKiw(tc.input)
			if err != nil {
				t.Fatalf("ParseKiw error: %v", err)
			}
			found := false
			for _, c := range mod.Components {
				if c.Name == tc.comp && c.Hydrate == tc.hydrate {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected component %q with hydrate %q in parsed components: %+v", tc.comp, tc.hydrate, mod.Components)
			}
		})
	}
}
