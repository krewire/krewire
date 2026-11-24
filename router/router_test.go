package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/router"
)

func TestRouterBasic(t *testing.T) {
	r := router.NewRouter()
	r.Get("/hello/{name}", func(w http.ResponseWriter, req *http.Request, p router.Params) {
		w.Write([]byte("Hello, " + p["name"]))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello/krewire", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "Hello, krewire" {
		t.Fatalf("expected 'Hello, krewire', got %q", string(body))
	}
}

func TestRouterURL(t *testing.T) {
	r := router.NewRouter()
	r.GET("/users/{id}/profile").Name("users.profile").Handle(func(w http.ResponseWriter, req *http.Request, p router.Params) {})

	url, err := r.URL("users.profile", router.PathParams{"id": "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "/users/42/profile" {
		t.Fatalf("expected /users/42/profile, got %s", url)
	}
}
