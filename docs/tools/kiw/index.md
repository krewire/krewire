# Krewire Devtool (`kiw`)

The unified command-line toolchain powering every phase of the Krewire development lifecycle.

---

## Overview

Rather than juggling fragmented bash scripts, npm build runners, and separate CLI utilities, Krewire developers use a single binary: `kiw`.

```text
kiw <command> [flags]
```

`kiw` dogfoods the monorepo's own Terminal UI toolkit ([`packages/tui`](../packages/web-runtime/ui-tui.md)) and kernel execution standards ([`packages/kern`](../packages/core/kern.md)).

---

## Documentation Sections

1. **[Command Reference](./commands.md)**
   Full documentation of every command across `project`, `inspect`, `build`, `develop`, and `ship` groups.
2. **[Configuration Specification (`krewire.yaml`)](./configuration.md)**
   Detailed schema reference, environment overrides, and custom script task runners.
3. **[Development & Build Pipelines](./dev-and-build.md)**
   Local live-reload, WebSocket HMR, compilation pipelines, and production staging.
