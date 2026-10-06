---
name: security-review
description: Use when reviewing code or changes with security in mind, or when asked to audit for vulnerabilities. Trigger on "security", "vulnerability", "is this safe", "audit", "review for security".
---

# Security Review

A practical checklist for evaluating code and setups for security weaknesses.

## 1. Injection & input handling

- SQL/query injection, shell injection via `exec`/`system`/`subprocess`, template injection, path traversal, SSRF.
- Validate and whitelist input at every trust boundary. Escape or parameterize dynamic values.

## 2. Secrets

- Hardcoded credentials, API keys, or tokens in source or config defaults.
- Secrets leaking into logs, error messages, or URLs.
- Keys committed to the repository — check .gitignore and recent history.

## 3. Auth & access control

- Missing or weak authentication; defaults that are insecure.
- Authorization enforced at the right layer (server-side, not just hidden UI).
- Overly permissive CORS, ACLs, file permissions, or role defaults.

## 4. Data handling

- Sensitive data logged or exposed (PII, passwords, tokens).
- Insecure storage or transmission (plaintext, no TLS, weak ciphers).
- Unsafe deserialization (pickle, yaml.load, eval) of untrusted data.

## 5. Dependencies

- Known-vulnerable packages (run the `dependency-audit` skill for a full sweep).
- Pin/update responsibly; lockfiles committed.

## 6. Operations

- Risky shell commands in scripts/CI, privilege escalation, unsafe temp files, insecure network calls (http, unverified TLS).

## Output

Report findings by severity (Critical/High/Medium/Low) with file:line references and a concrete recommended action for each. Only report what is real and actionable in context.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.