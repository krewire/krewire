# Security Best Practices & Hardening Guide

Comprehensive security standards and defensive practices enforced across the Krewire ecosystem per **Pillar 1 (Secure / S1)**.

---

## 1. Defensive HTTP Defaults

Every application built using `pkg/web` should activate the security middleware suite (`pkg/sec`):

- **OWASP Recommended Headers**: Set `X-Frame-Options: SAMEORIGIN` (or `DENY`), `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, and restrictive `Content-Security-Policy`. `web.NewApp()` enables these headers by default.
- **Strict Transport Security (HSTS)**: Ensure TLS enforcement in production.
- **CSRF Token Validation**: Enforce token validation on state-mutating requests (`POST`, `PUT`, `DELETE`).
- **Strict CORS Validation**: Restrict `Access-Control-Allow-Origin` to verified domains only; avoid wildcard `*` with credentials.

---

## 2. SSRF (Server-Side Request Forgery) Prevention

When performing outbound HTTP requests to user-supplied URLs:
- Use `sec.NewSafeHTTPClient(opts...)` to automatically validate all resolved IP addresses before dialing. This mitigates DNS rebinding (TOCTOU) and blocks:
  - IPv4/IPv6 loopback (`127.0.0.0/8`, `::1`)
  - Private networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)
  - Cloud instance metadata endpoints (`169.254.169.254`, `metadata.google.internal`)
  - Carrier-grade NAT (`100.64.0.0/10`) and link-local ranges
- When host allowlists are known, enforce `sec.WithAllowedHosts("api.example.com")`.

---

## 3. Path Traversal Defense (`SafePath` / CWE-22)

- Never concatenate raw user input with filesystem paths using raw `filepath.Join` or `os.ReadFile`.
- Always wrap path resolution with `sec.SafePath(baseDir, untrustedPath)`:
  - Canonicalizes and sanitizes relative path segments (`.` and `..`).
  - Normalizes Windows backslashes (`\`) on POSIX systems.
  - Rejects null bytes (`\x00`).
  - Halts execution and returns `sec.ErrPathTraversal` if the target path attempts to escape `baseDir`.

```go
target, err := sec.SafePath(workspaceDir, userInput)
if err != nil {
    return fmt.Errorf("security error: path traversal blocked: %w", err)
}
data, err := os.ReadFile(target)
```

---

## 4. Password Hashing & Key Derivation (PBKDF2)

- Never store plain-text passwords or use broken hashing algorithms (MD5, SHA1).
- Use `sec.HashPasswordPBKDF2(password, iter)` to hash passwords using PBKDF2-HMAC-SHA256:
  - Implemented entirely with Go standard library crypto primitives (`crypto/hmac`, `crypto/sha256`, `crypto/rand`).
  - Generates a cryptographically secure 16-byte random salt.
  - Uses 100,000 iterations by default (OWASP recommended minimum).
- Verify credentials using `sec.VerifyPasswordPBKDF2(password, encodedHash)` which evaluates hashes in constant time.

---

## 5. Timing Attack Mitigation

- Always compare sensitive cryptographic tokens, webhook signatures, and secrets using constant-time comparison:
  ```go
  if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
      return errors.New("unauthorized: signature mismatch")
  }
  ```
- Standard string equality (`==`) leaks timing information through byte-by-byte early exits, exposing endpoints to side-channel recovery.

---

## 6. PII Masking & Zero Committed Secrets

- **Zero Hardcoded Secrets**: Never commit API keys, passwords, or tokens in source code or git history. Reference all secrets dynamically via `${env:SECRET_NAME}`.
- **PII Masking**: Wrap structured slog handlers with `sec.NewPIIHandler` to automatically redact credit card numbers, passwords, JWT tokens, and API keys from terminal and telemetry outputs.
- Run static security audits before deployment:
  ```bash
  kiw vet --sec
  ```
