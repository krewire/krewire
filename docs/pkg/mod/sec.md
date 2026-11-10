# `sec`

Import: `github.com/krewire/krewire/pkg/sec`

## Purpose

The `pkg/sec` package delivers hardened security defaults, defensive HTTP middleware, cryptographic hashing, SSRF prevention, path-traversal sandboxing, and PII masking. Built strictly on Go standard library primitives (Pillar S1: Secure), it eliminates external supply-chain dependencies while offering OWASP-compliant protection out-of-the-box.

---

## Main API

### 1. Path-Traversal Sandboxing (`SafePath`)
- `SafePath(baseDir, untrustedPath string) (string, error)` — Canonicalizes and sanitizes user-supplied paths, preventing directory traversal escapes (CWE-22) outside the designated root.
- `ErrPathTraversal` — Returned when the resolved path attempts to escape the root boundary.
- `ErrInvalidPath` — Returned when the root or target path is empty.

### 2. PBKDF2 Password Hashing & Key Derivation
- `HashPasswordPBKDF2(password string, iter int) (string, error)` — Hashes a password using PBKDF2-HMAC-SHA256 with a cryptographically secure 16-byte random salt. Format: `pbkdf2:sha256:<iter>:<hex-salt>:<hex-key>`.
- `VerifyPasswordPBKDF2(password, encodedHash string) bool` — Constant-time password verification using `crypto/subtle.ConstantTimeCompare` against timing attacks.
- `DeriveKeyPBKDF2(password, salt []byte, iter, keyLen int) []byte` — Pure-Go PBKDF2 key derivation per RFC 2898 / RFC 8018.
- `DefaultPBKDF2Iterations` (100,000), `DefaultSaltBytes` (16), `DefaultKeyBytes` (32).

### 3. SSRF & Outbound HTTP Protection
- `ValidateOutboundURL(raw string, allowedHosts ...string) error` — Validates URL schemes (`http`, `https`) and blocks private/link-local/cloud-metadata addresses without making a network request.
- `NewSafeTransport(opts ...SafeTransportOption) *http.Transport` — Returns an `http.RoundTripper` that resolves hostnames and validates all resolved IPs before dialing, preventing DNS rebinding attacks.
- `NewSafeHTTPClient(opts ...SafeTransportOption) *http.Client` — Ready-to-use HTTP client backed by `NewSafeTransport`.
- `WithAllowedHosts(hosts ...string)`, `WithDialTimeout(d time.Duration)` — Functional options for configuring outbound rules.

### 4. HTTP Middleware & Defense-in-Depth
- `SecurityHeaders()` — OWASP-recommended HTTP headers (`X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, CSP).
- `CORS(opts ...CORSOption)` — Strict origin whitelist matching with preflight options.
- `CSRF(opts ...CSRFOption)` — Encrypted double-submit token protection.
- `Require(policy)`, `Authenticated()`, `WithRoles(roles ...string)` — Declarative authorization filters.

### 5. Diagnostics & PII Sanitization
- `MaskPII(input string) string` — Redacts sensitive tokens, credit cards, emails, and passwords.
- `NewPIIHandler(h slog.Handler) *PIIHandler` — Structured logging middleware with automatic PII masking.

---

## Examples

### Path Traversal Defense with `SafePath`
```go
package main

import (
	"fmt"
	"os"

	"github.com/krewire/krewire/pkg/sec"
)

func ReadUserFile(workspace, userFilename string) ([]byte, error) {
	// Prevents attacks like "../../etc/passwd" or "..\\windows\\system32"
	target, err := sec.SafePath(workspace, userFilename)
	if err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}
	return os.ReadFile(target)
}
```

### PBKDF2 Password Hashing
```go
package main

import (
	"fmt"

	"github.com/krewire/krewire/pkg/sec"
)

func main() {
	password := "Correct-Horse-Battery-Staple"

	// Hash password with 100,000 PBKDF2-HMAC-SHA256 iterations
	hash, err := sec.HashPasswordPBKDF2(password, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Stored Hash: %s\n", hash)

	// Constant-time verification
	isValid := sec.VerifyPasswordPBKDF2(password, hash)
	fmt.Printf("Password Valid: %t\n", isValid)
}
```

### SSRF Protection with `SafeHTTPClient`
```go
package main

import (
	"context"
	"net/http"

	"github.com/krewire/krewire/pkg/sec"
)

func main() {
	// Blocks localhost, 127.0.0.1, 10.0.0.0/8, 169.254.169.254, and DNS rebinding
	client := sec.NewSafeHTTPClient(
		sec.WithAllowedHosts("api.github.com", "api.stripe.com"),
	)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/zen", nil)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}
```

---

## Architecture Boundaries

- **Stdlib-First Foundation**: Implemented entirely via `crypto/hmac`, `crypto/sha256`, `crypto/subtle`, `crypto/rand`, and `path/filepath`.
- **Zero Lateral Peer Imports**: Decoupled from application engines at higher layers.
