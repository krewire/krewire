# Conventional Commits Rules

- The summary is imperative, present tense, lowercase: "fix crash on empty input", not "fixed crash" or "Fixes crash".
- Keep the summary under ~72 characters.
- Scope is a short noun for the area changed (`feat(api)`, `fix(cart)`).
- Use the body to explain the "why", not restate the diff.
- Breaking changes: append `BREAKING CHANGE: description` in the footer or use `!` after type, e.g. `feat(api)!: remove v1 endpoints`.
- One logical change per commit. Do not mix unrelated changes or include secrets.