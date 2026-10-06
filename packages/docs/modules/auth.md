# `auth`

Import: `github.com/krewire/libs/auth`

## Purpose

Framework-agnostic HTTP authentication primitives for `net/http`: Basic Authentication, HS256 JWT, caller identity, claims, and structured HTTP errors.

## Main API

- `BasicAuth(realm, verify)`
- `JWTAuth(secret, options...)`
- `SignJWT(secret, claims)` and `ParseJWT(secret, token, options...)`
- `DefaultClaims`, `Claims`, `ClaimCheck`
- `IdentityFrom`, `WithIdentity`
- `Unauthorized`, `Forbidden`, `Error`
- `WithJWTLeeway`, `WithJWTMinSecretLength`, `WithJWTRequireExp`

JWT secrets require at least `RecommendedMinSecretLength` (32 bytes) by default. Secrets must come from runtime configuration.

## Example

```go
handler := auth.JWTAuth(secret, auth.WithJWTRequireExp(true))(protectedHandler)
```

## Design boundary

Authentication middleware does not replace authorization. Add explicit policy checks with `sec.Require` where role or identity authorization is required.
