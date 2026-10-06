package sec

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Check is a health check function. Return nil if healthy, or an error if unhealthy.
type Check func(ctx context.Context) error

// DefaultLivenessPath and DefaultReadinessPath are the probe endpoints served
// when HealthOptions leaves them unset.
const (
	DefaultLivenessPath  = "/healthz"
	DefaultReadinessPath = "/readyz"
)

// HealthOptions configures health and readiness probe endpoints.
type HealthOptions struct {
	// LivenessPath is the URL path for liveness probes. Defaults to "/healthz".
	LivenessPath string
	// ReadinessPath is the URL path for readiness probes. Defaults to "/readyz".
	ReadinessPath string
	// Checks is a map of named dependency checks executed on readiness probes.
	Checks map[string]Check
}

// WithLivenessPath sets the liveness endpoint path.
func WithLivenessPath(path string) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.LivenessPath = path
	}
}

// WithReadinessPath sets the readiness endpoint path.
func WithReadinessPath(path string) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.ReadinessPath = path
	}
}

// WithCheck registers a named health check on readiness probes.
func WithCheck(name string, check Check) func(*HealthOptions) {
	return func(o *HealthOptions) {
		if o.Checks == nil {
			o.Checks = make(map[string]Check)
		}
		o.Checks[name] = check
	}
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Health returns a framework-agnostic HTTP middleware providing /healthz and /readyz probes.
func Health(opts ...func(*HealthOptions)) Middleware {
	o := &HealthOptions{
		LivenessPath:  DefaultLivenessPath,
		ReadinessPath: DefaultReadinessPath,
		Checks:        make(map[string]Check),
	}
	for _, f := range opts {
		f(o)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == o.LivenessPath {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
				return
			}

			if r.Method == http.MethodGet && r.URL.Path == o.ReadinessPath {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if len(o.Checks) == 0 {
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
					return
				}

				var wg sync.WaitGroup
				var mu sync.Mutex
				results := make(map[string]string)
				allHealthy := true

				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()

				for name, check := range o.Checks {
					wg.Add(1)
					go func(n string, ch Check) {
						defer wg.Done()
						err := ch(ctx)
						mu.Lock()
						defer mu.Unlock()
						if err != nil {
							results[n] = err.Error()
							allHealthy = false
						} else {
							results[n] = "ok"
						}
					}(name, check)
				}
				wg.Wait()

				status := http.StatusOK
				statusText := "ok"
				if !allHealthy {
					status = http.StatusServiceUnavailable
					statusText = "degraded"
				}

				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(healthResponse{Status: statusText, Checks: results})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
