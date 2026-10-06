package sec_test

// Tests for SEC-A05-002.
import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/packages/sec"
)

func TestCORS_WildcardDefault(t *testing.T) {
	handler := sec.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Non-CORS request without Origin header
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS header for non-CORS request")
	}

	// Simple GET request with Origin
	req = httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
}

func TestCORS_PreflightOptions(t *testing.T) {
	called := false
	handler := sec.CORS(
		sec.WithOrigins("https://app.example.com"),
		sec.WithMethods(http.MethodGet, http.MethodPost),
		sec.WithHeaders("X-Custom-Header"),
		sec.WithCredentials(true),
		sec.WithMaxAge(3600),
		sec.WithExposeHeaders("X-Total-Count"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Errorf("preflight request should not call next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected origin https://app.example.com, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credentials true, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Errorf("expected allowed methods 'GET, POST', got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Custom-Header" {
		t.Errorf("expected allowed headers 'X-Custom-Header', got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("expected max-age 3600, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Total-Count" {
		t.Errorf("expected expose headers 'X-Total-Count', got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary: Origin header, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	handler := sec.CORS(
		sec.WithOrigins("https://trusted.com"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secret"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://untrusted.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin should not receive Access-Control-Allow-Origin header")
	}
}

func TestCORS_WildcardWithCredentialsDoesNotReflectOrigin(t *testing.T) {
	handler := sec.CORS(sec.WithOrigins("*"), sec.WithCredentials(true))(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://attacker.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("wildcard credentials must not reflect origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("credentials must not be enabled for wildcard origin, got %q", got)
	}
}
