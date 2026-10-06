---
name: bug-triage
description: Use in the maintenance phase to triage incoming bugs or support issues into actionable, prioritized tasks: reproduce, classify severity/priority, and route to a fix. Triggers: "triage", "bug report", "issue", "severity", "priority", "classify this bug".
---

# Bug Triage

Turn raw bug reports and support issues into clear, prioritized, actionable tasks.

## 1. Reproduce first

- Determine the environment (version, OS, step/flow) from the report.
- Attempt a minimal reproduction. Record the exact steps and observed behavior.
- If it cannot be reproduced, state the missing information explicitly instead of guessing.

## 2. Classify

- **Severity** (impact if real): Critical (data loss/security/blocked workflow) · High (core feature broken) · Medium (partial, has workaround) · Low (cosmetic/nice-to-have).
- **Priority** (urgency): P0 (fix now) · P1 (this iteration) · P2 (next) · P3 (backlog).
- Category: bug · regression (worked before — check git history to confirm) · enhancement · doc gap · environment/config.

A regression is the most actionable: `git bisect` the history or diff the last-known-good version.

## 3. Route

- Produce one actionable outcome per report: either a concrete fix task (with the suspected root-cause area and a failing-test-first suggestion), or a clarified follow-up question to the reporter.
- Link related or duplicate reports.

## 4. Report

```
## Triaged: <ID or title>
- Reproduced: yes/no (steps, env)
- Severity / Priority
- Category
- Suspected area (file:line if found)
- Recommended action (fix task + regression test) / needs info from reporter
```

Escalate P0 immediately rather than waiting for approval.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.