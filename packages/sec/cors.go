package sec

import (
	"net/http"
	"strconv"
	"strings"
)

// DefaultMaxAgeSeconds is the preflight cache lifetime applied when
// CORSOptions.MaxAge is left unset: 24 hours, expressed in seconds because that
// is the unit the Access-Control-Max-Age header requires.
const DefaultMaxAgeSeconds = 24 * 60 * 60

// DefaultMethods, DefaultAllowedHeaders, and DefaultExposedHeaders are the CORS
// defaults applied when the corresponding option is empty.
var (
	// DefaultMethods is the method set allowed by default.
	DefaultMethods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}

	// DefaultAllowedHeaders is the request-header allowlist used by default.
	DefaultAllowedHeaders = []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Requested-With",
		"X-CSRF-Token",
	}
)

// wildcardOrigin is the CORS origin that matches any origin. It is only valid
// when credentials are disabled; browsers reject "*" together with
// Access-Control-Allow-Credentials.
const wildcardOrigin = "*"

// CORSOptions configures Cross-Origin Resource Sharing.
type CORSOptions struct {
	// AllowOrigins is a list of allowed origins. Defaults to ["*"].
	AllowOrigins []string
	// AllowMethods is a list of allowed HTTP methods. Defaults to GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
	AllowMethods []string
	// AllowHeaders is a list of allowed request headers. Defaults to standard headers.
	AllowHeaders []string
	// ExposeHeaders is a list of response headers exposed to the client.
	ExposeHeaders []string
	// AllowCredentials indicates whether cookies, authorization headers, or TLS client certificates are exposed.
	AllowCredentials bool
	// MaxAge is the duration in seconds that preflight requests can be cached. Defaults to 86400 (24h).
	MaxAge int
}

// WithOrigins appends allowed origins to CORSOptions.
func WithOrigins(origins ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowOrigins = append(o.AllowOrigins, origins...)
	}
}

// WithMethods appends allowed HTTP methods to CORSOptions.
func WithMethods(methods ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowMethods = append(o.AllowMethods, methods...)
	}
}

// WithHeaders appends allowed headers to CORSOptions.
func WithHeaders(headers ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowHeaders = append(o.AllowHeaders, headers...)
	}
}

// WithExposeHeaders appends exposed response headers to CORSOptions.
func WithExposeHeaders(headers ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.ExposeHeaders = append(o.ExposeHeaders, headers...)
	}
}

// WithCredentials sets AllowCredentials in CORSOptions.
func WithCredentials(allow bool) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowCredentials = allow
	}
}

// WithMaxAge sets preflight cache duration in seconds.
func WithMaxAge(seconds int) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.MaxAge = seconds
	}
}

// CORS returns a framework-agnostic HTTP middleware implementing Cross-Origin Resource Sharing.
func CORS(opts ...func(*CORSOptions)) Middleware {
	o := newCORSOptions(opts...)
	p := &corsPolicy{
		methods:        strings.Join(o.AllowMethods, ", "),
		headers:        strings.Join(o.AllowHeaders, ", "),
		expose:         strings.Join(o.ExposeHeaders, ", "),
		credentials:    o.AllowCredentials,
		allowAnyOrigin: !o.AllowCredentials && containsOrigin(o.AllowOrigins, wildcardOrigin),
		allowList:      o.AllowOrigins,
		maxAgeSeconds:  o.MaxAge,
	}
	return p.middleware
}

// newCORSOptions applies the supplied options on top of the defaults.
func newCORSOptions(opts ...func(*CORSOptions)) *CORSOptions {
	o := &CORSOptions{MaxAge: DefaultMaxAgeSeconds}
	for _, f := range opts {
		f(o)
	}
	if len(o.AllowOrigins) == 0 {
		o.AllowOrigins = []string{wildcardOrigin}
	}
	if len(o.AllowMethods) == 0 {
		o.AllowMethods = DefaultMethods
	}
	if len(o.AllowHeaders) == 0 {
		o.AllowHeaders = DefaultAllowedHeaders
	}
	return o
}

// containsOrigin reports whether list holds want.
func containsOrigin(list []string, want string) bool {
	for _, origin := range list {
		if origin == want {
			return true
		}
	}
	return false
}

// corsPolicy holds the precomputed, per-application state derived from
// CORSOptions. Separating it from option parsing keeps CORS() concerned only
// with wiring and leaves the middleware responsible for request handling.
type corsPolicy struct {
	methods     string
	headers     string
	expose      string
	credentials bool
	// allowAnyOrigin is true only when the wildcard origin is configured and
	// credentials are disabled.
	allowAnyOrigin bool
	allowList      []string
	// maxAgeSeconds is the preflight cache lifetime, advertised only when
	// positive.
	maxAgeSeconds int
}

// middleware is the returned Middleware. It resolves the incoming Origin and
// writes the matching CORS response headers.
func (p *corsPolicy) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !p.allowed(w, origin) {
			next.ServeHTTP(w, r)
			return
		}

		if p.credentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if p.expose != "" {
			w.Header().Set("Access-Control-Expose-Headers", p.expose)
		}

		// A preflight is an OPTIONS request asking which method/headers the real
		// request may use; it never reaches the wrapped handler.
		if isPreflight(r) {
			w.Header().Set("Access-Control-Allow-Methods", p.methods)
			w.Header().Set("Access-Control-Allow-Headers", p.headers)
			if p.maxAgeSeconds > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(p.maxAgeSeconds))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allowed writes the Access-Control-Allow-Origin header when origin matches the
// policy, and reports whether the request may proceed with CORS headers set.
//
// An arbitrary origin is never reflected while credentials are enabled: the
// wildcard origin is valid only for non-credentialed CORS.
func (p *corsPolicy) allowed(w http.ResponseWriter, origin string) bool {
	if p.allowAnyOrigin {
		w.Header().Set("Access-Control-Allow-Origin", wildcardOrigin)
		return true
	}
	for _, allowed := range p.allowList {
		if allowed != wildcardOrigin && strings.EqualFold(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			return true
		}
	}
	return false
}

// isPreflight reports whether r is a CORS preflight request.
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}
