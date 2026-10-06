# Bug Triage Rules

- Reproduce before classifying; if it cannot be reproduced, state the missing information explicitly instead of guessing.
- Escalate P0 immediately rather than waiting for approval.
- Treat a regression as the most actionable case: `git bisect` or diff against the last-known-good version.
- Produce one actionable outcome per report: a fix task or a clarified follow-up question.
- Link related or duplicate reports.