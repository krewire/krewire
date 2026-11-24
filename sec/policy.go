package sec

import (
	"net/http"
)

// Policy is a before-gate: return an error to reject the request.
type Policy func(r *http.Request) error

// Require returns middleware running policies in order.
func Require(policies ...Policy) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range policies {
				if err := p(r); err != nil {
					Error(w, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Authenticated requires any identity (401 otherwise).
func Authenticated() Policy {
	return func(r *http.Request) error {
		if IdentityFrom(r.Context()) == nil {
			return Unauthorized("authentication required")
		}
		return nil
	}
}

// WithRoles requires the identity to carry at least one of roles.
func WithRoles(roles ...string) Policy {
	return func(r *http.Request) error {
		id := IdentityFrom(r.Context())
		if id == nil {
			return Unauthorized("authentication required")
		}
		for _, want := range roles {
			if id.HasRole(want) {
				return nil
			}
		}
		return Forbidden("insufficient role")
	}
}

// PolicySet names policies once and references them declaratively.
type PolicySet map[string]Policy

// Require resolves the named policies in order. Unknown names panic.
func (ps PolicySet) Require(names ...string) Middleware {
	policies := make([]Policy, len(names))
	for i, n := range names {
		p, ok := ps[n]
		if !ok {
			panic("sec: unknown policy " + n)
		}
		policies[i] = p
	}
	return Require(policies...)
}
