package kiw

import (
	"html/template"
	"strings"
	"testing"
)

// Nested {#each} blocks used to pair the outer header with the inner
// {/each}, so the inner loop ranged over .Row and failed with
// "can't evaluate field Row in type []string" (ui Table.kiw).
func TestDesugar_NestedEach(t *testing.T) {
	src := `<table>{#each rows as row}<tr>{#each row as cell}<td>{cell}</td>{/each}</tr>{/each}</table>`
	got, _ := DesugarTemplate(src)
	want := `<table>{{range $row := .Rows}}<tr>{{range $cell := $row}}<td>{{$cell}}</td>{{end}}</tr>{{end}}</table>`
	if got != want {
		t.Fatalf("desugar:\n got %s\nwant %s", got, want)
	}
	tpl := template.Must(template.New("t").Parse(got))
	var b strings.Builder
	if err := tpl.Execute(&b, map[string]any{"Rows": [][]string{{"a", "b"}, {"c"}}}); err != nil {
		t.Fatal(err)
	}
	if b.String() != `<table><tr><td>a</td><td>b</td></tr><tr><td>c</td></tr></table>` {
		t.Errorf("render = %s", b.String())
	}
}

func TestDesugar_NestedEachOuterScope(t *testing.T) {
	src := `{#each groups as g, i}{#each g.items as item}{item.label}{label}{i}{/each}{/each}`
	got, _ := DesugarTemplate(src)
	want := `{{range $i, $g := .Groups}}{{range $item := $g.Items}}{{$item.Label}}{{$.Label}}{{$i}}{{end}}{{end}}`
	if got != want {
		t.Fatalf("desugar:\n got %s\nwant %s", got, want)
	}
}

func TestDesugar_UnclosedEachLeftAlone(t *testing.T) {
	src := `{#each items as item}<li>{item}</li>`
	got, _ := DesugarTemplate(src)
	if !strings.Contains(got, "{#each items as item}") {
		t.Errorf("unclosed block should be left for Lint to report, got %s", got)
	}
}
