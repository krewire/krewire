package kiw

import (
	"errors"
	"html/template"
	"strings"
	"testing"
)

// stubFuncs mirrors the helper names a renderer registers (pkg/web/ssg).
var stubFuncs = map[string]any{
	"component": func(string, ...any) (template.HTML, error) { return "", nil },
	"dict":      func(...any) (map[string]any, error) { return nil, nil },
	"class":     func(...any) string { return "" },
}

const sample = `---
title: Demo
---
<style>
.card { color: red; }
</style>
<Card
  title="x"
/>
<p>{title}</p>
{{ end }}
`

func TestAlignedBodyKeepsLines(t *testing.T) {
	got := AlignedBody(sample)
	if strings.Count(got, "\n") != strings.Count(sample, "\n") {
		t.Fatalf("aligned body has %d lines, source %d:\n%s", strings.Count(got, "\n"), strings.Count(sample, "\n"), got)
	}
	if line := strings.Split(got, "\n")[9]; line != "<p>{{.Title}}</p>" {
		t.Errorf("line 10 = %q", line)
	}
}

func TestExplainParseError(t *testing.T) {
	mod, _ := ParseKiw(sample)
	_, perr := template.New("Demo").Funcs(stubFuncs).Parse(mod.Body)
	if perr == nil {
		t.Fatal("expected parse error")
	}
	err := Explain("components/Demo.kiw", "Demo", sample, stubFuncs, perr)
	var d *Diagnostic
	if !errors.As(err, &d) {
		t.Fatalf("Explain = %v, want *Diagnostic", err)
	}
	if d.Line != 11 || d.Col != 1 {
		t.Errorf("position = %d:%d, want 11:1 (%v)", d.Line, d.Col, d)
	}
	if !strings.HasPrefix(d.Error(), "components/Demo.kiw:11:1: unexpected {{end}}") {
		t.Errorf("message = %q", d.Error())
	}
	if !errors.Is(err, perr) {
		t.Error("diagnostic must wrap the template error")
	}
}

func TestExplainExecError(t *testing.T) {
	src := "<ul>\n{#each rows as row}\n  <li>\n    {row.name.first}\n  </li>\n{/each}\n</ul>"
	mod, _ := ParseKiw(src)
	tpl := template.Must(template.New("List").Funcs(stubFuncs).Parse(mod.Body))
	xerr := tpl.Execute(&strings.Builder{}, map[string]any{"Rows": []string{"a"}})
	if xerr == nil {
		t.Fatal("expected exec error")
	}
	var d *Diagnostic
	if !errors.As(Explain("List.kiw", "List", src, stubFuncs, xerr), &d) {
		t.Fatalf("not explained: %v", xerr)
	}
	if d.Line != 4 || d.Col != 6 {
		t.Errorf("position = %d:%d, want 4:6 (%v)", d.Line, d.Col, d)
	}
	if !strings.Contains(d.Hint, "prop name") {
		t.Errorf("hint = %q", d.Hint)
	}
}

func TestExplainUndefinedDict(t *testing.T) {
	src := "<div>\n  <Badge label=\"x\" />\n</div>"
	mod, _ := ParseKiw(src)
	_, perr := template.New("Box").Parse(mod.Body)
	var d *Diagnostic
	if !errors.As(Explain("Box.kiw", "Box", src, nil, perr), &d) {
		t.Fatalf("not explained: %v", perr)
	}
	if d.Line != 2 || !strings.Contains(d.Hint, "kiw build") {
		t.Errorf("got %v", d)
	}
}

func TestExplainIgnoresOtherTemplates(t *testing.T) {
	err := errors.New(`template: Other:3: unexpected "}"`)
	if got := Explain("x.kiw", "Mine", "<p></p>", nil, err); got != err {
		t.Errorf("Explain changed a foreign error: %v", got)
	}
}

func TestLintControlBlockInComponentAttribute(t *testing.T) {
	src := "<div>\n  <Toggle\n    AriaLabel=\"{#if on}On{:else}Off{/if}\" />\n</div>"
	diags := Lint("Toggle.kiw", src)
	if len(diags) != 1 {
		t.Fatalf("diags = %v", diags)
	}
	d := diags[0]
	if d.Line != 3 || d.Col != 16 || !strings.Contains(d.Message, "<Toggle>") || d.Hint == "" {
		t.Errorf("got %+v", d)
	}
	// The same error surfaces through Explain when the template fails to parse.
	mod, _ := ParseKiw(src)
	_, perr := template.New("T").Funcs(stubFuncs).Parse(mod.Body)
	if perr == nil {
		t.Fatal("expected parse error")
	}
	var got *Diagnostic
	if !errors.As(Explain("Toggle.kiw", "T", src, stubFuncs, perr), &got) || got.Line != 3 {
		t.Errorf("Explain = %v", got)
	}
}

func TestLintBlocks(t *testing.T) {
	cases := map[string]string{
		"{#if a}\n<p>x</p>":                 "1:1: {#if} is never closed",
		"<p>\n{/if}":                        "2:1: {/if} without a matching {#if}",
		"{#each xs as x}\n{#if x}\n{/each}": "3:1: {/each} closes {#if} opened at line 2",
		"{:else}":                           "1:1: {:else} without an enclosing {#if}",
		"<style>{#if}</style><p>ok</p>":     "",
		"{#each xs as x}{:else}{/each}":     "",
	}
	for src, want := range cases {
		diags := Lint("", src)
		got := ""
		if len(diags) > 0 {
			got = diags[0].Error()
		}
		if !strings.HasPrefix(got, want) || (want == "" && got != "") {
			t.Errorf("Lint(%q) = %q, want prefix %q", src, got, want)
		}
	}
}

func TestLintFrontmatter(t *testing.T) {
	diags := Lint("p.kiw", "---\ntitle: ok\nfoo: bar\n  baz: 1\n---\n<p></p>")
	if len(diags) != 1 || diags[0].Line != 4 || !strings.Contains(diags[0].Message, "invalid frontmatter") {
		t.Errorf("diags = %v", diags)
	}
}
