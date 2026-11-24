# Reproducible AI Verification & Bounded Execution Guide

This document establishes the canonical verification evidence model, execution sandboxing boundaries, and anti-hallucination guarantees enforced for autonomous AI agents across the Krewire digital SDLC ecosystem per the **Three Pillars (3S)**:
- **Secure (S1)**: Hard filesystem sandboxes (`SafePath`), blocked destructive commands, mandatory human approval for high-impact actions, and automatic secret redaction.
- **Sustain (S2)**: Deterministic, pure-Go verification toolchain (`krew verify`) executable locally on any standard machine without proprietary SaaS gateways.
- **Scalable (S3)**: Evidence-backed verification records that scale across multi-agent workflows, CI/CD qualification gates, and human peer review.

---

## 1. The Verification Evidence Problem

An autonomous AI agent may claim that a code modification or feature implementation is complete while:
- Build errors, static analysis diagnostics, or runtime panics remain unaddressed.
- Architectural layering invariants or security policies have been violated.
- Credentials or sensitive API keys have been hardcoded into configuration.
- Critical required quality gates were silently skipped.

In Krewire, **verbal agent assertions are never accepted as proof of completion**. An AI-generated change is recognized as complete if and only if verifiable, reproducible evidence confirms that all required quality gates have passed.

---

## 2. Canonical Quality Gates & Evidence Lifecycle

Every task evaluated under the verification framework is checked against an explicit progression of quality gates (`model.QualityGateType`):

| Gate ID | Discipline | Canonical Command | Required | Success Criteria |
|---|---|---|---|---|
| `fmt` | Code Formatting | `gofmt -l .` | Yes | Exit code 0, zero unformatted files listed |
| `vet` | Static Analysis | `go vet ./...` | Yes | Exit code 0, zero compiler diagnostics |
| `test` | Automated Testing | `go test -v ./...` | Yes | Exit code 0, all unit and integration tests pass |
| `build` | Compilation Check | `CGO_ENABLED=0 go build ./...` | Yes | Exit code 0, successful pure-Go static binary compile |
| `layering` | Architecture Guard | `go test -v ./layering_test.go` | Yes (when present) | Exit code 0, downward-only dependency invariants upheld |
| `sec` | Security Assurance | `krew vet --sec` | Conditional | Exit code 0, zero hardcoded secrets or CORS misconfigurations |

### The Verification Invariant
```text
OverallVerified = true  ⟺  ∀ Gate g ∈ RequiredGates : Status(g) == PASSED
```
If **any** required quality gate is `FAILED`, `SKIPPED`, `BLOCKED`, or `PLANNED`, `OverallVerified` is strictly `false`. No agent can fabricate completion when a required gate has not executed cleanly.

---

## 3. Tamper-Evident Evidence Records

Each verification execution produces a structured `TaskVerificationRecord` (`model.TaskVerificationRecord`):

```json
{
  "task_id": "issue-123",
  "agent_id": "krew-agent-01",
  "timestamp": "2026-11-21T07:15:00Z",
  "git_commit": "4c3f30d",
  "working_tree_clean": true,
  "overall_verified": true,
  "summary": "all 5 required verification gates passed successfully",
  "gates": [
    {
      "gate_id": "fmt",
      "type": "FORMAT",
      "command": "gofmt -l .",
      "required": true,
      "status": "PASSED",
      "exit_code": 0,
      "duration_ms": 22,
      "evidence_hash": "a1b2c3d4e5f6..."
    },
    {
      "gate_id": "test",
      "type": "TEST",
      "command": "go test ./...",
      "required": true,
      "status": "PASSED",
      "exit_code": 0,
      "duration_ms": 1450,
      "evidence_hash": "f6e5d4c3b2a1..."
    }
  ]
}
```

### Cryptographic Fingerprinting (`EvidenceHash`)
Each gate's evidence is cryptographically bound to the git commit, exact command string, exit status, and sanitized output using SHA-256:
$$\text{EvidenceHash} = \text{SHA-256}(\text{GateID} \mathbin{\Vert} \text{Command} \mathbin{\Vert} \text{GitCommit} \mathbin{\Vert} \text{ExitCode} \mathbin{\Vert} \text{Status} \mathbin{\Vert} \text{Output})$$
A human reviewer or downstream CI pipeline can verify that the evidence matches the recorded working-tree state.

---

## 4. AI Agent Execution Sandbox & Authorization Policy

Autonomous AI agents operate under bounded execution policies defined in `sec.AgentPolicy`:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AI Agent Security Policy Boundary                    │
├──────────────────────────┬─────────────────────────────────────────────┤
│ Dimension                │ Rule & Enforcement Mechanism                │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Filesystem Boundary      │ Constrained strictly to WorkspaceRoot via   │
│                          │ sec.SafePath. Writes outside abort with     │
│                          │ ErrAgentPolicyDenied.                       │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Destructive Commands     │ Denied by default. Blocks rm -rf, mkfs,     │
│                          │ git push --force, drop database, kill -9 1. │
├──────────────────────────┼─────────────────────────────────────────────┤
│ High-Impact Operations   │ Require explicit human authorization        │
│                          │ (ActionGitPush, ActionDeploy).              │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Network Access           │ Outbound socket connections disabled        │
│                          │ unless explicitly granted in policy.        │
├──────────────────────────┼─────────────────────────────────────────────┤
│ Secret Redaction         │ Automatic regex masking of passwords,       │
│                          │ bearer tokens, and private keys in logs.    │
└──────────────────────────┴─────────────────────────────────────────────┘
```

---

## 5. Running Verification via `krew verify`

Developers and autonomous agents execute reproducible verification checks using the `krew` CLI:

### Human-Readable Execution
```bash
krew verify --task issue-123 --agent krew-agent-01
```

Example Output:
```text
── Task Verification Evidence Record [issue-123] ─────────────
  • Agent ID:           krew-agent-01
  • Git Commit:         4c3f30d
  • Working Tree Clean: true
  • Verified Status:    ✓ FULLY VERIFIED (All required quality gates passed)

  [EVALUATED QUALITY GATES]
    ✓ [PASS] fmt        (25ms)  — Go code formatting check (gofmt -l .)
      • Command:  gofmt -l .
      • Exit:     0
      • Hash:     c4ca4238a0b923820dcc509a6f75849b...
    ✓ [PASS] vet        (120ms) — Go static analysis and vet checks (go vet ./...)
      • Command:  go vet ./...
      • Exit:     0
      • Hash:     c81e728d9d4c2f636f067f89cc14862c...
    ✓ [PASS] test       (1840ms)— Automated unit and integration test suite (go test ./...)
      • Command:  go test ./...
      • Exit:     0
      • Hash:     eccbc87e4b5ce2fe28308fd9f2a7baf3...
    ✓ [PASS] build      (410ms) — Pure-Go static compilation check (go build ./...)
      • Command:  go build ./...
      • Exit:     0
      • Hash:     a87ff679a2f3e71d9181a67b7542122c...

  Summary: all 4 required verification gates passed successfully
```

### Machine-Readable JSON Pipeline Output
```bash
krew verify --json --task issue-123 > .krewire/evidence.json
```
If any gate fails, the command terminates with `errs.ExitCodeFailure` (code 1), preventing broken code from being committed, merged, or deployed.
