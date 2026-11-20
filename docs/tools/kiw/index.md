# Krewire Devtool (`kiw`)

The unified command-line toolchain powering every phase of the Krewire digital software development lifecycle (SDLC).

---

## 1. Overview & Toolchain Philosophy

Rather than juggling fragmented bash scripts, npm build runners, webpack/vite configs, and separate CLI utilities, Krewire developers use a single standalone binary: `kiw`.

```text
kiw <command> [flags]
```

`kiw` dogfoods the monorepo's own Terminal UI toolkit ([`pkg/tui`](../../pkg/web-runtime/ui-tui.md)), declarative models ([`pkg/model`](../../pkg/core/model.md)), and supervisor lifecycle standards ([`pkg/lifecycle`](../../pkg/lifecycle.md)).

```text
Go Stdlib ──► pkg/* ──► tools/kiw (Layer 3) ──► krewire.yaml ──► 8 Workload Runtimes
```

---

## 2. Command Taxonomy & Categories

`kiw` groups commands into cohesive operational categories:

### A. Project Scaffolding & Initialization (`project`)
- **`kiw new <name> [--kind]`**: Generates a new project directory with standard layout, starter code, and `krewire.yaml`.
- **`kiw init [--kind]`**: In-place initialization of existing directories without clobbering foreign files.
- **`kiw boost install [dir]`**: Equips the repository with 21 autonomous AI agents, slash commands, and development constitution.

### B. Development & Iteration (`develop`)
- **`kiw dev`**: Starts live file watching, instant Go recompilation, and WebSocket Hot Module Replacement (HMR).
- **`kiw serve`**: Serves static SSG artifacts from `.krewire/build/` with live-reload previews.
- **`kiw run [task]`**: Executes declared script tasks from `krewire.yaml` (replaces `make` / `npm run`).

### C. Build & Staging (`build`)
- **`kiw build`**: Multi-workload compiler producing single static binaries or static site bundles.
- **`kiw bundle`**: Bundles, tree-shakes, fingerprints, and minifies CSS and client JavaScript.
- **`kiw deploy`**: Deploys verified artifacts to systemd, Docker, GitHub Pages, or cloud infrastructure blueprints.

### D. Quality Gates & Validation (`inspect`)
- **`kiw fmt`**: Checks and formats Go code (`gofmt -l`).
- **`kiw vet`**: Runs Go static analysis (`go vet ./...`).
- **`kiw test`**: Executes unit, integration, and table-driven tests.
- **`kiw info`**: Inspects active environment, workload shape, detected tools, and co-located stacks.
- **`kiw compat`**: Validates semantic versioning and backward compatibility invariants.

### E. Package & Plugin Management (`package`)
- **`kiw add <package>`**: Installs Go packages, opt-in plugins (Tailwind, Svelte, Vue), or vendor UI components.
- **`kiw remove <package>`**: Uninstalls packages or deactivates modular plugins.
- **`kiw update`**: Updates dependencies according to semantic version constraints.

---

## 3. Quick Start

```bash
# 1. Scaffold a new fullstack web application
kiw new blog --app
cd blog

# 2. Start local development server with HMR
kiw dev

# 3. Add an opt-in UI component or plugin
kiw add ui/button
kiw add svelte

# 4. Verify quality gates
kiw fmt
kiw test

# 5. Build production binary
kiw build --minify
```

---

## 4. Documentation Sections

1. **[Command Reference (`kiw commands`)](./commands.md)**: Exhaustive manual for every command, argument, and flag.
2. **[Configuration Specification (`krewire.yaml`)](./configuration.md)**: Full schema reference, environment overlays, and script runner syntax.
3. **[Development & Build Pipelines](./dev-and-build.md)**: Watch daemon, process supervision, WASM compilation, and artifact staging.
4. **[Unified Project Lifecycle & Contracts](./lifecycle-and-contracts.md)**: The 5-stage lifecycle from init to deploy and fluid hybrid workloads.
