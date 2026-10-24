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
  ├── Validate Invariants (pkg/model)
  ├── Run Pre-Build Scripts
  └── Execute Kind Pipeline:
        ├── site:   SSG Builder -> Minification -> .krewire/build
        ├── book:   book compiler -> Navigation index -> .krewire/build
        ├── app:    WASM build (wasm:) -> Go binary -> .krewire/build/bin/<name>
        └── cli:    Go build with embedded assets -> .krewire/build/bin/<name>
```

---

## 3. Staging & Deployment (`kiw deploy`)

Stages deployable artifacts inside `.krewire/dist/`:
- Builds the binary (`binary` target) or exports the site into `.krewire/dist/site`.
- Publishes the pages branch for `gh-pages`, emits a systemd unit for `systemd`, or writes `plan.json` for `infra`.
- `--dry-run` stages artifacts and skips publishing; `--target` selects the deployment target.
