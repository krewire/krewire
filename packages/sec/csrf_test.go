package sec

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFSafeMethods(t *testing.T) {
	mw := CSRF()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := CSRFFrom(r.Context())
		w.Write([]byte(tok))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/form", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "XSRF-TOKEN=") {
		t.Errorf("Set-Cookie missing XSRF-TOKEN: %s", cookie)
	}
}

func TestCSRFUnsafeBlockedWithoutToken(t *testing.T) {
	mw := CSRF()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestCSRFUnsafeAllowedWithValidHeader(t *testing.T) {
	mw := CSRF()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := "valid-csrf-token-1234567890abcdef"
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: token})
	req.Header.Set("X-CSRF-Token", token)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestCSRFPrefixHost(t *testing.T) {
	mw := CSRF(func(o *CSRFOptions) {
		o.PrefixHost = true
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "__Host-XSRF-TOKEN=") {
		t.Errorf("expected __Host- prefix in cookie: %s", cookie)
	}
	if !strings.Contains(cookie, "Secure") {
		t.Errorf("expected Secure attribute on cookie: %s", cookie)
	}
}

func TestCSRFHMACSecret(t *testing.T) {
	secret := []byte("a-very-secret-csrf-signing-key-1")
	mw := CSRF(func(o *CSRFOptions) {
		o.Secret = secret
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1. GET request receives signed cookie
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	cookieHeader := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "XSRF-TOKEN=") {
		t.Fatalf("expected XSRF-TOKEN cookie in response: %s", cookieHeader)
	}

	// Extract cookie value
	var cookieVal string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "XSRF-TOKEN" {
			cookieVal = c.Value
			break
		}
	}
	parts := strings.Split(cookieVal, ".")
	if len(parts) != 2 {
		t.Fatalf("expected token.sig format, got: %s", cookieVal)
	}
	rawToken := parts[0]

	// 2. POST with valid signed cookie and matching header should pass
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/submit", nil)
	reqPost.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: cookieVal})
	reqPost.Header.Set("X-CSRF-Token", rawToken)
	handler.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Errorf("POST with valid signed cookie: status = %d, want 200", recPost.Code)
	}

	// 3. POST with tampered cookie signature should fail (cookie tossing defense)
	recTampered := httptest.NewRecorder()
	reqTampered := httptest.NewRequest(http.MethodPost, "/submit", nil)
	reqTampered.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: rawToken + ".invalidSignature"})
	reqTampered.Header.Set("X-CSRF-Token", rawToken)
	handler.ServeHTTP(recTampered, reqTampered)
	if recTampered.Code != http.StatusForbidden {
		t.Errorf("POST with tampered cookie: status = %d, want 403", recTampered.Code)
	}
}

func TestCSRFAuthorizationHeaderHandling(t *testing.T) {
	// Default: CSRF is NOT skipped just because Authorization header is present
	mwDefault := CSRF()
	handlerDefault := mwDefault(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req1.Header.Set("Authorization", "Bearer fake-token")
	handlerDefault.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusForbidden {
		t.Errorf("default CSRF should not bypass for Authorization header: got status %d, want 403", rec1.Code)
	}

	// With IgnoreAuthorization = true: CSRF is skipped if Authorization is present
	mwIgnore := CSRF(func(o *CSRFOptions) {
		o.IgnoreAuthorization = true
	})
	handlerIgnore := mwIgnore(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req2.Header.Set("Authorization", "Bearer fake-token")
	handlerIgnore.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("IgnoreAuthorization should bypass CSRF: got status %d, want 200", rec2.Code)
	}
}
