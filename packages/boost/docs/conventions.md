# Conventions

Conventions the agent enforces across projects, regardless of stack. Projects may tighten
these in AGENTS.md under "Project-Specific Customization".

## Code

- Follow the existing style of the file/section being touched — never reformat unrelated code.
- Prefer small, focused files and functions. Clear names over cleverness.
- No speculative abstraction: add structure when there is a second real use, not before.
- Comments explain "why", not "what". Do not add comments unless needed or asked.

## Dependencies

- Reuse what the project already depends on. Adding a library needs a concrete reason.

## Version control

- Conventional Commits (see `../.agents/skills/conventional-commit/SKILL.md`).
- One logical change per commit; do not commit unrelated files or secrets.

## Testing

- Changes to behavior come with tests, matching the project's existing harness and style.
- Keep the suite green before finishing.

## Documentation

- Public or setup behavior changes update the README/docs.
- Complex features get a spec in `docs/specs/` before implementation.

## Safety

- Destructive operations (`rm`, force-push, overwrite) require explicit confirmation.
- Never introduce or log secrets. Never commit secrets.