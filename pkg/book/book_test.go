package book

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeContent creates a temporary content directory with the given
// filename-to-body mapping and returns its path. Nested names create
// subdirectories.
func writeContent(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadSubchapters(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-intro.md":              "# Intro\n",
		"02-tutorials/_index.md":   "# Tutorials\n",
		"02-tutorials/02-setup.md": "# Setup\n",
		"02-tutorials/01-hello.md": "# Hello\n",
		"03-outro.md":              "# Outro\n",
	})
	b, err := Load(dir, "B", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 3 {
		t.Fatalf("got %d chapters, want 3", len(b.Chapters))
	}
	tut := &b.Chapters[1]
	if tut.Title != "Tutorials" || len(tut.Subs) != 2 {
		t.Fatalf("tutorials = %+v, want title Tutorials with 2 subs", tut)
	}
	if tut.Subs[0].Title != "Hello" || tut.Subs[1].Title != "Setup" {
		t.Errorf("sub order = %q, %q; want Hello, Setup", tut.Subs[0].Title, tut.Subs[1].Title)
	}
	if got := tut.Subs[0].Path(); got != "/tutorials/hello" {
		t.Errorf("sub path = %q, want /tutorials/hello", got)
	}
	if tut.Subs[0].Parent != tut || tut.Subs[0].Number != 2 || tut.Subs[0].Sub != 1 {
		t.Errorf("sub lineage wrong: parent/set %+v", tut.Subs[0])
	}
	flat := b.flattened()
	if len(flat) != 5 {
		t.Fatalf("flattened = %d pages, want 5", len(flat))
	}
	if tut.Prev != flat[0] || tut.Next != tut.Subs[0] || tut.Subs[0].Next != tut.Subs[1] || tut.Subs[1].Next != flat[4] {
		t.Error("linear prev/next wiring across subchapters is wrong")
	}
}

