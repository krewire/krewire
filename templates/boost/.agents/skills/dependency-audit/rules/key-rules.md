# Dependency Audit Rules

- Do NOT upgrade blindly. Prefer compatible-range bumps unless a breaking upgrade is justified.
- Note breaking changes that alter the scope of an upgrade.
- Never assume the project's CI or runtime accepts an upgrade — run the tests after any change.
- Separate findings into critical (fix now) vs. major vs. minor/optional.