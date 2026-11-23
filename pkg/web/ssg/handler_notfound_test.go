package ssg

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, string(body)
}

// TestHandlerUnknownPathIs404 is the regression test for the bug found by
// workload conformance (#141): the "/" route matched every unknown path and
// served the index page with 200.
func TestHandlerUnknownPathIs404(t *testing.T) {
	h := testSite().Handler()
	if code, body := get(t, h, "/"); code != http.StatusOK || !strings.Contains(body, "Welcome") {
		t.Fatalf("index: %d %q", code, body)
	}
	if code, _ := get(t, h, "/index.html"); code != http.StatusOK {
		t.Errorf("emitted file name /index.html: %d", code)
	}
	if code, _ := get(t, h, "/assets/style.css"); code != http.StatusOK {
		t.Errorf("asset: %d", code)
	}
	for _, p := range []string{"/no-such-page", "/assets/missing.css", "/deep/unknown/"} {
		if code, body := get(t, h, p); code != http.StatusNotFound || strings.Contains(body, "Welcome") {
			t.Errorf("%s: %d, want 404 without the index page", p, code)
		}
	}
}

func TestHandlerServesSite404Page(t *testing.T) {
	s := New().
		Component(Component{Name: "p", Body: `<p>{{.}}</p>`}).
		Layout(Layout{Name: "l", Body: `<!DOCTYPE html><html><body>{{.Content}}</body></html>`}).
		Page(Page{Path: "/", Layout: "l", Root: "p", Data: "home"}).
		Page(Page{Path: "/about", Layout: "l", Root: "p", Data: "about us"}).
		Page(Page{Path: "/404.html", Layout: "l", Root: "p", Data: "custom gone"})
	h := s.Handler()
	if code, body := get(t, h, "/missing"); code != http.StatusNotFound || !strings.Contains(body, "custom gone") {
		t.Fatalf("unknown path: %d %q, want the site's 404 page with status 404", code, body)
	}
	if code, body := get(t, h, "/about"); code != http.StatusOK || !strings.Contains(body, "about us") {
		t.Errorf("/about: %d %q", code, body)
	}
	if code, _ := get(t, h, "/about.html"); code != http.StatusOK {
		t.Errorf("emitted file name /about.html: %d", code)
	}
	if code, _ := get(t, h, "/404.html"); code != http.StatusOK {
		t.Errorf("explicit /404.html is a real file on a static host: %d", code)
	}
}
