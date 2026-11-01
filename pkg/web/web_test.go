package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRouterDispatchesParamRoute(t *testing.T) {
	r := NewRouter()
	r.Get("/chapters/{slug}", func(w http.ResponseWriter, _ *http.Request, p Params) {
		HTML(w, http.StatusOK, "ok:"+p["slug"])
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/chapters/intro", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "ok:intro" {
		t.Errorf("body = %q, want ok:intro", got)
	}
}

func TestRouterMethodMismatchIsNotFound(t *testing.T) {
	r := NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request, _ Params) {
		HTML(w, http.StatusOK, "pong")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ping", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRouterUnmatchedIsNotFound(t *testing.T) {
	r := NewRouter()
	r.Get("/known", func(w http.ResponseWriter, _ *http.Request, _ Params) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/other", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRouterServesStaticFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRouter()
	r.Static("/assets", dir)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "body{}" {
		t.Errorf("body = %q, want body{}", got)
	}
}

func TestRouterStaticSecurity(t *testing.T) {
	dir := t.TempDir()
	outsideDir := t.TempDir()

	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super-secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	validFile := filepath.Join(dir, "valid.txt")
	if err := os.WriteFile(validFile, []byte("valid content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Symlink pointing outside static root
	if err := os.Symlink(secretFile, filepath.Join(dir, "escape-link.txt")); err != nil {
		t.Fatal(err)
	}

	// Symlink pointing inside static root
	if err := os.Symlink(validFile, filepath.Join(dir, "internal-link.txt")); err != nil {
		t.Fatal(err)
	}

	// Subdirectory without index.html
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inner.txt"), []byte("inner"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Subdirectory with index.html
	indexDir := filepath.Join(dir, "site")
	if err := os.Mkdir(indexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(indexDir, "index.html"), []byte("<h1>Welcome</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewRouter()
	r.Static("/static", dir)

	// 1. Symlink escape should be blocked with 404
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/escape-link.txt", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for symlink escape, got %d", rec.Code)
	}

	// 2. Internal symlink within root should be served
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/internal-link.txt", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for internal symlink, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "valid content" {
		t.Fatalf("expected 'valid content', got %q", got)
	}

	// 3. Directory without index.html should return 404 (no directory listing)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/sub", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for directory without index, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/sub/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for directory/ without index, got %d", rec.Code)
	}

	// 4. Directory with index.html should be served or redirected
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/site/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for directory with index.html, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "<h1>Welcome</h1>" {
		t.Fatalf("expected '<h1>Welcome</h1>', got %q", got)
	}
}

func TestTemplatesParseAndExecute(t *testing.T) {
	fsys := fstest.MapFS{
		"page.html": &fstest.MapFile{Data: []byte(`<h1>{{.Title}}</h1>`)},
	}
	ts, err := ParseTemplates(fsys, "*.html")
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := ts.Execute(&buf, "page.html", struct{ Title string }{Title: "Intro"}); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "<h1>Intro</h1>" {
		t.Errorf("rendered = %q, want <h1>Intro</h1>", got)
	}
}

func TestTemplatesEscapeUnsafeHTML(t *testing.T) {
	fsys := fstest.MapFS{
		"page.html": &fstest.MapFile{Data: []byte(`<p>{{.Body}}</p>`)},
	}
	ts, err := ParseTemplates(fsys, "*.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := ts.Execute(&buf, "page.html", struct{ Body string }{Body: "<script>bad()</script>"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "<script>") {
		t.Errorf("template did not escape unsafe HTML: %q", buf.String())
	}
}

func TestExportWritesPagesAndAssets(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	pages := []Page{
		{Path: "/", Render: func(w io.Writer) error { _, err := io.WriteString(w, "<html>index</html>"); return err }},
		{Path: "/chapters/intro", Render: func(w io.Writer) error { _, err := io.WriteString(w, "<article>intro</article>"); return err }},
	}
	assets := map[string]string{"assets/style.css": "body{}"}

	if err := Export(out, pages, assets); err != nil {
		t.Fatal(err)
	}

	assertFile(t, filepath.Join(out, "index.html"), "<html>index</html>")
	assertFile(t, filepath.Join(out, "chapters", "intro", "index.html"), "<article>intro</article>")
	assertFile(t, filepath.Join(out, "assets", "style.css"), "body{}")
}

func TestExportSanitizesAssetPaths(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "site")
	if err := Export(out, nil, map[string]string{"../../evil": "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "evil")); !os.IsNotExist(err) {
		t.Errorf("export leaked outside outDir: %v", err)
	}
	assertFile(t, filepath.Join(out, "assets", "_"), "x")
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Errorf("%s = %q, want %q", path, body, want)
	}
}
