# Boost AI SDLC & Multi-Agent Workflow

This specification defines the 6-step autonomous development lifecycle, slash command mapping, and quality gate enforcement across all projects managed by Krewire Boost.

---

## 1. Six-Step Autonomous Execution Lifecycle

Every agent task moves through six deterministic stages:

```text
[1. Understand] ──► [2. Map] ──► [3. Plan & Spec] ──► [4. Implement] ──► [5. Verify] ──► [6. Summarize]
```

1. **Understand (Clarify & Align)**:
   - Identify the user's intent. If requirements are ambiguous, clarify using structured questions before taking action.
2. **Map (Scout Repository Context)**:
   - Inspect workspace layout, `krewire.yaml`, active Git branch, dependencies, and project conventions once per session.
3. **Plan & Spec (Architecture & Contracts)**:
   - For non-trivial features, draft a clear plan: file changes, public interfaces, and test scenarios. Document ADRs if architectural decisions are involved.
4. **Implement (Atomic Code Changes)**:
   - Modify or create files using specialized agents (`builder`, `decomposer`). Follow idiomatic Go conventions, downward-only dependencies, and the Single Responsibility Principle.
5. **Verify (Quality Gates Checklist)**:
   - Execute formatting (`gofmt`), static analysis (`go vet`), unit tests (`go test ./...`), and pure-Go build (`CGO_ENABLED=0 go build ./...`).
6. **Summarize (Deterministic Feedback)**:
   - Provide a clear, structured summary citing updated files, passing tests, and follow-up recommendations.

---

## 2. SDLC Phase & Slash Command Matrix

Boost provides user-facing slash commands mapped directly to autonomous agent workflows:

| SDLC Phase | Specialized Subagents | Predefined Commands | Skill Packages |
| :--- | :--- | :--- | :--- |
| **Requirements & Planning** | `planner`, `thinker`, `manager` | `/plan`, `/spec`, `/grill-me` | `planning`, `thinking`, `managing` |
| **Architecture & Design** | `designer`, `thinker` | `/design`, `/adr` | `designing`, `thinking` |
| **Implementation** | `builder`, `decomposer`, `automator` | `/build`, `/refactor`, `/fix`, `/commit` | `building`, `decomposing`, `automating` |
| **Testing & Verification** | `tester`, `auditor` | `/test`, `/audit`, `/verify` | `testing`, `auditing` |
| **Release & Deployment** | `deployer`, `maintainer` | `/deploy`, `/release` | `deploying`, `maintaining` |
| **Hygiene & Evolution** | `housekeeper`, `learner`, `writer` | `/learn`, `/compact`, `/docs` | `housekeeping`, `learning`, `writing` |

---

## 3. Checkpoint Coordination & Quality Gates

Before an agent considers any task complete, all four gates must pass:

1. **Formatting Gate**:
   ```bash
   gofmt -l . # Must return 0 files
   ```
2. **Static Analysis Gate**:
   ```bash
   go vet ./... # Must exit with code 0
   ```
3. **Layering & Test Gate**:
   ```bash
   go test -v ./... # All tests and architectural guards must pass
   ```
4. **Pure-Go Build Gate**:
   ```bash
   CGO_ENABLED=0 go build ./... # Must compile cleanly without CGO
   ```
