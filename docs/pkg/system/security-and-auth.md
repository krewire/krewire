# Package: `sec` & `auth` (Security & Authentication Primitives)

The `packages/sec` and `packages/auth` libraries deliver hardened security defaults, defensive middleware, authentication tokens, and identity management.

---

## 1. Security Defaults (`packages/sec`)

- **Security Headers Middleware**: Enforces OWASP-recommended HTTP response headers (`X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Content-Security-Policy`, `Referrer-Policy`, `HSTS`).
- **CSRF Defense**: Double-submit cookie pattern and encrypted CSRF token verification.
- **CORS Policies**: Strict whitelist origin matching with preflight caching.
- **SSRF Validation**: URL parser verifying outbound HTTP targets against internal network ranges (`10.0.0.0/8`, `127.0.0.0/8`, `192.168.0.0/16`, AWS metadata `169.254.169.254`).
- **PII Masking**: Regular expression redaction for passwords, API keys, JWT tokens, and sensitive strings in log outputs.

---

## 2. Authentication Primitives (`packages/auth`)

- **JWT Claims & Verification**: Type-safe HMAC-SHA256 and RSA signature verification.
- **Identity & Context Binding**: Attaches verified identity claims to `context.Context`.
- **API Key Authentication**: Timing-safe constant-comparison validation of API keys.
