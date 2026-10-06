# Spec Writing Rules

- Read the relevant code first; ground the spec in reality. Cite file paths for claims.
- Keep it concise. A spec is direction, not an essay.
- Language matches the project's docs (English unless the project uses another).
- Save to `docs/specs/` with the ecosystem file name `{ProjectId}-{Scope}-{SpecID}-{slug}.md` and a unique random 5-char `SpecID`.
- Metadata table: `SpecID`, `Title`, `Status`, `Date`, `Author`, `Domain` — no `Version`, `Last updated`, or `Changes`.
- Every new feature gets a requirement row with an ID tracing spec → implementation → test; update `docs/specs/index.md`.