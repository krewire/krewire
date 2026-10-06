---
description: Diagnose and fix failing tests or errors by rooting out the cause first.
agent: build
---

Fix the failing behavior in this project.

1. Reproduce the failure: run the failing test or command and capture the full error output ($ARGUMENTS may target a specific test/suite).
2. Run the `debugger` subagent to trace the root cause before changing anything.
3. Apply the minimal fix that addresses the root cause, following project conventions.
4. Verify: rerun the previously failing test and the surrounding tests.

Report the root cause, the fix, and verification results.