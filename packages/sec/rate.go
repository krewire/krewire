package sec

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitOptions configures the rate limiting middleware.
type RateLimitOptions struct {
	KeyFunc func(*http.Request) string
}

// RateLimitOption is a functional option for RateLimitOptions.
type RateLimitOption func(*RateLimitOptions)

// WithKeyFunc configures a custom key extractor for the rate limiter.
func WithKeyFunc(fn func(*http.Request) string) RateLimitOption {
	return func(o *RateLimitOptions) {
		o.KeyFunc = fn
	}
}

type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	rps     float64
	burst   float64
	buckets map[string]*tokenBucket
	keyFunc func(*http.Request) string
}

func defaultKeyFunc(r *http.Request) string {
	// Extract client IP, prioritizing remote addr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.TrimSpace(host)
}

// RateLimit returns middleware applying a token-bucket rate limiter per key.
// Requests exceeding capacity are rejected with HTTP 429 Too Many Requests
// and appropriate standard headers (Retry-After, X-RateLimit-*).
func RateLimit(rps float64, burst int, opts ...RateLimitOption) Middleware {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = 20
	}

	cfg := &RateLimitOptions{
		KeyFunc: defaultKeyFunc,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	limiter := &rateLimiter{
		rps:     rps,
		burst:   float64(burst),
		buckets: make(map[string]*tokenBucket),
		keyFunc: cfg.KeyFunc,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := limiter.keyFunc(r)
			if key == "" {
				key = "anonymous"
			}

			limiter.mu.Lock()
			now := time.Now()
			b, exists := limiter.buckets[key]
			if !exists {
				b = &tokenBucket{
					tokens:    limiter.burst,
					lastCheck: now,
				}
				limiter.buckets[key] = b
			} else {
				elapsed := now.Sub(b.lastCheck).Seconds()
				b.tokens = math.Min(limiter.burst, b.tokens+elapsed*limiter.rps)
				b.lastCheck = now
			}

			if b.tokens < 1.0 {
				limiter.mu.Unlock()
				w.Header().Set("Retry-After", "1")
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			b.tokens -= 1.0
			remaining := int(b.tokens)
			limiter.mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			next.ServeHTTP(w, r)
		})
	}
}
