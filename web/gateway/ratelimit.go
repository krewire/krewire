package gateway

import (
	"sync"
	"time"
)

// routeLimiter is the per-route fixed-interval limiter behind Route.RateLimit.
// It mirrors RateLimitMiddleware's semantics: RateLimit is requests per second,
// so two requests for the same route must be at least 1/rps apart.
type routeLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// allow reports whether a request may proceed at now, adopting interval as the
// current spacing so a Reload that changes rate_limit takes effect immediately.
func (l *routeLimiter) allow(now time.Time, interval time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.interval = interval
	if !l.last.IsZero() && now.Sub(l.last) < interval {
		return false
	}
	l.last = now
	return true
}

// allow reports whether route may serve a request at now, creating and caching
// the route's limiter on first use. Limiters are keyed by path and method, so a
// route keeps its budget across Reload calls.
func (g *Gateway) allow(route *Route, now time.Time) bool {
	key := route.Path + "|" + route.Method
	interval := time.Second / time.Duration(route.RateLimit)

	g.mu.Lock()
	l, ok := g.limiters[key]
	if !ok {
		l = &routeLimiter{}
		g.limiters[key] = l
	}
	g.mu.Unlock()

	return l.allow(now, interval)
}
