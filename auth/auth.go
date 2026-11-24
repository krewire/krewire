// Package auth provides Krewire authentication primitives: Basic and JWT
// middleware, identity propagation through context, and structured HTTP errors.
//
// The middleware is framework-agnostic and works with plain net/http handlers.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// Identity is the authenticated caller.
type Identity struct {
	Subject string
	Method  string
	Roles   []string
	Claims  map[string]any
}

// HasRole reports whether the identity carries the role.
func (id *Identity) HasRole(role string) bool {
	for _, r := range id.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type identityCtxKey struct{}

// IdentityFrom returns the request identity, nil when anonymous.
func IdentityFrom(ctx context.Context) *Identity {
	id, _ := ctx.Value(identityCtxKey{}).(*Identity)
	return id
}

func withIdentity(ctx context.Context, id *Identity) context.Context {
	return WithIdentity(ctx, id)
}

// WithIdentity stores identity in context.
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// Unauthorized returns a 401 HTTPError.
func Unauthorized(message string) *HTTPError {
	return &HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

// Forbidden returns a 403 HTTPError.
func Forbidden(message string) *HTTPError {
	return &HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

// authParam splits an Authorization header into scheme and parameter.
func authParam(h string) (scheme, param string, ok bool) {
	if h == "" {
		return "", "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
