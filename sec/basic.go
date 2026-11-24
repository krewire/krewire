// Package sec — BasicAuth re-export.
//
// Deprecated: use github.com/krewire/krewire/auth.BasicAuth.
package sec

import "github.com/krewire/krewire/auth"

// BasicVerifier validates an identifier/password pair.
//
// Deprecated: use auth.BasicVerifier.
type BasicVerifier = auth.BasicVerifier

var (
	BasicAuth = auth.BasicAuth
)
