# Security Assurance Baseline & Verification Guide

This document establishes the canonical security assurance baseline, threat models, and verification controls enforced across the Krewire ecosystem per **Pillar 1 (Secure / S1)**:

> *"Safety is the default, not an option or a paid tier: a small attack surface, no secrets in the tree, and behavior anyone can verify."*

---

## 1. Overview & Assurance Posture

Security in Krewire is an end-to-end, reproducible operational discipline rather than an assortment of isolated helper functions. While individual packages provide cryptographic and defensive primitives, application assurance requires verifying that the complete workload is safe within its configuration and deployment context.

### Non-Goals & Boundaries
- **No Claim of Magic Infallibility**: Using Krewire does not automatically make any application immune to flawed business logic or malicious operator configuration.
- **Not a Pen-Test Substitute**: Automated security assurance does not replace professional penetration testing or independent code review.
- **Fail-Closed Default**: Every security boundary fails closed: unauthenticated requests yield `401 Unauthorized`, unauthorized mutations yield `403 Forbidden`, traversal attempts yield `ErrPathTraversal`, and SSRF attempts fail before socket creation.

---

## 2. Threat Models by Canonical Workload

Each canonical starter kind in Krewire (`pkg/model.AllKinds`) operates under a distinct threat model and attack surface:

| Workload Kind | Primary Attack Surface & Threat Vectors | Critical Defenses |
|---|---|---|
| **App** (`app`) | Internet-facing interactive web applications. Primary threats: Cross-Site Scripting (XSS), Cross-Site Request Forgery (CSRF), session hijacking, Server-Side Request Forgery (SSRF), SQL injection, and credential compromise. | Defense HTTP headers, automatic CSRF token validation, PBKDF2 adaptive hashing, constant-time compare, and SafePath guards. |
| **Service** (`service`) | REST/JSON API microservices and backend services. Primary threats: Broken Object-Level Authorization (BOLA), denial of service (DoS), malformed JSON payloads, and unauthenticated endpoint exposure. | Token bucket rate limiting, strict JSON unmarshaling (`DisallowUnknownFields`), size-limited request bodies, and CORS origin allowlists. |
| **Gateway** (`gateway`) | Edge reverse proxies and routing gateways. Primary threats: HTTP request smuggling, hop-by-hop header forgery, IP spoofing, and backend connection exhaustion. | Hop-by-hop header stripping, client IP normalization, upstream host validation, and TLS termination. |
| **Worker** (`worker`) | Asynchronous background consumers. Primary threats: Malicious queue message deserialization, poison pill messages, and broker credential leakage. | Zero exposed public ports, schema validation before job dispatch, and TLS/mutual auth to message brokers. |
| **Site** (`site`) | Compiled static documentation and marketing portals. Primary threats: DOM XSS from untrusted third-party widgets and edge caching misconfigurations. | Ahead-of-time static compilation (zero backend runtime execution), CSP headers, and Subresource Integrity (SRI). |
| **Book** (`book`) | Multi-page structured documentation books. Primary threats: Unescaped markdown rendering and untrusted external hyperlinks. | Contextual HTML escaping and strict external link sanitization. |
| **Infra** (`infra`) | Docker compose topologies, orchestration, and container definitions. Primary threats: Container breakout, running as root, and insecure port exposure to public networks. | Non-root container user (`1000:1000`), read-only root filesystems, and strict loopback binding defaults. |
| **CLI** (`cli`) | Workstation command-line tools. Primary threats: Executing untrusted scripts in current working directory and unauthorized privilege elevation. | Local standard I/O only (no listening sockets) and sha256 checksum verification on downloads. |

---

## 3. Shared Responsibility Model

