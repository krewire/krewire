# AI-Native Engineering Model: Krew, Boost, and kiw Interoperability Specification

| Field | Value |
|---|---|
| ID | ENG-AI-001 |
| Title | AI-Native Engineering Model: Krew, Boost, and kiw Interoperability Specification |
| Status | Approved (Canonical Engineering Specification) |
| Date | 2026-11-21 |
| Owners | Krewire AI Engineering & Toolchain Maintainers |
| Precedence | Precedence 1 (Core Technical Specification) |

---

## 1. Executive Summary

This specification defines the unified, governed AI engineering model across **Krewire Krew**, **Krewire Boost**, and the **`krew` devtool CLI**.

Autonomous AI coding agents can dramatically accelerate software delivery, but without strict contracts, they risk introducing hallucinated dependencies, violating architectural invariants, and reporting false success.

This document formalizes the division of responsibilities, context contracts, autonomy boundaries, approval checkpoints, and evidence-backed verification standards required for consequential engineering work.

---

## 2. Tripartite Division of Responsibilities

The Krewire AI engineering battery divides responsibilities strictly across three components:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   KREWIRE KREW (krewire/krew)                          │
│               Autonomous Agent Execution Engine                        │
│  • ReAct cognitive execution loop (Think ──► Act ──► Observe)         │
│  • LLM provider routing & streaming (Claude, Gemini, OpenAI, Ollama)   │
│  • Multi-agent communication swarms & peer delegations                 │
│  • Model Context Protocol (MCP) transport (Embed, CLI, SSE, Stdio)     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ guided by
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                KREWIRE BOOST (templates/boost, pkg/boost)              │
│               Multi-Agent Blueprint & Constitution                     │
│  • 21 Specialized persona definitions (planner, builder, tester, etc.) │
│  • Architectural conventions & safety constitutions (AGENTS.md)        │
│  • Workflow slash commands (/plan, /test, /build, /commit)             │
│  • Reusable project initializer (krew boost install)                    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ drives & verifies through
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   KIW DEVTOOL (github.com/krewire/krew)                      │
│               Deterministic Toolchain & Context SSOT                   │
│  • Project context inspection & linting (kiw context --json, --lint)   │
│  • Declarative configuration engine (krewire.yaml)                     │
│  • Workload lifecycle execution & build runner (krew dev, krew build)    │
│  • Objective quality gates (gofmt, go vet, go test, krew qualify)       │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.1 The `krew` Toolchain (The Truth Provider)
- **Role**: Deterministic CLI driver, build runner, and project truth provider.
- **Contract**: `krew` exposes the machine-readable single source of truth via `kiw context --json` and validates integrity via `kiw context --lint`.
- **Independence**: `krew` is 100% pure Go and operates deterministically without requiring an AI provider, API key, or network connection.

### 2.2 Krewire Boost (The Engineering Constitution)
- **Role**: Declarative multi-agent template, persona catalog, and behavioral constitution.
- **Contract**: Boost provides the rules, commit conventions, architectural constraints, and slash commands embedded into the repository via `AGENTS.md` and `.agents/`.
- **Independence**: Boost templates are standard Markdown and JSON files installable into any project.

### 2.3 Krewire Krew (The Cognitive Engine)
- **Role**: Autonomous agent runtime and execution coordinator.
- **Contract**: Krew consumes the context provided by `krew`, follows the constitution defined by Boost, and interacts with tools via the Model Context Protocol (MCP).

---

## 3. The 7-Phase Governed AI Engineering Lifecycle

Every autonomous agent task must follow an unbroken, auditable 7-phase lifecycle:

```text
[1. Context Discovery] ──► [2. Clarify & Align] ──► [3. Phased Planning] ──► [4. Implement]
                                                                                   │
[7. Evidence Report]   ◄── [6. Audit & Commit]  ◄── [5. Multi-Gate Verify] ◄───────┘
```

1. **Phase 1: Context Discovery**:
   - Agents must invoke `kiw context --json` or parse `model.ProjectContext` to discover the authoritative workload, tier, Go version, module path, and Three Pillars invariants before planning.
   - Relying on unstructured prompt assumptions is prohibited.
2. **Phase 2: Task Clarification & Alignment**:
   - Map user requirements to an authoritative GitHub Issue. If ambiguity exists, clarify intent before modifying code.
3. **Phase 3: Phased Planning & Acceptance Criteria**:
   - Draft a concise implementation plan specifying affected packages, interface contracts, and unit tests.
4. **Phase 4: Atomic Implementation**:
   - Make minimal, granular code changes ($\le 300$ lines of code per commit). Follow the Single Responsibility Principle and strict downward-only layering.
5. **Phase 5: Multi-Gate Quality Verification**:
   - Execute all four mandatory quality gates:
     1. Formatting: `gofmt -l .` (must return 0 lines).
     2. Static analysis: `go vet ./...` (exit code 0).
     3. Test execution: `go test -v ./...` (all unit and layering tests pass).
     4. Pure-Go compilation: `CGO_ENABLED=0 go build ./...` (compilation succeeds).
