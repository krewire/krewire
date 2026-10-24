# Security Best Practices & Hardening Guide

Comprehensive security standards and defensive practices enforced across the Krewire ecosystem.

---

## 1. Defensive HTTP Defaults

Every application built using `pkg/web` should activate the security middleware suite (`pkg/sec`):

- **OWASP Recommended Headers**: Set `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and restrictive `Content-Security-Policy`.
- **Strict Transport Security (HSTS)**: Ensure TLS enforcement in production.
- **CSRF Token Validation**: Enforce token validation on state-mutating requests (`POST`, `PUT`, `DELETE`).

---

## 2. SSRF (Server-Side Request Forgery) Prevention

When performing outbound HTTP requests to user-supplied URLs:
- Use `sec.ValidateOutboundURL(target, allowedHosts...)` to reject literal private, loopback, link-local, and unspecified IP addresses and to require the host to be explicitly allowlisted. It performs no network request, so resolve-and-check DNS yourself when the allowlist is not enough.
- Block cloud instance metadata endpoints (`169.254.169.254`).
- Prevent IPv6 loopback and link-local addresses (`::1`, `fe80::/10`).

---

## 3. PII Masking & Secrets Sanitization

- Never log unencrypted passwords, API tokens, JWTs, or session cookies.
- Use `pkg/log` structured loggers configured with automatic PII masking.
- Run static security audits before deployment:
  ```bash
  kiw vet --sec
  ```

---

## 4. Path Traversal Defense (CWE-22)

- Never concatenate raw user input with filesystem paths.
- Enforce `filepath.Clean` and verify that the target path does not escape the intended root directory.
