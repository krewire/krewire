# Package: `sec` & `auth` (Security & Authentication Primitives)

The `pkg/sec` and `pkg/auth` libraries deliver hardened security defaults, defensive middleware, authentication tokens, identity management, and cryptographic protection per **Pillar 1 (Secure / S1)**.

---

## 1. Security Defaults (`pkg/sec`)

- **Path-Traversal Sandboxing (`SafePath`)**: Validates untrusted paths within a designated root, blocking directory traversal attacks (CWE-22) via `..`, Windows backslashes, and null bytes (`\x00`).
- **PBKDF2 Password Hashing**: Pure-Go PBKDF2-HMAC-SHA256 (`HashPasswordPBKDF2`, `VerifyPasswordPBKDF2`) with 16-byte cryptographic salts and constant-time verification (`crypto/subtle.ConstantTimeCompare`).
- **SSRF Defense & Outbound Transport (`SafeHTTPClient`)**: Restricts outbound HTTP requests by validating IP ranges (blocking loopback, private RFC 1918, link-local, and AWS/GCP cloud metadata `169.254.169.254`) and preventing DNS rebinding.
- **Security Headers Middleware**: Enforces OWASP-recommended HTTP response headers (`X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Content-Security-Policy`, `Referrer-Policy`, `HSTS`).
- **CSRF Defense**: Double-submit cookie pattern and encrypted CSRF token verification.
- **CORS Policies**: Strict whitelist origin matching with preflight caching.
- **PII Masking**: Regular expression redaction for passwords, API keys, JWT tokens, and sensitive strings in log outputs (`NewPIIHandler`).

---

## 2. Authentication Primitives (`pkg/auth`)

- **JWT Claims & Verification**: Type-safe HMAC-SHA256 and RSA signature verification.
- **Identity & Context Binding**: Attaches verified identity claims to `context.Context`.
- **API Key & Bearer Token Authentication**: Timing-safe constant-comparison validation of API keys and bearer tokens.
- **Basic Authentication**: RFC 7617 HTTP Basic authentication middleware.
