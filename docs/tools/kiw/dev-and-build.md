# Development & Build Pipelines in `kiw`

Detailed execution model for local development, watch mode, compilation pipelines, and artifact staging managed by the `kiw` toolchain.

---

## 1. Local Development Modes

### A. Live Watch & Reload (`kiw dev`)
Monitors the filesystem for changes across Go source files, `.kiw` templates, markdown files, and configuration. When changes occur:
1. **Apps, Services, Workers, Gateways, and CLIs**: Sends `SIGINT` to cancel the active running process, recompiles the binary with `CGO_ENABLED=0`, and restarts the supervised process immediately.
2. **Static Sites & Books**: Re-evaluates templates or markdown chapters and broadcasts hot reload updates over WebSockets.

### B. Static Site Live Preview (`kiw serve`)
Serves pre-built static content from `.krewire/build` and injects a lightweight WebSocket script into HTML pages:
- Changes to Markdown files trigger instant partial DOM re-renders.
- Changes to CSS trigger instant style sheet swaps without full page reloads.

---

## 2. Compilation Pipelines (`kiw build`)

`kiw build` automatically detects the active project shape ([`shape.Detect`](../../../tools/kiw/internal/shape/shape.go)) and routes execution to the appropriate pipeline:

```text
kiw build
  ├── 1. Run Pre-Build Dependencies (build.depends_on in krewire.yaml)
  ├── 2. Shape & Invariant Validation (pkg/model)
  └── 3. Workload Pipeline Execution:
        ├── site:    SSG Template Compiler -> Markdown -> Minification -> .krewire/build
        ├── book:    pkg/book Compiler -> Chapter TOC & Search Index -> .krewire/build
        ├── app:     WASM Client Compilation (wasm/ -> client.wasm) + Go Server Binary -> .krewire/build/bin/<name>
        ├── cli:     Static CLI Binary with Embedded Assets -> .krewire/build/bin/<name>
        ├── worker:  Queue Daemon Binary -> .krewire/build/bin/<name>
        ├── service: Headless Microservice Binary -> .krewire/build/bin/<name>
        ├── gateway: Reverse Proxy Binary -> .krewire/build/bin/<name>
        └── infra:   Blueprint Synthesis -> Topology Graph -> .krewire/build/plan.json
```

### Progressive Fluid Hybrid Compilation
Krewire supports fluid co-located workloads. If an `app`, `service`, or `worker` project also contains a `pages/` directory or static content, `kiw build` builds both:
1. Compiles the Go application binary into `.krewire/build/bin/<name>`.
2. Generates the static web pages into `.krewire/build/`.
3. Records all generated files in `.krewire/build/manifest.json`.

---

## 3. Build Command Flags

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output <dir>` | `-o` | `.krewire/build` | Directory where compiled artifacts are placed. |
| `--target <target>` | | *auto* | Explicitly override build target (`site`, `book`, `binary`, `wasm`, `infra`). |
| `--bundle` | | `true` | Bundle static CSS and JS into `assets/krewire.css` and `assets/krewire.js`. |
| `--minify` | | `false` | Minify bundled CSS and JavaScript assets. |
| `--skip-deps` | | `false` | Skip running `build.depends_on` lifecycle commands. |
| `--deps-only` | | `false` | Run only `build.depends_on` hooks and skip framework compilation. |
| `--plan` | | `false` | For `infra` workloads: compile and display infrastructure plan without deploying. |
| `--env <name>` | | *empty* | Target environment overlay (`local`, `production`, `testing`). |

---

## 4. Production Staging & Deployment (`kiw deploy`)

Stages verified deployable artifacts inside `.krewire/dist/`:
- **Binary Target**: Copies production binary and assets into `.krewire/dist/`.
- **Static Target**: Exports static site bundle ready for CDN distribution.
- **Systemd Target**: Generates production Linux `systemd` service unit definition.
- **Infra Target**: Writes finalized cloud deployment topology and state files.
- Use `--dry-run` to preview staging without mutating remote infrastructure.
