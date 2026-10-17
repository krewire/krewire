package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHTML_Valid(t *testing.T) {
	html := `<nav><a class="link" href="/">a</a><a class="link" href="/b">b</a></nav><div id="main">Hello Docs</div>`
	HTML(t, html).Has("nav a", 2).Has("a.link", 2).Has("#main", 1).ContainsText("Hello").HasText("div", "Hello Docs").Attr("a", "href", "/")
}

func TestSnapshot_Valid(t *testing.T) {
	name := "ui_snapshot_test"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	Snapshot(t, name, `<div>  hello  </div><link href="assets/app.abc123.css">`)
	t.Setenv("UPDATE_GOLDEN", "")
	Snapshot(t, name, `<div>  hello  </div><link href="assets/app.abc123.css">`)
	_ = os.Remove(path)
}

func TestThemeSnapshot_Valid(t *testing.T) {
	name := "theme_snapshot_test"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	ThemeSnapshot(t, name, `<html data-theme="dark"><style>:root{--primary:#ff6b00}</style></html>`)
	t.Setenv("UPDATE_GOLDEN", "")
	ThemeSnapshot(t, name, `<html data-theme="dark"><style>:root{--primary:#ff6b00}</style></html>`)
	_ = os.Remove(path)
}
