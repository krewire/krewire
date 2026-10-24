package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/cloud/service/registry"
	"github.com/krewire/krewire/pkg/cloud/service/resilience"
)

func TestGateway_NoRoute_Returns404(t *testing.T) {
	g := New()
	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGateway_NoRegistry_Returns502(t *testing.T) {
	g := New()
	g.AddRoute(Route{Path: "/api", Service: "backend"})
	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestGateway_WithEndpoint_Proxies(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend"))
	}))
	defer backend.Close()

	reg := registry.NewMemoryRegistry()
	ctx := context.Background()
	_ = reg.Register(ctx, registry.Service{ID: "be-1", Name: "backend", Address: backend.Listener.Addr().String()})

	g := New(WithRegistry(reg))
	g.AddRoute(Route{Path: "/api", Service: "backend"})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGateway_PrefixAndStripPrefix(t *testing.T) {
	var receivedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	g := New()
	g.AddRoute(Route{
		Path:        "/api/v1/users",
		TargetURL:   backend.URL,
		Prefix:      true,
		StripPrefix: true,
	})

	req := httptest.NewRequest("GET", "/api/v1/users/42/profile", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedPath != "/42/profile" {
		t.Fatalf("expected stripped path /42/profile, got %q", receivedPath)
	}
}

func TestGateway_RoundRobinLoadBalancing(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}

	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts["s1"]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts["s2"]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer s2.Close()

	reg := registry.NewMemoryRegistry()
	ctx := context.Background()
	_ = reg.Register(ctx, registry.Service{ID: "s1", Name: "cluster", Address: s1.Listener.Addr().String()})
	_ = reg.Register(ctx, registry.Service{ID: "s2", Name: "cluster", Address: s2.Listener.Addr().String()})

	g := New(WithRegistry(reg))
	g.AddRoute(Route{Path: "/cluster", Service: "cluster"})

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/cluster", nil)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d failed: %d", i, w.Code)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if counts["s1"] != 3 || counts["s2"] != 3 {
		t.Fatalf("expected 3 requests each, got s1=%d, s2=%d", counts["s1"], counts["s2"])
	}
}

func TestGateway_TracingAndCORSMiddleware(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	g := New(
		WithMiddleware("tracing", TracingMiddleware()),
		WithMiddleware("cors", CORSMiddleware("*")),
	)
	g.AddRoute(Route{
		Path:       "/test",
		TargetURL:  backend.URL,
		Middleware: []string{"tracing", "cors"},
	})

	// Preflight OPTIONS
	reqOpt := httptest.NewRequest("OPTIONS", "/test", nil)
	wOpt := httptest.NewRecorder()
	g.ServeHTTP(wOpt, reqOpt)
	if wOpt.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", wOpt.Code)
	}
	if wOpt.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing Access-Control-Allow-Origin")
	}

	// GET request with X-Request-ID propagation
	reqGet := httptest.NewRequest("GET", "/test", nil)
	wGet := httptest.NewRecorder()
	g.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}
	if wGet.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID in response")
	}
}

func TestGateway_CircuitBreakerMiddleware(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold: 2,
		Timeout:   100 * time.Millisecond,
	})

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	g := New(WithMiddleware("cb", CircuitBreakerMiddleware(cb)))
	g.AddRoute(Route{
		Path:       "/fail",
		TargetURL:  failServer.URL,
		Middleware: []string{"cb"},
	})

	// Trigger 2 failures to open breaker
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/fail", nil)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
	}

	// 3rd request should fail fast with 503 circuit breaker open
	req := httptest.NewRequest("GET", "/fail", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 circuit breaker open, got %d", w.Code)
	}
}

func TestGateway_Reload_ReplacesRoutes(t *testing.T) {
	g := New()
	g.AddRoute(Route{Path: "/old", Service: "old-svc"})
	g.Reload([]Route{{Path: "/new", Service: "new-svc"}})

	req := httptest.NewRequest("GET", "/old", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatal("old route should not exist after reload")
	}
}

func TestWriteProblem_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writeProblem(w, http.StatusBadGateway, "service unavailable")
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected problem+json, got %q", ct)
	}
}
