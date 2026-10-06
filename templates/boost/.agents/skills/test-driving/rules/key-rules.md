# Test Driving Rules

- Ask rather than guess a test command that is not backed by the manifest/scripts.
- Establish a baseline: run the existing suite once before changing anything.
- Mirror existing test naming, location, and style.
- Test behavior (input/output), not private implementation details.
- For bug fixes: write a failing test that reproduces the bug first, then fix — it becomes regression protection.
- Never leave the suite red; keep it green.