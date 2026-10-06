---
description: Refactor code safely in small, behavior-preserving steps. Optionally target a path or area: /refactor [path].
agent: build
---

Refactor the specified code area safely.

1. Scope: $ARGUMENTS if provided (file, package, or area). Otherwise ask what should be refactored and why.
2. Establish a baseline: run the relevant tests so you have a safety net.
3. Delegate the work to the `refactor` subagent: small, behavior-preserving steps with verification after each.
4. Do not fix unrelated bugs or add features along the way.

Report the steps taken, verification results at each step, and the final status.