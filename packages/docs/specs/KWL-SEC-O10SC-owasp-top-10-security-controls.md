# Specification — OWASP Top 10 Security Controls

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWL-O10SC                                   |
| Title       | OWASP Top 10 Security Controls              |
| Status      | Draft                                       |
| Date        | 2026-09-28                                  |
| Author      | Krewire Contributors                         |
| Domain      | Libraries — Security                        |

## 1. Context

Web applications and services in the Krewire ecosystem must defend against common web application security vulnerabilities as categorized by the OWASP Top 10 (2021). The `libs/sec` module (alongside `libs/auth`) provides HTTP middleware, policy evaluation, headers, CSRF protection, and authentication mechanisms to serve as the standard security layer for Krewire framework components.

## 2. Problem Statement

Without a centralized, standardized security specification and baseline implementation, individual applications across the ecosystem risk implementing ad-hoc or incomplete security controls. This leads to common misconfigurations, inconsistent authentication/authorization checks, weak cryptographic parameters, missing HTTP security headers, and susceptibility to injection or CSRF attacks.

## 3. Goals

- G1 — Align `libs/sec` (and `libs/auth`) middleware and primitives directly with the OWASP Top 10 (2021) categories.
- G2 — Provide secure-by-default HTTP security headers, CORS policies, and CSRF protection.
- G3 — Enforce robust identity, role-based authorization policies, and constant-time cryptographic checks.
- G4 — Guarantee zero external third-party dependencies in security packages (`sec` and `auth`), relying solely on the Go standard library.
- G5 — Ensure complete specification-to-test traceability for all OWASP Top 10 controls.

## 4. Non-Goals

- NG1 — Replacing dedicated edge reverse proxies or Web Application Firewalls (WAF).
- NG2 — Storing user credentials or managing database-backed user directories.
- NG3 — Automated OAuth2/OIDC discovery or complex multi-tenant identity provider hosting.

## 5. Requirements

### 5.1 OWASP Top 10 Mapping & Controls

| ID          | OWASP Category | Requirement Description | Priority |
| ----------- | -------------- | ----------------------- | -------- |
| SEC-A01-001 | A01:2021 - Broken Access Control | `sec.Require` and `PolicySet` must enforce deny-by-default authorization, verifying `Identity` and roles (`WithRoles`) from request context. | Must |
| SEC-A01-002 | A01:2021 - Broken Access Control | `IdentityFrom` and `WithIdentity` must safely isolate and propagate authenticated caller state across the HTTP context pipeline. | Must |
| SEC-A02-001 | A02:2021 - Cryptographic Failures | JWT HS256 operations must enforce a minimum secret length (default 32 bytes via `WithMinSecretLength`). | Must |
| SEC-A02-002 | A02:2021 - Cryptographic Failures | Secret comparisons, Basic Auth verifications, and HMAC validations must use constant-time comparisons (`subtleCompare`, `constantTimeEqual`). | Must |
| SEC-A03-001 | A03:2021 - Injection | `BasicAuth` realm values must be sanitized to strip CRLF characters (`\r`, `\n`) to prevent HTTP header injection. | Must |
| SEC-A03-002 | A03:2021 - Injection | `StripTags` must sanitize untrusted string inputs by stripping HTML elements and script tags. | Must |
| SEC-A04-001 | A04:2021 - Insecure Design | `sec.Health` must decouple liveness (`/healthz`) from readiness (`/readyz`) probes with configurable check callbacks. | Must |
| SEC-A05-001 | A05:2021 - Security Misconfiguration | `sec.SecurityHeaders` must apply default security headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, HSTS, and Content Security Policy. | Must |
| SEC-A05-002 | A05:2021 - Security Misconfiguration | `sec.CORS` must validate request origins explicitly and prevent unrestricted wildcard usage when credentials are allowed. | Must |
| SEC-A06-001 | A06:2021 - Vulnerable Components | `libs/auth` uses only the Go standard library. `libs/sec` may use the repository's pinned `golang.org/x/net/html` tokenizer for safe HTML parsing; dependencies must remain pinned and vulnerability-scanned. | Must |
| SEC-A07-001 | A07:2021 - Identification & Auth | `JWTAuth` and `ParseJWT` must validate claims (`exp`, `nbf`, `iat`) with configurable clock skew leeway (`WithLeeway`). | Must |
| SEC-A08-001 | A08:2021 - Software/Data Integrity | `sec.CSRF` must implement double-submit cookie protection with HMAC-SHA256 signature verification for state-changing HTTP methods. | Must |
| SEC-A09-001 | A09:2021 - Security Logging | Security failures must return structured `HTTPError` instances (`401`, `403`) without leaking secrets or credentials in logs. | Must |
| SEC-A10-001 | A10:2021 - Server-Side Request Forgery (SSRF) | `libs/sec` must not make outbound requests from untrusted URL input; callers must validate allowed schemes, hosts, ports, and resolved addresses before network access. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Zero Allocations on Safe Paths.** Header and policy evaluation on safe paths must minimize allocations.
- NFR2 — **Dependency Hygiene.** `auth` remains stdlib-only; `sec` uses only pinned dependencies with vulnerability scanning required.
- NFR3 — **Fail-Closed Default.** Any unauthenticated or unauthorized evaluation must result in immediate `401 Unauthorized` or `403 Forbidden` rejection.
- NFR4 — **Quality & Test Coverage.** All implemented OWASP requirement IDs must be covered by unit tests with `// Tests for SEC-*` traceability tags. The SSRF boundary is verified by integration tests in consuming services.

## 7. Traceability Matrix

| Requirement ID | Implementation Package | Primary Tests |
| -------------- | ---------------------- | ------------- |
| SEC-A01-001 | `sec/policy.go` | `sec/policy_test.go` |
| SEC-A01-002 | `auth/auth.go` | `auth/auth_test.go` |
| SEC-A02-001 | `auth/jwt.go` | `auth/jwt_test.go` |
| SEC-A02-002 | `auth/basic.go`, `sec/csrf.go` | `auth/auth_test.go`, `sec/csrf_test.go` |
| SEC-A03-001 | `auth/basic.go` | `auth/auth_test.go` |
| SEC-A03-002 | `sec/headers.go` | `sec/sec_test.go` |
| SEC-A04-001 | `sec/health.go` | `sec/health_test.go` |
| SEC-A05-001 | `sec/headers.go` | `sec/sec_test.go` |
| SEC-A05-002 | `sec/cors.go` | `sec/cors_test.go` |
| SEC-A06-001 | `go.mod` | `go.mod` |
| SEC-A07-001 | `auth/jwt.go` | `auth/jwt_test.go` |
| SEC-A08-001 | `sec/csrf.go` | `sec/csrf_test.go` |
| SEC-A09-001 | `auth/error.go` | `auth/auth_test.go` |
| SEC-A10-001 | `sec/ssrf.go` | `sec/ssrf_test.go` plus integration tests in consuming services |