func TestSubchapterAutoListAndSidebar(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-guide/01-a.md":  "# A\n",
		"01-guide/02-b.md":  "# B\n",
		"02-part/_index.md": "# Part\n",
		"02-part/01-one.md": "# One\n",
	})
	b, err := Load(dir, "B", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b.Chapters[0].Body), "class=\"sub-toc\"") {
		t.Error("directory without an index file must auto-list its subchapters")
	}
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B", BasePath: "/guide/"}); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(out, "guide", "a.html")
	html, err := os.ReadFile(sub)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="crumbs"`,
		`>Home</a>`,
		`href="/guide/guide"`,
		`toc-sub active`, // viewing a subchapter marks it active...
		`toc-sub`,        // ...its sibling shows as plain
		`href="/guide/guide/b"`,
		`class="pager"`,
	} {
		if !strings.Contains(string(html), want) {
			t.Errorf("subchapter page missing %q", want)
		}
	}
}

func TestLoadOrdersChapters(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"02-second.md": "# Second\n\nBody B.\n",
		"01-first.md":  "# First\n\nBody A.\n",
		"NOTES.md":     "# Not a chapter heading source, but a plain chapter\n",
	})
	b, err := Load(dir, "My Book", "Author")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 3 {
		t.Fatalf("got %d chapters, want 3", len(b.Chapters))
	}
	if b.Chapters[0].Title != "First" || b.Chapters[1].Title != "Second" {
		t.Errorf("chapters = %q, %q; want First, Second", b.Chapters[0].Title, b.Chapters[1].Title)
	}
	if got := b.Chapters[0].Path(); got != "/first" {
		t.Errorf("path = %q, want /first", got)
	}
	if got := b.Chapters[2].Path(); got != "/notes" {
		t.Errorf("un-numbered chapter path = %q, want /notes", got)
	}
	if b.Chapters[0].Next != &b.Chapters[1] || b.Chapters[1].Prev != &b.Chapters[0] {
		t.Error("prev/next navigation not wired correctly")
	}
	if !strings.Contains(string(b.Chapters[0].Body), "<p>Body A.</p>") {
		t.Errorf("body = %q, want rendered HTML", b.Chapters[0].Body)
	}
}

func TestLoadTitleFallbackToSlug(t *testing.T) {
	dir := writeContent(t, map[string]string{"intro.md": "no heading here\n"})
	b, err := Load(dir, "T", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 1 {
		t.Fatalf("got %d chapters, want 1", len(b.Chapters))
	}
	if b.Chapters[0].Title != "intro" {
		t.Errorf("title = %q, want intro", b.Chapters[0].Title)
	}
}

func TestBuildExportsDeterministicSite(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-chapter-one.md": "# Chapter One\n\nHello **world**.\n",
	})
	if _, err := Build(Config{Input: dir, Output: filepath.Join(dir, "..", "out"), Title: "Book", Author: "Reas"}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildSetsAffordDefaults(t *testing.T) {
	dir := writeContent(t, map[string]string{"index.md": "# Home\n"})
	created, err := Build(Config{Input: dir, Title: "B", Author: "A"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(".krewire") })
	if len(created) < 2 {
		t.Fatalf("created %d paths, want at least 2: %v", len(created), created)
	}
	for _, p := range created {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("created path missing: %s", p)
		}
	}

	index, err := os.ReadFile(".krewire/build/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), ">B<") {
		t.Errorf("index.html = %q, want book title", index)
	}
	css, err := os.ReadFile(".krewire/build/assets/krewire.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), "krewire") {
		t.Errorf("css = %q, want default stylesheet", css)
	}
}

func TestBuildDeterministic(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-a.md": "# A\n"})
	mk := func() string {
		out := filepath.Join(t.TempDir(), "site")
		if _, err := Build(Config{Input: dir, Output: out, Title: "T", Author: "A"}); err != nil {
			t.Fatal(err)
		}
		paths := []string{
			filepath.Join(out, "index.html"),
			filepath.Join(out, "a.html"),
			filepath.Join(out, "assets", "krewire.css"),
		}
		digest := ""
		for _, p := range paths {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			digest += string(b)
		}
		return digest
	}
	if mk() != mk() {
		t.Error("build output is not deterministic")
	}
}

func TestHandlerServesBookAndAssets(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-hello.md": "# Hello\n"})
	b, err := Load(dir, "T", "")
	if err != nil {
		t.Fatal(err)
	}
	h, err := b.Handler()
	if err != nil {
		t.Fatal(err)
	}
	tt := []struct {
		path string
		code int
	}{
		{"/", http.StatusOK},
		{"/hello", http.StatusOK},
		{"/assets/krewire.css", http.StatusOK},
		{"/missing", http.StatusNotFound},
	}
	for _, tc := range tt {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.code {
			t.Errorf("%s = %d, want %d", tc.path, rec.Code, tc.code)
		}
	}
}

func TestBuildWithBasePath(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-one.md": "# One\n\nSee the [next](/two) chapter.\n",
		"02-two.md": "# Two\n\nBack to [home](/).\n",
	})
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B", Author: "A", BasePath: "/guide/"}); err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/guide/assets/krewire.css"`, `href="/guide/one"`, `href="/guide/two"`} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html missing %s", want)
		}
	}

	chapter, err := os.ReadFile(filepath.Join(out, "one.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="/guide/assets/krewire.css"`,
		`href="/guide/two"`,
		`href="/guide/two"`, // in-body absolute link rewritten
		`href="/guide/"`,    // in-body "/" link rewritten
	} {
		if !strings.Contains(string(chapter), want) {
			t.Errorf("chapter one missing %s", want)
		}
	}

	// Page files stay root-relative; only links are prefixed.
	if _, err := os.Stat(filepath.Join(out, "one.html")); err != nil {
		t.Errorf("chapter exported at wrong path: %v", err)
	}
}

