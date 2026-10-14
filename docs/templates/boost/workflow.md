# Boost AI SDLC & Multi-Agent Workflow

How autonomous agents execute requests from inception to verified deployment.

---

## 1. Six-Step Execution Lifecycle

```text
1. Understand ──► 2. Map (Scout) ──► 3. Plan & Spec ──► 4. Implement ──► 5. Verify ──► 6. Summarize
```

1. **Understand**: Clarify ambiguous user requests before starting work.
2. **Map**: Map project layout, scripts, dependencies, and rules once per repository.
3. **Plan & Spec**: Generate design decisions, architecture models, or formal specifications for complex tasks.
4. **Implement**: Execute minimal, convention-respecting code changes using specialized subagents (`builder`, `decomposer`).
5. **Verify**: Run automated quality gates (`gofmt`, `vet`, `test`, `build`) to guarantee zero regression.
6. **Summarize**: Report clear, human-readable summaries citing changed files, rationale, and test results.

---

## 2. SDLC Phase Mapping

| Phase | Subagents | Predefined Commands | Skills |
|---|---|---|---|
| **Planning & Design** | `planner`, `thinker`, `designer` | `/plan`, `/spec` | `planning`, `designing`, `thinking` |
| **Implementation** | `builder`, `decomposer`, `automator` | `/refactor`, `/fix`, `/commit` | `building`, `decomposing`, `automating` |
| **Testing & QA** | `tester`, `auditor` | `/test`, `/audit` | `testing`, `auditing` |
| **Release & Deploy** | `deployer`, `maintainer` | `/deploy`, `/release` | `deploying`, `maintaining` |
| **Hygiene & Learning** | `housekeeper`, `learner` | `/learn`, `/compact` | `housekeeping`, `learning` |
