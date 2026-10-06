# CI/CD Rules

- Use the exact install/lint/test/build commands the project already has; never invent new ones.
- Match the platform the team already uses (default: GitHub Actions).
- Cache aggressively and keep CI fast; split matrix jobs only when the stack truly needs it.
- Protect production: required status checks, branch protection, and an approval step for prod environments.
- Never claim a pipeline works without testing it — push a branch/PR and confirm green, or give exact trigger steps.