// The book's own stylesheet must carry the same cache-busting query the host
// site uses. Without it, krewire.css would be the only unversioned stylesheet
// on a page whose other assets all refresh on release.
func TestBuildVersionsItsStylesheet(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-one.md": "# One\n"})
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B", Version: "0.1.0"}); err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `href="/assets/krewire.css?v=0.1.0"`) {
		t.Errorf("index.html does not version its stylesheet:\n%s", index)
	}

	chapter, err := os.ReadFile(filepath.Join(out, "one.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(chapter), `href="/assets/krewire.css?v=0.1.0"`) {
		t.Errorf("chapter does not version its stylesheet:\n%s", chapter)
	}
}

// A declared version may already carry the "v" prefix; the query must not
// double it, and the rendered credit line must show one "v".
func TestBuildStylesheetVersionToleratesVPrefix(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-one.md": "# One\n"})
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B", Version: "v1.2.3"}); err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(index)
	if !strings.Contains(got, `href="/assets/krewire.css?v=1.2.3"`) {
		t.Errorf("stylesheet query = wrong form:\n%s", got)
	}
	if strings.Contains(got, "vv") {
		t.Error("version must not render a doubled v prefix")
	}
}

// A book rendered without a version must not invent a cache-busting query.
func TestBuildWithoutVersionLeavesStylesheetUnversioned(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-one.md": "# One\n"})
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B"}); err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `href="/assets/krewire.css"`) {
		t.Errorf("index.html missing the plain stylesheet link:\n%s", index)
	}
	if strings.Contains(string(index), "krewire.css?") {
		t.Error("no version was declared, so the query must be absent")
	}
}

