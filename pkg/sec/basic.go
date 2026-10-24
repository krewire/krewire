// Package sec — BasicAuth re-export.
//
// Deprecated: use github.com/krewire/krewire/pkg/auth.BasicAuth.
package sec

import "github.com/krewire/krewire/pkg/auth"

// BasicVerifier validates an identifier/password pair.
//
// Deprecated: use auth.BasicVerifier.
type BasicVerifier = auth.BasicVerifier

var (
	BasicAuth = auth.BasicAuth
)