Security assurance is divided across three distinct ownership layers:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                      Shared Responsibility Matrix                      │
├──────────────────────────┬─────────────────────────────────────────────┤
│ Layer                    │ Responsibilities                            │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Framework Defaults       │ • Embedded security headers (nosniff, DENY) │
│ (Zero-Config / pkg/*)    │ • Automatic CSRF protection                 │
│                          │ • Path traversal defense (SafePath)         │
│                          │ • SSRF protection (SafeHTTPClient)          │
│                          │ • Cryptographic password hashing (PBKDF2)   │
│                          │ • AI agent filesystem workspace sandbox     │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Developer Configuration  │ • Parameterized SQL queries (Anti-SQLi)     │
│ (krewire.yaml & Code)    │ • Domain-level RBAC and policy definitions  │
│                          │ • Origin CORS allowlist specification       │
│                          │ • Rate limiting thresholds                  │
│                          │ • Secret injection via ${env:NAME}          │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Deployment Operator      │ • TLS certificate provisioning (Edge Proxy) │
│ (Infra & Environment)    │ • Production secret rotation & KMS vault    │
│                          │ • Container capability dropping (CAP_DROP)  │
│                          │ • Network firewall & private VPC isolation  │
└──────────────────────────┴─────────────────────────────────────────────┘
```

---

## 4. Canonical Security Controls & Failure Behaviors

The Krewire assurance framework classifies every control into one of three statuses (`model.SecurityControlStatus`):
- **`VERIFIED` (✓)**: Control is active by default and verified by automated test suites or static linters.
- **`CONFIG_DEPENDENT` (⚠)**: Control requires explicit operator configuration or environment variables.
- **`UNSUPPORTED` (ℹ)**: Control is outside the threat model of this workload (e.g. CSRF for asynchronous workers).

### Core Controls Catalog

| Control ID | Name | Category | Status | Default Active | Failure Behavior |
|---|---|---|---|---|---|
| `SEC-SECRETS-ZERO-CODE` | Zero Hardcoded Secrets | Secrets Management | `VERIFIED` | Yes | `krew vet --sec` fails quality gate if plaintext credential found in config or tree |
| `SEC-SUPPLY-CHAIN-VERIFY` | Checksum & Version Pinning | Supply Chain | `VERIFIED` | Yes | `go mod verify` failure halts compilation; pure-Go invariant enforced |
| `SEC-SAFEPATH-TRAVERSAL` | Path Traversal Guard | Input Handling | `VERIFIED` | Yes | Returns `sec.ErrPathTraversal` and refuses filesystem operation |
| `SEC-AI-SANDBOX-BOUNDARIES`| AI Agent Workspace Isolation | AI Governance | `VERIFIED` | Yes | Agent tool call outside workspace boundary is blocked and logged |
| `SEC-HTTP-DEFENSE-HEADERS` | OWASP Defensive Headers | Transport & Edge | `VERIFIED` | Yes | Middleware injects nosniff, DENY, strict-origin, CSP, and HSTS automatically |
| `SEC-CSRF-PROTECTION` | CSRF Token Guard | Session Security | `VERIFIED` | Yes | Fail-closed: returns `HTTP 403 Forbidden` (`sec.ErrCSRFTokenInvalid`) |
| `SEC-SSRF-SAFE-CLIENT` | SSRF Outbound Filter | Input Handling | `VERIFIED` | Yes | Dialer blocks connection to loopback, RFC1918, and metadata before socket dial |
| `SEC-AUTHN-PBKDF2` | Adaptive PBKDF2 Hashing | Authentication | `VERIFIED` | Yes | Constant-time verification; rejects incorrect passwords without timing leaks |
| `SEC-TLS-TERMINATION` | HTTPS / TLS Encryption | Transport & Edge | `CONFIG_DEPENDENT`| No | Plain HTTP on loopback; edge proxy (Cloudflare/Nginx) must enforce redirect |
| `SEC-API-RATE-LIMITING` | Token Bucket Rate Limiting | Transport & Edge | `CONFIG_DEPENDENT`| No | Returns `HTTP 429 Too Many Requests` with `Retry-After` header |
| `SEC-INFRA-CONTAINER-NONROOT`| Non-Root Container Execution | Supply Chain | `VERIFIED` | Yes | Dockerfile specifies `USER krewire (1000:1000)` |
| `SEC-INFRA-READONLY-ROOTFS` | Read-Only Root Filesystem | Supply Chain | `CONFIG_DEPENDENT`| No | Filesystem writes outside `/tmp` fail with `EROFS` |

---

## 5. AI Agent Security Sandbox & Governance

Autonomous AI agents (such as Krewire Krew and developer assistants) must be strictly governed to prevent unintended command execution, data exfiltration, or filesystem corruption:

1. **Filesystem Sandbox Boundary**:
   - Any agent file read, write, or modification is evaluated against `sec.SafePath(workspaceRoot, targetPath)`.
   - Access attempts outside the workspace boundary immediately abort and return `ErrPathTraversal`.
2. **Deterministic Tool Authorization**:
   - Tools that modify project configuration (`krewire.yaml`), update dependencies (`go.mod`), or execute shell commands are mediated by explicit permission checks.
3. **Structured Audit Trail**:
   - Every agent action generates an append-only JSON audit record capturing `who`, `when`, `where`, `what`, and `status`:
     ```json
     {
       "agent_id": "krew-agent-01",
       "timestamp": "2026-11-21T06:55:00Z",
       "action": "fs:write",
       "target": "internal/handler.go",
       "status": "allowed"
     }
     ```
4. **Secrets Redaction in Agent Context**:
   - Environment variables matching password, token, or secret patterns are masked before injection into agent conversation context.

---

## 6. Supply Chain Integrity & Software Bill-of-Materials (SBOM)

Krewire release and verification pipelines enforce multi-tier supply chain integrity:

1. **Cryptographic Checksum Verification (`go mod verify`)**:
   - Every dependency in `go.mod` is checked against `go.sum` hashes to prevent upstream tampering.
2. **Pure-Go Invariant (`CGO_ENABLED=0`)**:
   - Disallows native dynamic linkages (`.so`, `.dylib`), neutralizing vulnerabilities in unmanaged C libraries.
3. **Automated AST CVE Auditing (`govulncheck`)**:
   - Runs `govulncheck ./...` during `krew vet --vuln` to identify known CVEs affecting imported symbols.
4. **Cross-Repository Release Qualification (`krew release --qualify`)**:
   - Every release candidate commit must pass `security-assurance-baseline` contract checks before tagging.
5. **SBOM & Provenance Artifacts**:
   - Production releases generate CycloneDX SBOM manifests and SLSA L2+ provenance records.

---

## 7. Deployment Security Hygiene

Deployment configurations must adhere to defense-in-depth principles:

- **Zero Hardcoded Secrets**: All credentials and sensitive connection strings must be supplied dynamically via environment variables (`${env:VAR_NAME}`).
- **Zero Origin SSL Certificate (Cloudflare Invariant)**:
  - When deployed behind Cloudflare (such as `krewire.com`), origin servers run Nginx listening on port 80 bound to loopback or private host IPs.
  - SSL termination occurs strictly at the Cloudflare edge.
- **Least Privilege Containers**:
  - Production containers never run as UID 0 (`root`).
  - Containers mount the root filesystem read-only where possible, mounting `/tmp` as `tmpfs`.

---

## 8. Running Security Assurance Audits

Developers and CI pipelines run security assurance audits via the `krew` CLI:

```bash
# Run security posture and baseline assurance audit
krew vet --sec

# Run vulnerability and dependency integrity audit
krew vet --vuln

# Run release qualification across ecosystem boundaries
krew release --qualify
```

If any security regression (e.g. unencrypted plaintext secrets, insecure CORS wildcard with credentials, or failed test) is discovered, the quality gate exits with a non-zero exit code (`ExitCodeFailure`), preventing defective artifacts from reaching production.
