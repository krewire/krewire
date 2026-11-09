package ssg

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kiwdsl "github.com/krewire/krewire/pkg/kiw"
	"github.com/krewire/krewire/pkg/ui"
)

func writeSite(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const baseLayout = `<!doctype html><html><body>{{.Content}}</body></html>`

// A conditional inside a component attribute used to fail with
// "template: page:/:body:3: unexpected <if> in operand". The build error now
// names the .kiw file, line and column, with a hint.
func TestBuild_ErrorIsSourceMapped(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"layouts/Base.kiw": baseLayout,
		"pages/index.kiw":  "---\nlayout: Base\n---\n<div>\n  <Button\n    ariaLabel=\"{#if on}On{:else}Off{/if}\" />\n</div>\n",
	})
	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = site.Build(t.TempDir())
	if err == nil {
		t.Fatal("expected build error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "pages/index.kiw:6:16: {#if on} inside an attribute of <Button>") || !strings.Contains(msg, "hint:") {
		t.Errorf("error not source-mapped: %s", msg)
	}
}

func TestBuild_ExecErrorIsSourceMapped(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"layouts/Base.kiw":   baseLayout,
		"components/Row.kiw": "<ul>\n{#each items as item}\n  <li>{item.label.text}</li>\n{/each}\n</ul>",
		"pages/index.kiw":    "---\nlayout: Base\nItems: [a]\n---\n<Row items={items} />\n",
	})
	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = site.Build(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "components/Row.kiw:3:8:") {
		t.Errorf("error not source-mapped: %v", err)
	}
}

// kui Table.kiw nests {#each row as cell} inside {#each rows as row}.
func TestBuild_KuiTableNestedRows(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"layouts/Base.kiw": baseLayout,
		"pages/index.kiw":  "---\nlayout: Base\nHeaders: [Name, Plan]\nRows: [[Ada, Pro], [Linus, Free]]\n---\n<Table headers={headers} rows={rows} />\n",
	})
	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "index.html"))
	for _, want := range []string{"<td>Ada</td>", "<td>Free</td>", "<th>Plan</th>"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s in %s", want, b)
		}
	}
}

func TestKuiComponentsLintClean(t *testing.T) {
	files, _ := fs.Glob(ui.ComponentsFS, "*.kiw")
	for _, f := range files {
		src, _ := ui.ComponentsFS.ReadFile(f)
		for _, d := range kiwdsl.Lint(f, string(src)) {
			t.Errorf("%v", &d)
		}
	}
}

func TestPropAliasesInDict(t *testing.T) {
	m, err := dict("aria-label", "x", "Variant", "a", "variant", "b", "id", "7")
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{"ariaLabel": "x", "AriaLabel": "x", "Variant": "a", "variant": "b", "ID": "7"} {
		if m[k] != want {
			t.Errorf("m[%q] = %v, want %v", k, m[k], want)
		}
	}
}