6. **Phase 6: Provenance Audit & Atomic Commit**:
   - Stage and commit changes atomically inside the owning repository using Conventional Commits with issue traceability (`feat(scope): description (ref #123)`).
7. **Phase 7: Evidence-Backed Completion Reporting**:
   - Present a structured summary with verifiable proof of test execution, modified files, and closed acceptance criteria.

---

## 4. Human Decision Rights, Autonomy, and Escalation

To ensure safety and human sovereignty, agent actions are strictly bounded:

### 4.1 Level 1: Fully Autonomous Actions
Agents may perform the following without requesting interactive human approval:
- Reading files, documentation, and directory structures.
- Running read-only inspections (`krew info`, `kiw context`, `git status`).
- Executing tests and static analysis (`go test`, `go vet`, `gofmt`).
- Creating and editing source files within the active task scope.
- Executing local atomic git commits once all four quality gates pass.

### 4.2 Level 2: Actions Requiring Explicit Human Approval
Agents must pause and obtain interactive human confirmation before:
- Deleting files or directories outside temporary scratch folders.
- Pushing commits to remote git repositories (`git push`).
- Executing destructive database migrations on shared or production environments.
- Altering repository licensing posture or commercial tiers (`krewire.yaml` tier).
- Generating or modifying infrastructure provisioning plans for live cloud targets.

### 4.3 Level 3: Hard Escalation & Blocking Conditions
Agents must immediately halt and escalate to the human developer when:
- Quality gates fail repeatedly after a fixed retry budget (maximum 3 iterations).
- An instruction would violate the **Three Pillars (Secure, Sustain, Scalable)**.
- A proposed edit introduces circular package dependencies or violates downward layering (`Layer N → Layer > N`).
- A requested resource is gated by credentials, CAPTCHA, or undocumented access boundaries.

---

## 5. Security & Tool Execution Policies

1. **Path-Traversal Prevention (`SafePath`)**:
   - Any agent tool operation receiving file paths must validate them against the workspace root. Directory traversal attacks (`../`) are physically rejected.
2. **Zero Committed Secrets**:
   - Agents are prohibited from generating, hardcoding, or committing API keys, tokens, or private certificates. Secrets must be referenced dynamically via environment variables (`${env:NAME}`).
3. **Pure-Go Invariant**:
   - Agents must never introduce native C bindings or external runtime dependencies (e.g. Node.js, Python) into Go packages. `CGO_ENABLED=0` compilation must succeed at all times.

---

## 6. Evidence-Backed Completion Report Standard

An agent's self-reported claim of success is never considered proof of correctness. Every completion report must provide concrete, verifiable evidence:

```markdown
### Verification Summary
- **Context**: `github.com/krewire/krewire` (Workload: `app`, Tier: `free`, Go: `1.27.1`)
- **Quality Gate Results**:
  • `gofmt -l .` ──► Passed (0 unformatted lines)
  • `go vet ./...` ──► Passed (exit code 0)
  • `go test -v ./...` ──► Passed (42 tests, 0 failures)
  • `CGO_ENABLED=0 go build ./...` ──► Passed (pure-Go binary compiled)
- **Commits**: `cc6ea95` feat(kiw): establish canonical agent context (ref #108)
- **Files Modified**:
  • [context.go](file:///home/krewire/workspace/krewire/krewire/pkg/model/context.go)
  • [loader.go](file:///home/krewire/workspace/krewire/github.com/krewire/krew/internal/context/loader.go)
```

---

## 7. Replaceability and Zero-Lock-In Invariant

Krewire projects are never dependent on proprietary AI providers:
- Removing or disabling Boost (`AGENTS.md`, `.agents/`) has zero impact on application runtime.
- The `krew` devtool CLI and all 32 modular libraries (`pkg/*`) compile, run, test, and deploy identically with or without AI agents.
- Developers can swap agent engines (Krew, Claude Code, Cursor, Copilot, OpenCode) seamlessly because all agents consume the same open standard (`krew context --json`).

---

## 8. Architectural Invariants (The Three Pillars)

The AI engineering model enforces Krewire's Three Pillars:

1. **Secure**:
   - Agents operate under least-privilege sandboxes with SafePath traversal guards and zero stored secrets.
2. **Sustain**:
   - Agent workflows enforce lean footprints (10–30 MB RAM) and prevent dependency bloat from creeping into repositories.
3. **Scalable**:
   - Code produced by agents adheres strictly to the 7-stage progressive pipeline and 6-layer pyramid architecture, preventing monolithic rot.

> [!IMPORTANT]
> The three pillars must always remain explicitly named **Secure**, **Sustain**, and **Scalable**. They serve as the foundational constraints for all autonomous agent behavior.