func TestBuildWithMountPath(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-one.md": "# One\n\nSee the [next](/two) chapter.\n",
		"02-two.md": "# Two\n",
	})
	out := filepath.Join(t.TempDir(), "build")
	created, err := Build(Config{Input: dir, Output: out, Title: "B", Author: "A", MountPath: "/docs/"})
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		filepath.Join(out, "docs", "index.html"),
		filepath.Join(out, "docs", "one.html"),
		filepath.Join(out, "docs", "assets", "krewire.css"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("mounted export missing %s: %v", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); !os.IsNotExist(err) {
		t.Error("mounted book must not write a root index.html")
	}
	for _, got := range created {
		if _, err := os.Stat(got); err != nil {
			t.Errorf("created path missing: %s", got)
		}
	}

	index, err := os.ReadFile(filepath.Join(out, "docs", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/docs/assets/krewire.css"`, `href="/docs/one"`} {
		if !strings.Contains(string(index), want) {
			t.Errorf("mounted index missing %q", want)
		}
	}

	chapter, err := os.ReadFile(filepath.Join(out, "docs", "one.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/docs/two"`, `href="/docs/"`} {
		if !strings.Contains(string(chapter), want) {
			t.Errorf("mounted chapter missing %q", want)
		}
	}
}

func TestHandlerServesMountedBook(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-hello.md": "# Hello\n"})
	b, err := LoadWithBase(dir, "T", "", "/docs/")
	if err != nil {
		t.Fatal(err)
	}
	b.mount = "docs"
	h, err := b.Handler()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/hello", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/docs/hello = %d, want 200", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unmounted /hello = %d, want 404", rec.Code)
	}
}

func TestNormalizeBase(t *testing.T) {
	tt := []struct{ in, want string }{
		{"", "/"},
		{"/", "/"},
		{"guide", "/guide/"},
		{"/guide", "/guide/"},
		{"guide/", "/guide/"},
		{"/guide/", "/guide/"},
	}
	for _, tc := range tt {
		if got := normalizeBase(tc.in); got != tc.want {
			t.Errorf("normalizeBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPrefixLinks(t *testing.T) {
	if got := prefixLinks(`<a href="/getting-started">`, "/guide/"); got != `<a href="/guide/getting-started">` {
		t.Errorf("prefixLinks = %q", got)
	}
	if got := prefixLinks(`<a href="/">`, "/guide/"); got != `<a href="/guide/">` {
		t.Errorf("prefixLinks root = %q", got)
	}
	if got := prefixLinks(`<a href="/guide/y">`, "/guide/"); got != `<a href="/guide/y">` {
		t.Errorf("prefixLinks double = %q", got)
	}
	if got := prefixLinks(`<img src="/img.png">`, "/guide/"); got != `<img src="/guide/img.png">` {
		t.Errorf("prefixLinks src = %q", got)
	}
	if got := prefixLinks(`<a href="/getting-started">`, "/"); got != `<a href="/getting-started">` {
		t.Errorf("prefixLinks root base = %q", got)
	}
	if got := prefixLinks(`<a href="//example.com">`, "/guide/"); got != `<a href="//example.com">` {
		t.Errorf("prefixLinks protocol-relative = %q", got)
	}
}

func TestPageLink(t *testing.T) {
	if got := pageLink("/intro"); got != "/intro" {
		t.Errorf("pageLink = %q", got)
	}
	if got := pageLink("/intro/"); got != "/intro" {
		t.Errorf("pageLink slashed = %q, want extensionless", got)
	}
	if got := pageLink("/"); got != "/" {
		t.Errorf("pageLink root = %q", got)
	}
	if got := pageLink("/tutorials/setup"); got != "/tutorials/setup" {
		t.Errorf("pageLink nested = %q", got)
	}
	if got := pageLink("/assets/krewire.css"); got != "/assets/krewire.css" {
		t.Errorf("pageLink asset = %q", got)
	}
}

func TestChrome(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-one.md": "# One\n",
		"02-two.md": "# Two\n",
	})
	out := filepath.Join(t.TempDir(), "site")
	_, err := Build(Config{
		Input:      dir,
		Output:     out,
		Title:      "B",
		Author:     "Reas",
		BasePath:   "/docs/",
		NavLinks:   []Link{{Text: "Krewire", URL: "/"}, {Text: "GitHub", URL: "https://github.com/krewire"}},
		FooterText: "Krewire Book — MIT",
	})
	if err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="topbar"`,
		`href="/docs/"`,      // navbar brand
		`>Krewire</a>`,       // navbar extra link
		`>GitHub</a>`,        // navbar extra link
		`class="sidebar"`,    // sidebar
		`href="/docs/one"`,   // sidebar entry
		`href="/docs/two"`,   // sidebar entry
		`Krewire Book — MIT`, // footer text
		`class="start"`,      // start reading CTA
	} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	chapter, err := os.ReadFile(filepath.Join(out, "one.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="topbar"`,
		`class="sidebar"`,
		`toc-link active`, // current chapter highlighted
		`href="/docs/two"`,
		`Krewire Book — MIT`,
		`class="pager"`,
		`href="/docs/two"`, // next link
	} {
		if !strings.Contains(string(chapter), want) {
			t.Errorf("chapter one missing %q", want)
		}
	}

	if strings.Contains(string(chapter), `href="/docs/two"`) && strings.Contains(string(chapter), `toc-link active`) {
		// ensure only the active one has active class — count occurrences
		if strings.Count(string(chapter), `toc-link active`) != 1 {
			t.Error("non-current chapter must not be active in the sidebar")
		}
	}
}

func TestThemeSwitcher(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-one.md": "# One\n"})
	out := filepath.Join(t.TempDir(), "site")
	_, err := Build(Config{
		Input:    dir,
		Output:   out,
		Title:    "B",
		BasePath: "/guide/",
		Theme:    &Theme{StorageKey: "site-theme", Default: "dark"},
	})
	if err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-theme-toggle`,
		`icon-sun`,
		`icon-moon`,
		`site-theme`, // custom storage key
		`<meta name="color-scheme" content="light dark">`,
		`window.krewireTheme`,
	} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestThemeDisabledByDefault(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-one.md": "# One\n"})
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Build(Config{Input: dir, Output: out, Title: "B"}); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(index), "data-theme-toggle") {
		t.Error("theme switcher must be absent when Theme is unset")
	}
}

func TestLoadStripsFrontmatter(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-intro.md": "---\ntitle: \"Ignored\"\ndate: \"2026-08-25\"\n---\n\n# Real Title\n\nBody.\n",
	})
	b, err := Load(dir, "B", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Chapters[0].Title != "Real Title" {
		t.Errorf("title = %q, want Real Title from H1 after stripping frontmatter", b.Chapters[0].Title)
	}
	if strings.Contains(string(b.Chapters[0].Body), "frontmatter") || strings.Contains(string(b.Chapters[0].Body), "Ignored") {
		t.Errorf("body must not contain frontmatter: %q", b.Chapters[0].Body)
	}
	if !strings.Contains(string(b.Chapters[0].Body), "<p>Body.</p>") {
		t.Errorf("body = %q, want rendered markdown", b.Chapters[0].Body)
	}
}

func TestNoRootTOCSuppressesIndexPage(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-a.md": "# A\n"})
	out := filepath.Join(t.TempDir(), "build")
	created, err := Build(Config{Input: dir, Output: out, Title: "T", NoRootTOC: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); !os.IsNotExist(err) {
		t.Error("NoRootTOC must suppress the root index.html")
	}
	if _, err := os.Stat(filepath.Join(out, "a.html")); err != nil {
		t.Errorf("chapter page must still be exported: %v", err)
	}
	for _, p := range created {
		if strings.HasSuffix(filepath.ToSlash(p), "/index.html") {
			t.Errorf("created paths must not include a root index.html: %v", created)
		}
	}
}

func TestDefaultExcludesReadmeNotes(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"01-a.md":            "# A\n",
		"README.md":          "# Root notes\n",
		"docs/README.md":     "# Nested notes\n",
		"docs/readme.md":     "# lowercase notes\n",
		"docs/02-getting.md": "# Getting\n",
	})
	b, err := Load(dir, "B", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("got %d chapters, want 2 ([a] + [docs/getting]; README/readme excluded)", len(b.Chapters))
	}
	if b.Chapters[0].Slug != "a" {
		t.Errorf("chapter 1 = %q, want a", b.Chapters[0].Slug)
	}
	if b.Chapters[1].Slug != "docs" || len(b.Chapters[1].Subs) != 1 || b.Chapters[1].Subs[0].Slug != "getting" {
		t.Errorf("docs chapter = %+v, want single sub getting", b.Chapters[1])
	}
	for _, ch := range b.flattened() {
		if ch.Slug == "readme" {
			t.Error("README notes must be excluded by default")
		}
	}
}

func TestIncludeExcludeRules(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"guide/01-a.md":  "# A\n",
		"guide/02-b.md":  "# B\n",
		"api/01-rest.md": "# Rest\n",
		"drafts/x.md":    "# X\n",
	})

	b, err := LoadWithRules(dir, "B", "", "/", []string{"guide/**", "api/**"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("include filter: got %d chapters (%v), want guide+api", len(b.Chapters), b.Chapters)
	}

	b, err = LoadWithRules(dir, "B", "", "/", nil, []string{"drafts/**"})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("exclude filter: got %d chapters (%v), want guide+api only", len(b.Chapters), b.Chapters)
	}
	for _, ch := range b.flattened() {
		if ch.Slug == "x" || ch.Slug == "drafts" {
			t.Errorf("exclude drafts/** leaked %q", ch.Slug)
		}
	}

	empty := []string{}
	b, err = LoadWithRules(dir, "B", "", "/", empty, empty)
	if err != nil {
		t.Fatal(err)
	}
	// Filtering fully disabled: guide(1+2) + api(1+1) + drafts(1+1) = 7 pages.
	if len(b.flattened()) != 7 {
		t.Fatalf("explicit empty rules must disable filtering, got %d pages", len(b.flattened()))
	}
}

func TestSplitOrder(t *testing.T) {
	tt := []struct {
		in    string
		order int
		stem  string
	}{
		{"01-intro.md", 1, "-intro"},
		{"12-chapter.md", 12, "-chapter"},
		{"notes.md", int(^uint(0) >> 1), "notes"},
	}
	for _, tc := range tt {
		gotOrder, gotStem := splitOrder(tc.in)
		if gotOrder != tc.order || gotStem != tc.stem {
			t.Errorf("splitOrder(%q) = (%d, %q), want (%d, %q)", tc.in, gotOrder, gotStem, tc.order, tc.stem)
		}
	}
}

func TestSlugFor(t *testing.T) {
	tt := []struct {
		in, want string
	}{
		{"01-getting started.md", "getting-started"},
		{"chapter - one.md", "chapter-one"},
		{"NOTES.md", "notes"},
		{"12-Some Title.md", "some-title"},
	}
	for _, tc := range tt {
		if got := slugFor(tc.in); got != tc.want {
			t.Errorf("slugFor(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTitleFor(t *testing.T) {
	if got := titleFor([]byte("# Real Title\nbody"), "slug"); got != "Real Title" {
		t.Errorf("titleFor = %q, want Real Title", got)
	}
	if got := titleFor([]byte("## Ignore Me\n# Take Me\n"), "slug"); got != "Take Me" {
		t.Errorf("titleFor = %q, want Take Me", got)
	}
	if got := titleFor([]byte("no headings\n"), "the-slug"); got != "the-slug" {
		t.Errorf("titleFor = %q, want the-slug", got)
	}
}

func TestFrontmatterOrderAndNavigation(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"beta/index.md":   "---\ntitle: Beta Chapter\norder: 2\nprev: /alpha\nnext: none\n---\n# Beta Chapter\nBeta body",
		"alpha/index.md":  "---\ntitle: Alpha Chapter\norder: 1\nnext: /beta\n---\n# Alpha Chapter\nAlpha body",
		"alpha/second.md": "---\ntitle: Second Section\norder: 2\n---\n# Second Section\nSecond body",
		"alpha/first.md":  "---\ntitle: First Section\norder: 1\n---\n# First Section\nFirst body",
	})
	b, err := Load(dir, "Test Book", "Author")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(b.Chapters))
	}
	if b.Chapters[0].Title != "Alpha Chapter" || b.Chapters[1].Title != "Beta Chapter" {
		t.Errorf("chapters not ordered by frontmatter order: got %s, %s", b.Chapters[0].Title, b.Chapters[1].Title)
	}
	subs := b.Chapters[0].Subs
	if len(subs) != 2 {
		t.Fatalf("expected 2 subchapters in alpha, got %d", len(subs))
	}
	if subs[0].Title != "First Section" || subs[1].Title != "Second Section" {
		t.Errorf("subchapters not ordered by frontmatter order: got %s, %s", subs[0].Title, subs[1].Title)
	}

	pages, err := b.Pages()
	if err != nil {
		t.Fatal(err)
	}
	// Find beta page and check prev link
	for _, p := range pages {
		if p.Path == "/beta" {
			var buf bytes.Buffer
			if err := p.Render(&buf); err != nil {
				t.Fatal(err)
			}
			html := buf.String()
			if !strings.Contains(html, `href="/alpha"`) {
				t.Errorf("/beta missing custom prev link to /alpha, got: %s", html)
			}
			if strings.Contains(html, `class="next"`) {
				t.Errorf("/beta should have next suppressed with next: none, got: %s", html)
			}
		}
	}
}

func TestRootIndexMarkdown(t *testing.T) {
	dir := writeContent(t, map[string]string{
		"00-index.md":          "---\ntitle: Documentation Overview\n---\n# Documentation Overview\nWelcome to docs",
		"01-guide/00-index.md": "---\ntitle: Guide Index\n---\n# Guide Index\nGuide body",
		"01-guide/01-first.md": "---\ntitle: First Guide\n---\n# First Guide\nFirst guide body",
	})
	b, err := Load(dir, "Doc Site", "Author")
	if err != nil {
		t.Fatal(err)
	}
	if b.indexChapter == nil {
		t.Fatal("expected indexChapter to be loaded from root 00-index.md")
	}
	if len(b.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(b.Chapters))
	}
	if b.Chapters[0].Title != "Guide Index" {
		t.Errorf("chapter 0 title = %q, want 'Guide Index'", b.Chapters[0].Title)
	}
	pages, err := b.Pages()
	if err != nil {
		t.Fatal(err)
	}
	foundRoot := false
	for _, p := range pages {
		if p.Path == "/" {
			foundRoot = true
			var buf bytes.Buffer
			if err := p.Render(&buf); err != nil {
				t.Fatal(err)
			}
			html := buf.String()
			if !strings.Contains(html, "Documentation Overview") || !strings.Contains(html, "Welcome to docs") {
				t.Errorf("root page did not render 00-index.md body, got: %s", html)
			}
		}
	}
	if !foundRoot {
		t.Error("root page '/' not found in Pages()")
	}
}

func TestLocalizedNavLinksAndLocale(t *testing.T) {
	b := &Book{
		base:  "/id/docs/",
		mount: "docs",
		navLinks: []Link{
			{Text: "Get Started", URL: "/docs/getting-started"},
			{Text: "Workloads", URL: "/docs/workloads"},
			{Text: "Ecosystem", URL: "/#ecosystem"},
			{Text: "GitHub", URL: "https://github.com/krewire/krewire"},
		},
	}

	if loc := b.locale(); loc != "id" {
		t.Fatalf("expected locale 'id', got %q", loc)
	}

	links := b.localizedNavLinks()
	if len(links) != 4 {
		t.Fatalf("expected 4 links, got %d", len(links))
	}

	if links[0].URL != "/id/docs/getting-started" {
		t.Errorf("expected '/id/docs/getting-started', got %q", links[0].URL)
	}
	if links[1].URL != "/id/docs/workloads" {
		t.Errorf("expected '/id/docs/workloads', got %q", links[1].URL)
	}
	if links[2].URL != "/#ecosystem" {
		t.Errorf("expected '/#ecosystem' untouched, got %q", links[2].URL)
	}
	if links[3].URL != "https://github.com/krewire/krewire" {
		t.Errorf("expected GitHub URL untouched, got %q", links[3].URL)
	}

	pd := b.pageData(nil, nil)
	if pd.Locale != "id" {
		t.Errorf("expected pd.Locale 'id', got %q", pd.Locale)
	}
	if pd.HomeURL != "/id/docs/" {
		t.Errorf("expected pd.HomeURL '/id/docs/', got %q", pd.HomeURL)
	}
}

func TestBuildEmitsConfiguredLang(t *testing.T) {
	dir := writeContent(t, map[string]string{"01-a.md": "# A\n"})

	t.Run("default", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "site")
		if _, err := Build(Config{Input: dir, Output: out, Title: "T"}); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(filepath.Join(out, "a.html"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `<html lang="en">`) {
			t.Errorf("page = %q, want default <html lang=\"en\">", body)
		}
	})

	t.Run("explicit locale", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "site")
		if _, err := Build(Config{Input: dir, Output: out, Title: "T", Lang: "id"}); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"a.html", "index.html"} {
			body, err := os.ReadFile(filepath.Join(out, name))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), `<html lang="id">`) {
				t.Errorf("%s = %q, want <html lang=\"id\">", name, body)
			}
		}
	})
}
