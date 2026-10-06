---
name: spec-writing
description: Use when asked to write a design/spec document, plan a complex feature, or document architecture decisions before implementation. Trigger on "write a spec", "design doc", "plan this feature", "why did we choose this".
---

# Spec Writing

Produce clear, grounded design documents that guide implementation without over-specifying.

## When to use

- A new feature or complex change that touches multiple parts of the codebase.
- An architectural decision that should be recorded.
- The AGENTS.md says specs should be written before implementation.

## Template (Global Industry Standard — Krewire Adaptation)

Use `rules/spec-template.md` (this skill's template, IEEE 830 + RFC 2119 + Google Design Doc + ADR) as the single source. Copy it to `docs/specs/` as `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.

The template is optimized for both human review and AI implementation: every section has a purpose, RFC 2119 keywords, and a checklist. See `rules/spec-template.md` for the full 14-section structure and `rules/issue-template.md` (in `issue-writing` skill) for the paired GitHub issue.

File placement and metadata follow the ecosystem convention:

- Location: `docs/specs/` inside the owning repo; update `docs/specs/index.md`.
- File name: `{ProjectId}-{Scope}-{SpecID}-{slug}.md` (e.g. `KWN-init-7QM2X-init-project-variants.md`). `Scope` is a short category (`cli`, `web`, `init`, ...).
- `SpecID` is a unique random 5-character alphanumeric code — pick randomly, never reuse or sequence.
- Mandatory metadata table: `SpecID`, `Title`, `Status`, `Date`, `Author`, `Domain`. There is **no** `Version`, `Last updated`, or `Changes` field — revision history lives in git (`git log -1 -- <path>`).
- **Spec first.** Write the spec before implementation. Every new feature has a requirement row with an ID (e.g. `RND-INIT-012`) that traces spec → implementation → test.
- One spec per initiative — never one spec for several unrelated changes.

## Rules

Read `rules/key-rules.md` before acting and follow those rules.