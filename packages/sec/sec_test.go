package sec

// Tests for SEC-A03-002, SEC-A05-001.
import (
	"net/http"
	"net/http/httptest"
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
