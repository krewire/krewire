package sec

import (
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripTags(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<p>Hello <b>World</b></p>", "Hello World"},
		{"<script>alert(1)</script>", "alert(1)"},
		{"<img src=\"foo>bar\" onerror=\"alert(1)\">after", "after"},
		{"<<script>script>alert(1)<</script>/script>", "alert(1)"},
		{"no tags", "no tags"},
		{"", ""},
	}
	for _, c := range cases {
		if got := StripTags(c.in); got != c.want {
			t.Errorf("StripTags(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	mw := SecurityHeaders()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	h := rec.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", h.Get("X-Content-Type-Options"))
	}
	if h.Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", h.Get("X-Frame-Options"))
	}
	if h.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q", h.Get("Referrer-Policy"))
	}
	if h.Get("Content-Security-Policy") != "default-src 'self'" {
		t.Errorf("Content-Security-Policy = %q", h.Get("Content-Security-Policy"))
	}
	if h.Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Errorf("HSTS = %q", h.Get("Strict-Transport-Security"))
	}
}

func TestSecurityHeadersCustom(t *testing.T) {
	mw := SecurityHeaders(func(o *SecurityOptions) {
		o.CSP = "default-src 'none'"
		o.Frame = "SAMEORIGIN"
		o.HSTS = 3600
		o.PermissionsPolicy = "geolocation=()"
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	h := rec.Header()
	if h.Get("Content-Security-Policy") != "default-src 'none'" {
		t.Errorf("CSP = %q", h.Get("Content-Security-Policy"))
	}
	if h.Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q", h.Get("X-Frame-Options"))
	}
	if h.Get("Strict-Transport-Security") != "max-age=3600; includeSubDomains" {
		t.Errorf("HSTS = %q", h.Get("Strict-Transport-Security"))
	}
	if h.Get("Permissions-Policy") != "geolocation=()" {
		t.Errorf("Permissions-Policy = %q", h.Get("Permissions-Policy"))
	}
}

func TestDependencyHygiene(t *testing.T) {
	// 1. Verify pkg/auth imports only standard library packages.
	authDir := filepath.Join("..", "auth")
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, authDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseDir auth: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				firstPart := strings.Split(path, "/")[0]
				if strings.Contains(firstPart, ".") {
					t.Errorf("pkg/auth imports non-stdlib package %q in %v", path, file.Name)
				}
			}
		}
	}

	// 2. Verify pkg/sec third-party imports are restricted to pinned golang.org/x/net/html.
	secDir := "."
	pkgsSec, err := parser.ParseDir(fset, secDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseDir sec: %v", err)
	}
	for _, pkg := range pkgsSec {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				firstPart := strings.Split(path, "/")[0]
				if strings.Contains(firstPart, ".") {
					if path != "github.com/krewire/krewire/pkg/auth" && !strings.HasPrefix(path, "golang.org/x/net/html") {
						t.Errorf("pkg/sec imports unauthorized third-party package %q in %v", path, file.Name)
					}
				}
			}
		}
	}
}
