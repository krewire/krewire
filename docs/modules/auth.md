# Package Reference: `auth`

Import: `github.com/krewire/krewire/auth`

## 1. Purpose & Architecture

The `auth` package delivers core authentication primitives, type-safe JWT creation and verification (HS256), timing-safe HTTP Basic Authentication (RFC 7617), and context identity binding for HTTP workloads.

```text
auth/
├── jwt.go          # HS256 JWT creation, claims signing & verification
├── basic.go        # HTTP Basic authentication middleware
├── auth.go         # User identity models, claims & context propagation
└── middleware.go   # Authentication middleware chain
```

For full details on defensive security, SSRF guards, and password hashing, see [Security & Authentication Guide](./system/security-and-auth.md).

---

## 2. Main API Reference

- **`auth.SignJWT(claims Claims, secret []byte) (string, error)`**: Creates and signs a JSON Web Token using HMAC-SHA256.
- **`auth.VerifyJWT(tokenString string, secret []byte) (*Claims, error)`**: Validates signature, expiration (`exp`), and structure using constant-time comparison.
- **`auth.BasicAuthMiddleware(validator func(user, pass string) bool) func(http.Handler) http.Handler`**: Enforces RFC 7617 Basic Authentication.
- **`auth.JWTMiddleware(secret []byte) func(http.Handler) http.Handler`**: Extracts and validates bearer tokens from `Authorization: Bearer <token>` or cookies, injecting verified claims into `r.Context()`.
- **`auth.UserFromContext(ctx context.Context) (*Claims, bool)`**: Retrieves verified identity claims from request context.

---

## 3. Usage Example

```go
package main

import (
    "net/http"
    "time"
    "github.com/krewire/krewire/auth"
)

func main() {
    secret := []byte("your-cryptographic-secret-at-least-32-bytes")

    // Generate token
    token, _ := auth.SignJWT(auth.Claims{
        Subject:   "usr_42",
        Email:     "admin@example.com",
        Roles:     []string{"admin"},
        ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
    }, secret)

    // Protect handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        user, _ := auth.UserFromContext(r.Context())
        w.Write([]byte("Authorized: " + user.Email))
    })

    protected := auth.JWTMiddleware(secret)(mux)
    _ = token
    _ = protected
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Timing-safe token verification; no tokens committed in source code; secrets sourced via environment variables (`${env:AUTH_SECRET}`).
- **Sustainable (S2)**: Pure Go without heavyweight third-party JWT dependencies; lean RAM footprint.
- **Scalable (S3)**: Fully compatible with enterprise identity federation (OIDC / public JWKS).
