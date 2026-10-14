# Development & Build Pipelines in `kiw`

Detailed execution model for local development, watch mode, compilation pipelines, and artifact staging.

---

## 1. Local Development Modes

### A. Hot-Reload (`kiw dev`)
Monitors the filesystem for changes across Go source files, templates, and config files. When a change is detected:
1. Cancels the currently running process via SIGINT.
2. Recompiles the application binary.
3. Restarts the supervised process immediately.

### B. Static Site Live-Reload (`kiw serve`)
Serves static content from `.krewire/build` and injects a lightweight WebSocket script into HTML pages:
- Changes to Markdown files trigger instant partial DOM re-renders.
- Changes to CSS trigger instant style sheet swaps without full page reload.

---

## 2. Build Pipelines (`kiw build`)

```text
kiw build
  ├── Detect Kind
  ├── Validate Invariants (packages/kern/model)
  ├── Run Pre-Build Scripts
  └── Execute Kind Pipeline:
        ├── site:   SSG Builder -> Minification -> .krewire/build
        ├── book:   mdbind compiler -> Navigation index -> .krewire/build
        ├── app:    WASM build (client/) -> Go binary (server) -> bin/
        └── cli:    Go build with embedded assets -> bin/
```

---

## 3. Staging & Deployment (`kiw deploy`)

Prepares clean deployable bundles inside `dist/`:
- Validates quality gates (`gofmt`, `vet`, `test`).
- Embeds production assets.
- Generates container deployment specifications or cloud runner blueprints.
