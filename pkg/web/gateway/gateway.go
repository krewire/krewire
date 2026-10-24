// Package gateway provides an API gateway (KWF-L5H2F FRK-SVC-020/021/022/023).
// It owns a route table, proxies requests to services discovered via the
// service registry, and applies middleware (logging, tracing, CORS, auth,
// rate limiting, circuit breaker). Missing upstreams return 502 with a
// structured Problem JSON response.
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krewire/krewire/pkg/cloud/service/registry"
)

// Route maps a path/method pair to a target service (FRK-SVC-020).
type Route struct {
	Path        string        `json:"path"`
	Method      string        `json:"method,omitempty"`
	Service     string        `json:"service,omitempty"`
	TargetURL   string        `json:"targetUrl,omitempty"`
	Prefix      bool          `json:"prefix,omitempty"`
	StripPrefix bool          `json:"stripPrefix,omitempty"`
	Middleware  []string      `json:"middleware,omitempty"`
	RateLimit   int           `json:"rateLimit,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	RetryCount  int           `json:"retryCount,omitempty"`
	Auth        bool          `json:"auth,omitempty"`
}

// Gateway is the API gateway (FRK-SVC-020).
type Gateway struct {
	mu         sync.RWMutex
	routes     []Route
	registry   registry.Registry
	mws        map[string]Middleware
	lbCounters map[string]*uint64
}

// Middleware is a function that wraps an http.Handler (FRK-SVC-021).
type Middleware func(http.Handler) http.Handler

// Option configures a Gateway.
type Option func(*Gateway)

// WithRegistry sets the service registry for discovery.
func WithRegistry(r registry.Registry) Option {
	return func(g *Gateway) { g.registry = r }
}

// WithMiddleware registers named middleware.
func WithMiddleware(name string, mw Middleware) Option {
	return func(g *Gateway) { g.mws[name] = mw }
}

// New creates a gateway with the given options.
func New(opts ...Option) *Gateway {
	g := &Gateway{
		mws:        map[string]Middleware{},
		lbCounters: map[string]*uint64{},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// AddRoute registers a route (FRK-SVC-020).
func (g *Gateway) AddRoute(r Route) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes = append(g.routes, r)
}

// Reload atomically replaces the route table (FRK-SVC-022).
func (g *Gateway) Reload(routes []Route) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes = routes
}

// matchRoute searches for a route matching the request path and method.
// Exact matches are evaluated before prefix matches.
func (g *Gateway) matchRoute(path, method string) *Route {
	g.mu.RLock()
	routes := g.routes
	g.mu.RUnlock()

	// 1. Try exact matches first
	for i := range routes {
		if !routes[i].Prefix && routes[i].Path == path {
			if routes[i].Method == "" || routes[i].Method == method {
				return &routes[i]
			}
		}
	}

	// 2. Try prefix matches
	for i := range routes {
		if routes[i].Prefix {
			p := routes[i].Path
			if strings.HasPrefix(path, p) {
				if routes[i].Method == "" || routes[i].Method == method {
					return &routes[i]
				}
			}
		}
	}

	return nil
}

// ServeHTTP implements http.Handler (FRK-SVC-020).
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matched := g.matchRoute(r.URL.Path, r.Method)
	if matched == nil {
		writeProblem(w, http.StatusNotFound, "no route for "+r.URL.Path)
		return
	}

	// Clone request so modifications like StripPrefix don't leak upstream
	req := r.Clone(r.Context())
	if matched.Prefix && matched.StripPrefix {
		trimmed := strings.TrimPrefix(req.URL.Path, matched.Path)
		if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		req.URL.Path = trimmed
	}

	// Build middleware chain
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if matched.TargetURL != "" {
			g.proxyDirect(w, r, matched.TargetURL)
		} else {
			g.proxyTo(w, r, matched.Service)
		}
	})

	for i := len(matched.Middleware) - 1; i >= 0; i-- {
		if mw, ok := g.mws[matched.Middleware[i]]; ok {
			handler = mw(handler)
		}
	}
	handler.ServeHTTP(w, req)
}

// nextIndex returns a round-robin index for the given service.
func (g *Gateway) nextIndex(service string, count int) int {
	if count <= 1 {
		return 0
	}
	g.mu.Lock()
	counter, ok := g.lbCounters[service]
	if !ok {
		var zero uint64
		counter = &zero
		g.lbCounters[service] = counter
	}
	g.mu.Unlock()

	val := atomic.AddUint64(counter, 1)
	return int((val - 1) % uint64(count))
}

// proxyDirect proxies the request to a fixed URL.
func (g *Gateway) proxyDirect(w http.ResponseWriter, r *http.Request, targetURL string) {
	u, err := url.Parse(schemeHostURL(targetURL))
	if err != nil {
		writeProblem(w, http.StatusBadGateway, "invalid target URL: "+targetURL)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeProblem(w, http.StatusBadGateway, "upstream error: "+err.Error())
	}
	proxy.ServeHTTP(w, r)
}

// proxyTo discovers the service and proxies the request with round-robin load balancing (FRK-SVC-020).
func (g *Gateway) proxyTo(w http.ResponseWriter, r *http.Request, serviceName string) {
	if g.registry == nil {
		writeProblem(w, http.StatusBadGateway, "no registry configured")
		return
	}
	eps, err := g.registry.Discover(r.Context(), serviceName)
	if err != nil || len(eps) == 0 {
		writeProblem(w, http.StatusBadGateway, "service "+serviceName+" unavailable")
		return
	}

	idx := g.nextIndex(serviceName, len(eps))
	target := eps[idx].Address

	u, err := url.Parse(schemeHostURL(target))
	if err != nil {
		writeProblem(w, http.StatusBadGateway, "invalid upstream address")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeProblem(w, http.StatusBadGateway, "upstream error: "+err.Error())
	}
	proxy.ServeHTTP(w, r)
}

// Problem is a RFC 7807 Problem Details response (FRK-SVC-023).
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// writeProblem writes a RFC 7807 Problem JSON response (FRK-SVC-023).
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}

// schemeHostURL parses a host address into a URL for reverse proxy.
func schemeHostURL(addr string) string {
	if len(addr) > 7 && addr[:7] == "http://" {
		return addr
	}
	if len(addr) > 8 && addr[:8] == "https://" {
		return addr
	}
	return "http://" + addr
}
