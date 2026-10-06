# Workflow

How the agent moves from a request to shipped, verified work.

## 1. Understand

Read the request. If ambiguous, ask a targeted question instead of guessing.

## 2. Map (once per new project)

Run `/kickoff` → `scout` subagent maps stack, scripts, structure, and docs.
The map is recorded so later sessions reuse it instead of re-discovering.

## 3. Plan

Non-trivial work gets a short plan first:

- `/plan` (agent) — design/architecture decisions before coding.
- `/spec <name>` — writes a design doc to `docs/specs/` first.
- `/refactor [path]` — restructures with a test baseline first.

## 4. Implement

The `build` agent makes the smallest convention-respecting change.
Role subagents step in when specialized:

| Need | Subagent |
| --- | --- |
| Map an unfamiliar project | `scout` |
| Review before merge | `reviewer` |
| Write or run tests | `tester` |
| Root-cause a failure | `debugger` |
| Large safe restructure | `refactor` |
| Docs and specs | `docs` |
| Security pass | `security` |
| Ship a build | `deploy` |

## 5. Verify

Run the project's tests/lint/build. Never finish without this. If failing, debug to the root
cause instead of patching symptoms.

## 6. Summarize

Report: what changed, why, and how it was verified. Enough to trust without reading the diff.

## How the pieces fit

```
/new-project            (bootstrap a new project inc. template)
/kickoff → scout        (map the project)
/plan, /spec            (decide before coding)
build + subagents       (implement + specialized help)
/test, /fix, /review    (verify and harden)
/release → /deploy      (version, ship, verify)
/triage                 (support and fixes after release)
/commit                 (ship a clean commit)
```

Commands trigger workflows; skills provide the know-how inside the workflows;
agents carry the actual work. See `sdlc.md` for the phase-by-phase mapping.