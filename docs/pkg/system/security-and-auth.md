# Package: `sec` & `auth` (Security & Authentication Primitives)

The `pkg/sec` and `pkg/auth` libraries deliver hardened security defaults, defensive middleware, authentication tokens, identity management, and cryptographic protection per **Pillar 1 (Secure / S1: Safety by Default)**.

---

## 1. Architectural Overview & Responsibility Boundaries

```text
pkg/
├── sec/            # Defensive middleware, cryptographic hashing, SSRF guards & path traversal sandbox
│   ├── safepath.go # SafePath directory traversal guard (CWE-22)
│   ├── crypto.go   # PBKDF2-HMAC-SHA256 password hashing & timing-safe verification
│   ├── ssrf.go     # SafeHTTPClient outbound transport blocking private IPs & metadata (CWE-918)
│   ├── headers.go  # OWASP baseline security response headers (CSP, HSTS, X-Frame-Options)
│   ├── csrf.go     # Double-submit cookie & cryptographically signed CSRF validation
│   ├── cors.go     # Strict origin whitelist and preflight handling
│   ├── pii.go      # Structured slog handler redacting passwords, API keys & JWTs
│   └── rate.go     # Token-bucket rate limiter per IP / client token
└── auth/           # Identity binding, JWT tokens & HTTP authentication middleware
    ├── jwt.go      # HS256 JWT creation, claims signing & signature verification
    ├── basic.go    # Timing-safe HTTP Basic authentication middleware (RFC 7617)
    ├── auth.go     # User identity models, claims & context propagation
    └── middleware.go# HTTP authentication middleware chain
```

---

## 2. Path-Traversal Defense (`pkg/sec.SafePath`)

Direct filesystem operations with user input risk Directory Traversal vulnerabilities (CWE-22). `SafePath` enforces strict boundary sandboxing:

### Implementation Contract

- Rejects paths containing null bytes (`\x00`).
- Normalizes backslashes to standard forward slashes.
- Canonicalizes relative sequences (`..`, `.`).
- Verifies that the resolved path is strictly within the allowed root directory.

### Usage Example

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/sec"
)

func HandleUploadDownload(untrustedFilename string) ([]byte, error) {
    baseDir := "/var/data/uploads"
    
    // Safely resolves path or returns sec.ErrPathTraversal
    safeFullPath, err := sec.SafePath(baseDir, untrustedFilename)
    if err != nil {
        return nil, fmt.Errorf("security violation: %w", err)
    }

    // Now safe to read from disk
    return os.ReadFile(safeFullPath)
}
```

---

## 3. Cryptographic Password Hashing (`pkg/sec`)

Uses pure-Go standard library `crypto/sha256` and `crypto/subtle` to provide PBKDF2-HMAC-SHA256 hashing without external CGO dependencies:

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/sec"
)

func main() {
    password := "CorrectHorseBatteryStaple!"

    // 1. Hash password with cryptographically secure 16-byte salt
    // Iterations: 100,000 (OWASP recommended standard)
    hash, err := sec.HashPasswordPBKDF2(password)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Stored Hash: %s\n", hash)

    // 2. Verify password with constant-time equality check
    valid := sec.VerifyPasswordPBKDF2(password, hash)
    fmt.Printf("Password Valid: %t\n", valid) // true
}
```

---

## 4. SSRF Defense (`pkg/sec.SafeHTTPClient`)

Outbound HTTP requests made on behalf of users (webhooks, proxying, URL unfurling) are vulnerable to Server-Side Request Forgery (SSRF / CWE-918). `SafeHTTPClient` enforces network boundaries at the socket connection phase:

### Enforced Protections

- **Blocks Private IP Ranges (RFC 1918)**: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`.
- **Blocks Loopback Addresses**: `127.0.0.0/8`, `::1`.
- **Blocks Link-Local & Cloud Metadata**: `169.254.0.0/16`, specifically AWS/GCP/Azure instance metadata service (`169.254.169.254`).
- **Blocks Carrier-Grade NAT (RFC 6598)**: `100.64.0.0/10`.
- **Prevents DNS Rebinding**: Validates resolved IP addresses immediately prior to dial.

```go
package main

