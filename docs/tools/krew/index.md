# Krewire Toolchain & AI Agent Engine (`krew`)

The unified command-line toolchain powering every phase of the Krewire digital software development lifecycle (SDLC).

---

## 1. Overview & Toolchain Philosophy

Rather than juggling fragmented bash scripts, npm build runners, webpack/vite configs, and separate CLI utilities, Krewire developers use a single standalone binary: `krew`.

```text
krew <command> [flags]
```

`krew` dogfoods the ecosystem's Terminal UI toolkit ([`tui`](https://github.com/krewire/tui)), declarative models ([`pkg/model`](../../pkg/core/model.md)), and supervisor lifecycle standards ([`pkg/lifecycle`](../../pkg/lifecycle.md)).

```text
Go Stdlib ──► pkg/* ──► krew (Layer 3) ──► krewire.yaml ──► 8 Workload Runtimes
```

---

## 2. Command Taxonomy & Categories

`krew` groups commands into cohesive operational categories:

### A. Project Scaffolding & Initialization (`project`)
- **`krew new <name> [--kind]`**: Generates a new project directory with standard layout, starter code, and `krewire.yaml`.
- **`krew init [--kind]`**: In-place initialization of existing directories without clobbering foreign files.
- **`krew boost install [dir]`**: Equips the repository with 21 autonomous AI agents, slash commands, and development constitution.

### B. Development & Iteration (`develop`)
- **`krew dev`**: Starts live file watching, instant Go recompilation, and WebSocket Hot Module Replacement (HMR).
- **`krew serve`**: Serves static SSG artifacts from `.krewire/build/` with live-reload previews.
- **`krew run [task]`**: Executes declared script tasks from `krewire.yaml` (replaces `make` / `npm run`).

### C. Build & Staging (`build`)
- **`krew build`**: Multi-workload compiler producing single static binaries or static site bundles.
- **`krew bundle`**: Bundles, tree-shakes, fingerprints, and minifies CSS and client JavaScript.
- **`krew deploy`**: Deploys verified artifacts to systemd, Docker, GitHub Pages, or cloud infrastructure blueprints.

### D. Quality Gates & Validation (`inspect`)
- **`krew fmt`**: Checks and formats Go code (`gofmt -l`).
- **`krew vet`**: Runs Go static analysis (`go vet ./...`).
- **`krew test`**: Executes unit, integration, and table-driven tests.
- **`krew info`**: Inspects active environment, workload shape, detected tools, and co-located stacks.
- **`krew compat`**: Validates semantic versioning and backward compatibility invariants.

### E. Package & Plugin Management (`package`)
- **`krew add <package>`**: Installs Go packages, opt-in plugins (Tailwind, Svelte, Vue), or vendor UI components.
- **`krew remove <package>`**: Uninstalls packages or deactivates modular plugins.
- **`krew update`**: Updates dependencies according to semantic version constraints.

---

## 3. Quick Start

```bash
# 1. Scaffold a new fullstack web application
krew new blog --app
cd blog

# 2. Start local development server with HMR
krew dev

# 3. Add an opt-in UI component or plugin
krew add ui/button
krew add svelte

# 4. Verify quality gates
krew fmt
krew test

# 5. Build production binary
krew build --minify
```

---

## 4. Documentation Sections

1. **[Command Reference (`krew commands`)](./commands.md)**: Exhaustive manual for every command, argument, and flag.
2. **[Configuration Specification (`krewire.yaml`)](./configuration.md)**: Full schema reference, environment overlays, and script runner syntax.
3. **[Development & Build Pipelines](./dev-and-build.md)**: Watch daemon, process supervision, WASM compilation, and artifact staging.
4. **[Unified Project Lifecycle & Contracts](./lifecycle-and-contracts.md)**: The 5-stage lifecycle from init to deploy and fluid hybrid workloads.
