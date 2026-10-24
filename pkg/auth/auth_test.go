package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("01234567890123456789012345678901")

func TestIdentityHasRole(t *testing.T) {
	id := &Identity{
		Subject: "alice",
		Method:  "jwt",
		Roles:   []string{"admin", "editor"},
	}
	if !id.HasRole("admin") {
		t.Error("expected HasRole(admin) = true")
	}
	if !id.HasRole("editor") {
		t.Error("expected HasRole(editor) = true")
	}
	if id.HasRole("viewer") {
		t.Error("expected HasRole(viewer) = false")
	}
}

func TestContextIdentity(t *testing.T) {
	ctx := context.Background()
	if IdentityFrom(ctx) != nil {
		t.Error("IdentityFrom(empty) should be nil")
	}
	id := &Identity{Subject: "bob", Method: "basic"}
	ctx = WithIdentity(ctx, id)
	got := IdentityFrom(ctx)
	if got == nil || got.Subject != "bob" || got.Method != "basic" {
		t.Errorf("IdentityFrom = %+v, want bob", got)
	}
}

func TestBasicAuthMiddleware(t *testing.T) {
	verify := func(id, pass string) (*Identity, error) {
		if id == "alice" && pass == "wonder" {
			return &Identity{Subject: id, Roles: []string{"reader"}}, nil
		}
		return nil, nil
	}
	mw := BasicAuth("krewire", verify)

	var got *Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = IdentityFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Happy path
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.SetBasicAuth("alice", "wonder")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got == nil || got.Subject != "alice" || got.Method != "basic" || !got.HasRole("reader") {
		t.Errorf("got identity = %+v", got)
	}

	// Bad credentials
	req2 := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req2.SetBasicAuth("alice", "wrong")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec2.Code)
	}
	if ch := rec2.Header().Get("WWW-Authenticate"); !strings.HasPrefix(ch, `Basic realm="krewire"`) {
		t.Errorf("challenge = %q", ch)
	}

	// Anonymous request
	req3 := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec3.Code)
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	claims := DefaultClaims("carol", time.Minute)
	claims["roles"] = []string{"admin"}
	tok, err := SignJWT(testSecret, claims)
	if err != nil {
		t.Fatal(err)
	}

	mw := JWTAuth(testSecret)
	var got *Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = IdentityFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Valid Bearer
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got == nil || got.Subject != "carol" || got.Method != "jwt" || !got.HasRole("admin") {
		t.Errorf("identity = %+v", got)
	}

	// Missing token
	req2 := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec2.Code)
	}

	// ContinueOnMissing option
	mwPass := JWTAuth(testSecret, func(o *JWTOptions) { o.ContinueOnMissing = true })
	handlerPass := mwPass(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req3 := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec3 := httptest.NewRecorder()
	handlerPass.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 with continue", rec3.Code)
	}
}

func TestBasicAuthRealmSanitization(t *testing.T) {
	mw := BasicAuth("krewire\r\nInjected-Header: evil\n\"quote\"", func(id, pass string) (*Identity, error) {
		return nil, nil
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	handler.ServeHTTP(rec, req)

	ch := rec.Header().Get("WWW-Authenticate")
	if strings.Contains(ch, "\r") || strings.Contains(ch, "\n") {
		t.Errorf("WWW-Authenticate contains CRLF: %q", ch)
	}
	expected := `Basic realm="krewireInjected-Header: evil\"quote\""`
	if ch != expected {
		t.Errorf("WWW-Authenticate = %q, want %q", ch, expected)
	}
}
