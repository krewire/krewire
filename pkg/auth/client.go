package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// User represents an authenticated identity returned by Krewire Auth.
type User struct {
	ID         string   `json:"sub"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	TenantID   string   `json:"tenant_id"`
	Roles      []string `json:"roles"`
	MFAEnabled bool     `json:"mfa_enabled"`
	SessionID  string   `json:"session_id,omitempty"`
}

// ClientOption configures the Auth Client.
type ClientOption func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithCacheTTL configures the in-memory cache TTL for verified sessions.
// Setting duration <= 0 disables in-memory caching.
func WithCacheTTL(ttl time.Duration) ClientOption {
	return func(c *Client) {
		c.cacheTTL = ttl
	}
}

type cachedSession struct {
	user      *User
	expiresAt time.Time
}

// Client interacts with the centralized Krewire Auth daemon (https://auth.krewire.com).
type Client struct {
	authURL    string
	httpClient *http.Client
	cacheTTL   time.Duration
	cacheMu    sync.RWMutex
	cache      map[string]cachedSession
}

// NewClient returns a new Client pointing to canonical authURL.
func NewClient(authURL string, opts ...ClientOption) *Client {
	cleanURL := strings.TrimRight(authURL, "/")
	if cleanURL == "" {
		cleanURL = "https://auth.krewire.com"
	}
	c := &Client{
		authURL: cleanURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: 30 * time.Second,
		cache:    make(map[string]cachedSession),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// AuthURL returns the base URL of the auth server.
func (c *Client) AuthURL() string {
	return c.authURL
}

// LoginURL returns the redirection URL for user login, preserving returnTo.
func (c *Client) LoginURL(returnTo string) string {
	if returnTo == "" {
		return c.authURL + "/login"
	}
	return fmt.Sprintf("%s/login?return_to=%s", c.authURL, url.QueryEscape(returnTo))
}

// RegisterURL returns the redirection URL for account registration.
func (c *Client) RegisterURL(returnTo string) string {
	if returnTo == "" {
		return c.authURL + "/register"
	}
	return fmt.Sprintf("%s/register?return_to=%s", c.authURL, url.QueryEscape(returnTo))
}

// LogoutURL returns the redirection URL for ending the session.
func (c *Client) LogoutURL(returnTo string) string {
	if returnTo == "" {
		return c.authURL + "/logout"
	}
	return fmt.Sprintf("%s/logout?return_to=%s", c.authURL, url.QueryEscape(returnTo))
}

// VerifySession validates a session ID against Krewire Auth's /api/userinfo endpoint.
func (c *Client) VerifySession(ctx context.Context, sessionID string) (*User, error) {
	if sessionID == "" {
		return nil, errors.New("empty session token")
	}

	// 1. Check local cache if enabled
	if c.cacheTTL > 0 {
		c.cacheMu.RLock()
		cached, found := c.cache[sessionID]
		c.cacheMu.RUnlock()
		if found && time.Now().Before(cached.expiresAt) {
			return cached.user, nil
		}
	}

	// 2. Query remote Auth server
	reqURL := c.authURL + "/api/userinfo"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.AddCookie(&http.Cookie{
		Name:  "krewire_session",
		Value: sessionID,
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session verification failed: HTTP %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}
	if user.SessionID == "" {
		user.SessionID = sessionID
	}

	// 3. Cache result
	if c.cacheTTL > 0 {
		c.cacheMu.Lock()
		c.cache[sessionID] = cachedSession{
			user:      &user,
			expiresAt: time.Now().Add(c.cacheTTL),
		}
		c.cacheMu.Unlock()
	}

	return &user, nil
}

// InvalidateSession purges the session from local client cache.
func (c *Client) InvalidateSession(sessionID string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	delete(c.cache, sessionID)
}

// Context keys for authenticated user.
type userCtxKey struct{}

// UserFrom returns the authenticated User stored in context, or nil.
func UserFrom(ctx context.Context) *User {
	u, _ := ctx.Value(userCtxKey{}).(*User)
	return u
}

// WithUser associates the authenticated User with context.
func WithUser(ctx context.Context, u *User) context.Context {
	if u == nil {
		return ctx
	}
	// Also propagate to standard auth.Identity for cross-compatibility
	ctx = WithIdentity(ctx, &Identity{
		Subject: u.ID,
		Method:  "session",
		Roles:   u.Roles,
		Claims: map[string]any{
			"email":     u.Email,
			"name":      u.Name,
			"tenant_id": u.TenantID,
		},
	})
	return context.WithValue(ctx, userCtxKey{}, u)
}

// RequireAuth returns a net/http middleware that verifies Krewire Auth sessions.
// For browser requests (Accept: text/html), unauthenticated requests are redirected
// to the centralized login page. For API requests, HTTP 401 Unauthorized is returned.
func RequireAuth(client *Client, cookieName string) Middleware {
	if cookieName == "" {
		cookieName = "krewire_session"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Cookie or Authorization header
			var token string
			if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
				token = c.Value
			} else if authHdr := r.Header.Get("Authorization"); authHdr != "" {
				if scheme, param, ok := authParam(authHdr); ok && strings.EqualFold(scheme, "Bearer") {
					token = param
				}
			}

			if token != "" {
				user, err := client.VerifySession(r.Context(), token)
				if err == nil && user != nil {
					ctx := WithUser(r.Context(), user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Unauthenticated handling
			accept := r.Header.Get("Accept")
			isHTML := strings.Contains(accept, "text/html")

			if isHTML {
				// Reconstruct return_to target URL
				scheme := "https"
				if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" && (r.Host == "localhost" || strings.HasPrefix(r.Host, "127.0.0.1")) {
					scheme = "http"
				}
				target := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
				http.Redirect(w, r, client.LoginURL(target), http.StatusFound)
				return
			}

			Error(w, Unauthorized("authentication required: visit "+client.LoginURL("")))
		})
	}
}
