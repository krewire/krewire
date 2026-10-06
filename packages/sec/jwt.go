// Package sec — JWT re-export.
//
// Deprecated: use github.com/krewire/libs/auth for JWT.
package sec

import "github.com/krewire/krewire/packages/auth"

// Claims is a JWT claim set.
//
// Deprecated: use auth.Claims.
type Claims = auth.Claims

var (
	ErrInvalidToken = auth.ErrInvalidToken
	SignJWT         = auth.SignJWT
	DefaultClaims   = auth.DefaultClaims
	ParseJWT        = auth.ParseJWT
	JWTAuth         = auth.JWTAuth
)

// JWTOptions tunes the JWTAuth middleware.
//
// Deprecated: use auth.JWTOptions.
type JWTOptions = auth.JWTOptions

// ClaimCheck asserts one claim equality.
//
// Deprecated: use auth.ClaimCheck.
type ClaimCheck = auth.ClaimCheck
