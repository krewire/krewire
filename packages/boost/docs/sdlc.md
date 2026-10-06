# SDLC Phase Coverage

How the agent workspace covers each phase of the software development life cycle.
Every capability below is a real file in `.agents/` — agents, commands, or skills.

| SDLC phase | Agents | Commands | Skills |
| --- | --- | --- | --- |
| 1. Planning | `plan` | `/spec` | `requirement-gathering`, `spec-writing` |
| 2. Design | `plan` | `/spec` | `spec-writing` |
| 3. Implementation | `build`, `reviewer`, `refactor`, `security` | `/fix`, `/refactor`, `/commit` | `conventional-commit`, `security-review` |
| 4. Testing / QA | `tester`, `debugger` | `/test`, `/fix` | `test-driving`, `quality-gate` |
| 5. Deployment / Release | `deploy` | `/deploy`, `/release` | `ci-cd`, `release`, (gate: `quality-gate`) |
| 6. Maintenance / Ops | `debugger`, `reviewer`, `security` | `/triage` | `bug-triage`, `dependency-audit`, `security-review` |

## The full loop in practice

```
/new-project   → bootstrap a new project (project-init skill)
/kickoff       → map the project (scout)
/spec          → planning & design (spec-writing)
<implementation> → build + review + refactor (conventional-commit)
/test, /fix    → QA (test-driving, quality-gate)
/release       → version + changelog + tag (release, ci-cd)
/deploy        → ship & verify (deploy agent)
/triage        → support & fixes (bug-triage)
/dependency-audit → health of dependencies (maintenance)
```

Each phase ends with a verifiable quality gate before the next phase begins
(see the `quality-gate` skill).