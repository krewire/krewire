package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/auth"
)

func TestClient_URLs(t *testing.T) {
	client := auth.NewClient("https://auth.krewire.com")

	if got := client.LoginURL(""); got != "https://auth.krewire.com/login" {
		t.Errorf("unexpected LoginURL: %s", got)
	}

	target := "https://cloud.krewire.com/console"
	loginWithReturn := client.LoginURL(target)
	if !strings.Contains(loginWithReturn, "return_to=https%3A%2F%2Fcloud.krewire.com%2Fconsole") {
		t.Errorf("unexpected LoginURL with return_to: %s", loginWithReturn)
	}

	registerWithReturn := client.RegisterURL(target)
	if !strings.Contains(registerWithReturn, "return_to=") {
		t.Errorf("unexpected RegisterURL: %s", registerWithReturn)
	}

	logoutWithReturn := client.LogoutURL(target)
	if !strings.Contains(logoutWithReturn, "return_to=") {
		t.Errorf("unexpected LogoutURL: %s", logoutWithReturn)
	}
}

func TestClient_VerifySession(t *testing.T) {
	var callCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)

		if r.URL.Path != "/api/userinfo" {
			http.NotFound(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":        "usr_123",
			"email":      "dev@krewire.com",
			"name":       "Krewire Dev",
			"tenant_id":  "ten_core",
			"roles":      []string{"admin"},
			"session_id": "valid-token",
		})
	}))
	defer server.Close()

	client := auth.NewClient(server.URL, auth.WithCacheTTL(100*time.Millisecond))

	// 1. Valid token
	u, err := client.VerifySession(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "usr_123" || u.Email != "dev@krewire.com" || u.TenantID != "ten_core" {
		t.Errorf("unexpected user data: %+v", u)
	}

	// 2. Cached verification (should not trigger new HTTP request)
	uCached, err := client.VerifySession(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error from cache: %v", err)
	}
	if uCached.ID != "usr_123" {
		t.Errorf("unexpected cached user data: %+v", uCached)
	}
	if calls := atomic.LoadInt64(&callCount); calls != 1 {
		t.Errorf("expected 1 remote call due to caching, got %d", calls)
	}

	// 3. Invalidate session cache
	client.InvalidateSession("valid-token")
	_, _ = client.VerifySession(context.Background(), "valid-token")
	if calls := atomic.LoadInt64(&callCount); calls != 2 {
		t.Errorf("expected 2 remote calls after cache invalidate, got %d", calls)
	}

	// 4. Invalid token
	_, err = client.VerifySession(context.Background(), "invalid-token")
	if err == nil {
		t.Errorf("expected error for invalid token, got nil")
	}

	// 5. Empty token
	_, err = client.VerifySession(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty token, got nil")
	}
}

func TestRequireAuth_Middleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/userinfo" && r.Header.Get("Authorization") == "Bearer tok_pass" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":       "usr_pass",
				"email":     "test@krewire.com",
				"tenant_id": "ten_pass",
			})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := auth.NewClient(server.URL)
	mw := auth.RequireAuth(client, "krewire_session")

	securedHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.UserFrom(r.Context())
		if u == nil {
			t.Errorf("expected user in context")
		}
		id := auth.IdentityFrom(r.Context())
		if id == nil || id.Subject != "usr_pass" {
			t.Errorf("expected identity with subject usr_pass")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Case 1: Browser request unauthenticated -> Redirect to LoginURL
	req1 := httptest.NewRequest(http.MethodGet, "https://cloud.krewire.com/console", nil)
	req1.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec1 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d", rec1.Code)
	}
	loc := rec1.Header().Get("Location")
	if !strings.HasPrefix(loc, server.URL+"/login?return_to=") {
		t.Errorf("unexpected redirect location: %s", loc)
	}

	// Case 2: API request unauthenticated -> 401 Unauthorized
	req2 := httptest.NewRequest(http.MethodGet, "https://cloud.krewire.com/api/projects", nil)
	req2.Header.Set("Accept", "application/json")
	rec2 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec2.Code)
	}

	// Case 3: Authenticated via cookie -> Success
	req3 := httptest.NewRequest(http.MethodGet, "https://cloud.krewire.com/console", nil)
	req3.AddCookie(&http.Cookie{Name: "krewire_session", Value: "tok_pass"})
	rec3 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec3.Code)
	}
}
