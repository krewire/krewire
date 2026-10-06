package sec_test

// Tests for SEC-A04-001.
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/packages/sec"
)

func TestHealth_Liveness(t *testing.T) {
	handler := sec.Health()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /healthz, got %d", rec.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", res["status"])
	}
}

func TestHealth_Readiness(t *testing.T) {
	// 1. Healthy checks
	handlerHealthy := sec.Health(
		sec.WithCheck("db", func(ctx context.Context) error { return nil }),
		sec.WithCheck("cache", func(ctx context.Context) error { return nil }),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handlerHealthy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /readyz healthy, got %d", rec.Code)
	}

	// 2. Degraded checks
	handlerDegraded := sec.Health(
		sec.WithCheck("db", func(ctx context.Context) error { return nil }),
		sec.WithCheck("redis", func(ctx context.Context) error { return errors.New("connection timeout") }),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec2 := httptest.NewRecorder()
	handlerDegraded.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable for degraded check, got %d", rec2.Code)
	}
}
