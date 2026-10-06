package sec

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

// DefaultCSRFNames are the cookie, header, and form-field names carrying the
// double-submit token when CSRFOptions leaves them unset. The header name is
// matched case-insensitively by net/http, so clients may send any casing.
const (
	DefaultCSRFCookieName = "XSRF-TOKEN"
	DefaultCSRFHeaderName = "X-CSRF-Token"
	DefaultCSRFFieldName  = "csrf_token"
)

// hostCookiePrefix forces Path=/, Secure, and no Domain attribute, which
// confines the cookie to the exact origin and blocks subdomain injection.
const hostCookiePrefix = "__Host-"

// CSRFOptions tunes CSRF protection.
type CSRFOptions struct {
	CookieName          string
	HeaderName          string
	FieldName           string
	Secure              bool
	HTTPOnly            bool
	PrefixHost          bool   // If true, prefixes cookie with __Host- and forces Secure + Path=/
	Secret              []byte // Optional HMAC secret to sign cookie tokens against cookie tossing
	IgnoreAuthorization bool   // If true, requests with Authorization header skip CSRF check
	SameSite            http.SameSite
}

type csrfCtxKey struct{}

// CSRFOption configures CSRFOptions.
type CSRFOption func(*CSRFOptions)

// WithCSRFCookie sets the cookie name for the double-submit token.
func WithCSRFCookie(name string) CSRFOption {
	return func(o *CSRFOptions) { o.CookieName = name }
}

// WithCSRFHeader sets the request header carrying the submitted token.
func WithCSRFHeader(name string) CSRFOption {
	return func(o *CSRFOptions) { o.HeaderName = name }
}

// WithCSRFField sets the form field carrying the submitted token.
func WithCSRFField(name string) CSRFOption {
	return func(o *CSRFOptions) { o.FieldName = name }
}

// WithCSRFSecret enables HMAC signing of the cookie value, which defends
// against cookie tossing from a sibling subdomain or a non-HTTP writer.
func WithCSRFSecret(secret []byte) CSRFOption {
	return func(o *CSRFOptions) { o.Secret = secret }
}

// WithCSRFSecure marks the cookie Secure and forces the __Host- prefix, which
// also requires Path=/ and forbids a Domain attribute.
func WithCSRFSecure() CSRFOption {
	return func(o *CSRFOptions) { o.PrefixHost = true }
}

// WithCSRFSameSite sets the cookie SameSite mode.
func WithCSRFSameSite(mode http.SameSite) CSRFOption {
	return func(o *CSRFOptions) { o.SameSite = mode }
}

// WithCSRFIgnoreAuthorization lets requests carrying an Authorization header
// skip the CSRF check. Only enable it when bearer tokens, and not cookies,
// carry the session.
func WithCSRFIgnoreAuthorization() CSRFOption {
	return func(o *CSRFOptions) { o.IgnoreAuthorization = true }
}

// CSRF returns double-submit cookie middleware.
//
// SECURITY: the token cookie is not HttpOnly by design — the client must read
// it to echo it back in a header — so it is SameSite=Lax and path-scoped to
// limit exposure. Supply a secret with WithCSRFSecret so a value planted by a
// sibling subdomain (cookie tossing) cannot satisfy the check.
func CSRF(opts ...CSRFOption) Middleware {
	o := &CSRFOptions{}
	for _, f := range opts {
		f(o)
	}
	if o.CookieName == "" {
		o.CookieName = DefaultCSRFCookieName
	}
	if o.HeaderName == "" {
		o.HeaderName = DefaultCSRFHeaderName
	}
	if o.FieldName == "" {
		o.FieldName = DefaultCSRFFieldName
	}
	if o.SameSite == 0 {
		// Lax is the strongest value that still allows a plain form post to
		// carry the double-submit token.
		o.SameSite = http.SameSiteLaxMode
	}
	if o.PrefixHost {
		if !strings.HasPrefix(o.CookieName, hostCookiePrefix) {
			o.CookieName = hostCookiePrefix + o.CookieName
		}
		o.Secure = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookieValStr := cookieVal(r, o.CookieName)
			safe := r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions || r.Method == http.MethodTrace

			rawToken := cookieValStr
			if len(o.Secret) > 0 && cookieValStr != "" {
				parts := strings.Split(cookieValStr, ".")
				if len(parts) == 2 && validHMAC(parts[0], parts[1], o.Secret) {
					rawToken = parts[0]
				} else {
					rawToken = ""
				}
			}

			if rawToken == "" && safe {
				rawToken = randomToken()
				cookieValToSet := rawToken
				if len(o.Secret) > 0 {
					cookieValToSet = rawToken + "." + signHMAC(rawToken, o.Secret)
				}
				http.SetCookie(w, &http.Cookie{
					Name:     o.CookieName,
					Value:    cookieValToSet,
					Path:     "/",
					Secure:   o.Secure,
					HttpOnly: o.HTTPOnly,
					SameSite: o.SameSite,
				})
			}

			skip := o.IgnoreAuthorization && r.Header.Get("Authorization") != ""
			if !safe && !skip {
				submitted := r.Header.Get(o.HeaderName)
				if submitted == "" {
					_ = r.ParseForm()
					submitted = r.PostForm.Get(o.FieldName)
				}
				if submitted == "" || rawToken == "" || !constantTimeEqual(submitted, rawToken) {
					Error(w, Forbidden("csrf token mismatch"))
					return
				}
			}

			ctx := context.WithValue(r.Context(), csrfCtxKey{}, rawToken)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFFrom returns the current request's CSRF token.
func CSRFFrom(ctx context.Context) string {
	v, _ := ctx.Value(csrfCtxKey{}).(string)
	return v
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("sec: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func signHMAC(token string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func validHMAC(token, sig string, secret []byte) bool {
	expected := signHMAC(token, secret)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) == 1
}
