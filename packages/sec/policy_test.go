package sec

// Tests for SEC-A01-001.
import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/packages/auth"
)

func TestPolicyRequire(t *testing.T) {
	adminGate := Require(Authenticated(), WithRoles("admin"))
	handler := adminGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Anonymous -> 401
	req1 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", rec1.Code)
	}

	// User without role -> 403
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req2 = req2.WithContext(auth.WithIdentity(req2.Context(), &auth.Identity{Subject: "user", Roles: []string{"viewer"}}))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Errorf("viewer status = %d, want 403", rec2.Code)
	}

	// User with role -> 200
	req3 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req3 = req3.WithContext(auth.WithIdentity(req3.Context(), &auth.Identity{Subject: "admin", Roles: []string{"admin"}}))
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("admin status = %d, want 200", rec3.Code)
	}
}

func TestPolicySet(t *testing.T) {
	set := PolicySet{
		"auth":  Authenticated(),
		"admin": WithRoles("admin"),
	}
	gate := set.Require("auth", "admin")
	handler := gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
