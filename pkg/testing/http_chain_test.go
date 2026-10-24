package test

import (
	"net/http"
	"testing"
)

func TestRequestBuilder_Valid(t *testing.T) {
	req := GET(t, "/hello").Query("q", "a").Header("X-Test", "1").Request()
	if req.URL.Path != "/hello" {
		t.Errorf("Path = %q want /hello", req.URL.Path)
	}
	if req.URL.Query().Get("q") != "a" {
		t.Errorf("Query q = %q want a", req.URL.Query().Get("q"))
	}
	if req.Header.Get("X-Test") != "1" {
		t.Errorf("Header = %q", req.Header.Get("X-Test"))
	}
	req2 := JSONRequest(t, "POST", "/api", map[string]string{"name": "a"}).Request()
	if req2.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req2.Header.Get("Content-Type"))
	}
}

func TestDo_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	Do(t, handler, GET(t, "/").Request()).Status(200).Contains("ok")
}

func TestResponse_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "v")
		http.SetCookie(w, &http.Cookie{Name: "sess", Value: "abc"})
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"name":"a"}`))
	})
	resp := Do(t, handler, GET(t, "/").Request())
	resp.Status(200).Header("X-Custom", "v").Contains(`"name"`)

	var out struct {
		Name string `json:"name"`
	}
	resp.JSON(&out)
	if out.Name != "a" {
		t.Errorf("JSON Name = %q", out.Name)
	}
	c := resp.Cookie("sess")
	if c == nil || c.Value != "abc" {
		t.Errorf("Cookie = %v", c)
	}
}

func TestServer_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi"))
	})
	srv := Server(t, handler)
	if srv.URL == "" {
		t.Error("Server URL empty")
	}
}

func TestBackwardCompat_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	req := NewRequest(t, "GET", "/", nil)
	rec := Record(handler, req)
	EqualStatus(t, rec, 200)
}
