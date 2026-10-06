---
name: requirement-gathering
description: Use in the planning phase to elicit, structure, and document requirements before design or implementation: feature requests, user stories, acceptance criteria, scoping. Triggers: "requirement", "user story", "acceptance criteria", "scope", "what should this do", "use case".
---

# Requirement Gathering

Turn a vague request into clear, testable requirements before any design or code.

## Elicit

Ask targeted questions to close the gaps. Do not guess product decisions:

- **Users** — who uses this, and what job are they trying to do?
- **Inputs/outputs** — what does it take in, what does it produce?
- **Rules** — business rules, edge cases, constraints (perf, security, compliance)?
- **Success** — how do we know it works? (measurable)
- **Boundaries** — what is explicitly NOT in scope for this iteration?

Prefer confirming statements: "So the flow is X on success and Y on failure — correct?"

## Structure

For each requirement record:

- **ID** — short reference (REQ-1, US-2).
- **Story** — "As a <role>, I want <capability>, so that <benefit>."
- **Acceptance criteria** — concrete, verifiable conditions ("Given/When/Then").
- **Priority** — Must / Should / Could / Won't (this iteration).
- **Dependencies** — what must exist first.

Requirements must be testable. If you cannot write a test for it, the requirement is not done.

## Hand off

Save as a requirements section, e.g. `docs/requirements.md`, then reference it in the spec
(the `spec-writing` skill) or hand it to `plan` for design. Requirements belong to the
planning phase; keep design decisions out of them.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.