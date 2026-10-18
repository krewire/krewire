package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/krewire/krewire/packages/cloud/service/resilience"
)

// CircuitBreakerMiddleware wraps an HTTP handler with a circuit breaker.
func CircuitBreakerMiddleware(cb *resilience.CircuitBreaker) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := cb.Do(r.Context(), func() error {
				rw := &statusResponseWriter{ResponseWriter: w, code: http.StatusOK}
				next.ServeHTTP(rw, r)
				if rw.code >= 500 {
					return &httpUpstreamError{code: rw.code}
				}
				return nil
			})
			if err != nil {
				if _, ok := err.(resilience.ErrCircuitOpen); ok {
					writeProblem(w, http.StatusServiceUnavailable, "circuit breaker open: "+err.Error())
				}
			}
		})
	}
}

// TimeoutMiddleware sets a deadline on the request context.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TracingMiddleware injects X-Request-ID and distributed trace headers.
func TracingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				b := make([]byte, 16)
				_, _ = rand.Read(b)
				reqID = hex.EncodeToString(b)
				r.Header.Set("X-Request-ID", reqID)
			}
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds standard Cross-Origin Resource Sharing headers.
func CORSMiddleware(allowedOrigins ...string) Middleware {
	origins := "*"
	if len(allowedOrigins) > 0 {
		origins = allowedOrigins[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware provides simple token-bucket rate limiting per route.
func RateLimitMiddleware(rps int) Middleware {
	if rps <= 0 {
		rps = 100
	}
	interval := time.Second / time.Duration(rps)
	var mu sync.Mutex
	lastRequest := time.Now()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			now := time.Now()
			elapsed := now.Sub(lastRequest)
			if elapsed < interval {
				mu.Unlock()
				w.Header().Set("Retry-After", "1")
				writeProblem(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			lastRequest = now
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

type httpUpstreamError struct {
	code int
}

func (e *httpUpstreamError) Error() string {
	return http.StatusText(e.code)
}
