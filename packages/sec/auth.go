// Package sec provides Krewire Security — browser hardening, CSRF, and
// policy. Authentication primitives have moved to github.com/krewire/libs/auth
// (Krewire Auth); this file re-exports auth for backward compatibility.
package sec

import "github.com/krewire/krewire/packages/auth"

// Identity is the authenticated caller.
//
// Deprecated: use auth.Identity.
type Identity = auth.Identity

var (
	// Deprecated: use auth.IdentityFrom.
	IdentityFrom = auth.IdentityFrom
	Unauthorized = auth.Unauthorized
	Forbidden    = auth.Forbidden
)
