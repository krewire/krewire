package sec

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitMiddleware(t *testing.T) {
	// 5 rps, burst of 2
	mw := RateLimit(5, 2, WithKeyFunc(func(r *http.Request) string {
		return "client-1"
	}))

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Request 1: success (tokens 2 -> 1)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("req1: expected 200, got %d", rr1.Code)
	}
	if rem := rr1.Header().Get("X-RateLimit-Remaining"); rem != "1" {
		t.Errorf("req1: expected remaining 1, got %q", rem)
	}

	// Request 2: success (tokens 1 -> 0)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("req2: expected 200, got %d", rr2.Code)
	}
	if rem := rr2.Header().Get("X-RateLimit-Remaining"); rem != "0" {
		t.Errorf("req2: expected remaining 0, got %q", rem)
	}

	// Request 3: immediate 429 rate limit exceeded
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusTooManyRequests {
		t.Fatalf("req3: expected 429, got %d", rr3.Code)
	}
	if retry := rr3.Header().Get("Retry-After"); retry != "1" {
		t.Errorf("req3: expected Retry-After 1, got %q", retry)
	}
}