import (
    "context"
    "time"
    "github.com/krewire/krewire/pkg/sec"
)

func FetchExternalWebhook(ctx context.Context, targetURL string) error {
    client := sec.NewSafeHTTPClient(sec.SafeClientConfig{
        Timeout: 5 * time.Second,
        AllowPrivateIPs: false, // Default: false
    })

    resp, err := client.Get(targetURL)
    if err != nil {
        // Returns error if targetURL resolves to 127.0.0.1 or 169.254.169.254
        return err
    }
    defer resp.Body.Close()
    return nil
}
```

---

## 5. Security Headers & Defensive Middleware (`pkg/sec`)

Applies industry-standard defensive HTTP response headers:

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/sec"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello Secure World"))
    })

    // Wraps handler with OWASP defense headers
    handler := sec.SecurityHeadersMiddleware(sec.SecurityHeadersConfig{
        ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';",
        FrameOptions:          "DENY",
        ContentTypeOptions:    "nosniff",
        ReferrerPolicy:        "strict-origin-when-cross-origin",
        HSTSMaxAgeSeconds:     31536000, // 1 year
    })(mux)

    http.ListenAndServe(":8080", handler)
}
```

---

## 6. PII & Secret Redaction (`pkg/sec.NewPIIHandler`)

Wraps any Go standard library `log/slog.Handler` to automatically redact sensitive fields (API keys, authorization tokens, passwords, and credit card patterns) from application logs:

```go
package main

import (
    "log/slog"
    "os"
    "github.com/krewire/krewire/pkg/sec"
)

func main() {
    baseHandler := slog.NewJSONHandler(os.Stdout, nil)
    secureHandler := sec.NewPIIHandler(baseHandler)
    logger := slog.New(secureHandler)

    // Automatically prints: {"level":"INFO","msg":"User Login","token":"[REDACTED]"}
    logger.Info("User Login", "token", "Bearer secret-token-value-12345")
}
```

---

## 7. Authentication Primitives (`pkg/auth`)

The `pkg/auth` package provides token generation, verification, and context identity binding.

### Type-Safe JWT Sign & Verify

```go
package main

import (
    "fmt"
    "time"
    "github.com/krewire/krewire/pkg/auth"
)

func main() {
    secret := []byte("super-secret-key-at-least-32-bytes-long")

    // 1. Generate JWT Claims
    claims := auth.Claims{
        Subject:   "usr_12345",
        Email:     "developer@example.com",
        Roles:     []string{"admin", "developer"},
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    }

    tokenString, err := auth.SignJWT(claims, secret)
    if err != nil {
        panic(err)
    }

    // 2. Parse and Verify JWT
    verifiedClaims, err := auth.VerifyJWT(tokenString, secret)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Authenticated User: %s (%s)\n", verifiedClaims.Email, verifiedClaims.Subject)
}
```

### Context Identity Binding & Middleware

```go
package main

import (
    "net/http"
    "github.com/krewire/krewire/pkg/auth"
)

func ProtectedHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    w.Write([]byte("Welcome, " + user.Email))
}

func main() {
    secret := []byte("super-secret-key-at-least-32-bytes-long")
    jwtMiddleware := auth.JWTMiddleware(secret)

    http.Handle("/api/me", jwtMiddleware(http.HandlerFunc(ProtectedHandler)))
    http.ListenAndServe(":8080", nil)
}
```

---

## 8. Alignment with Pillar 1 (Secure / S1)

- **Zero Hardcoded Secrets**: Secret tokens and signing keys are resolved dynamically from the environment (`${env:SECRET_KEY}`).
- **Constant-Time Verification**: All token and password checks use `crypto/subtle.ConstantTimeCompare` to eliminate timing side-channel attacks.
- **Stdlib-First & Pure-Go**: Uses standard Go `crypto/*` packages with zero third-party dependencies and zero CGO.
