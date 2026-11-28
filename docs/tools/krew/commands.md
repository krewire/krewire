# `krew` CLI Command Reference

Comprehensive command-line reference for the unified developer devtool (`krew`).

---

## Command Groups Overview

`krew help` uses progressive disclosure: the six everyday commands are described
in full, everything else is listed by name under its group. `krew help --all`
(or `krew help -a`) describes every command; `krew help <command>` shows one.

```text
CORE WORKFLOW
  new · dev · build · test · add · dash · make

ADVANCED CAPABILITIES
  project   init · remove · update · migrate · seed · ui
  inspect   version · info · compat · tier · config · doctor · self-update
  build     bundle · serve
  develop   run · vet · fmt · worker · gateway · mcp
  ship      deploy · ws · boost · release
```

Typing a command from another tool prints the kiw equivalent, e.g.
`kiw start` → `krew dev` or `krew run`, `kiw install` → `krew add`,
`kiw generate` → `krew make`, `kiw preview` → `krew serve`, `kiw lint` → `krew vet`.

### Choosing a run mode: `dev` vs `run` vs `serve`

| Command | What it does | Watches files? | Use it for |
|---|---|---|---|
| `krew dev` | Live development server: builds, then rebuilds and hot-reloads a site/book or restarts an app/service on every change | Yes | Day-to-day development |
| `krew run` | Runs once: builds and executes the app or CLI binary, a Go file (`krew run tool.go`), or a `scripts:` task | No | Executing a program or task |
| `krew serve` | Previews production output: serves the built site/book from `.krewire/build` (apps and CLIs run as with `krew run`) | No | Checking a `krew build` result before deploying |

`krew run` on a site or book project explains this and points to `krew dev`, or
to `krew build` followed by `krew serve`.

### Starting a project: `new` vs `init`

| | `krew new <name>` | `krew init` |
|---|---|---|
| Where | Creates a **new directory** `<name>` | Uses the **current directory** (or the path given) |
| Default kind | `app` | none: only `krewire.yaml`, `.gitignore`, `git init` |
| Kind flags | `--app --site --book --cli --worker --service --gateway --infra --template <url>` (same set) | same set |
| Run bare in a terminal | asks for the project name, then the kind | asks for the kind (Enter keeps the minimal setup) |

Both write the same `krewire.yaml` (with the JSON Schema modeline). Prompts
appear only when stdin is a terminal; `--no-input` disables them for scripts.

---

## 1. Project Commands

### `krew new [project-name] [flags]`
Creates the project directory and scaffolds it in one step. The default variant is a fullstack `app`; choose another kind with a variant flag (at most one). Run without a name in a terminal, it asks for the name and the kind.
- `--module <path>`: Go module path (defaults to the project name).
- `--dir <dir>`: Parent directory (defaults to the current directory).
- `--site` / `--book` / `--cli` / `--app` / `--worker` / `--service` / `--gateway` / `--infra`: Variant to scaffold.
- `--template <git-url>`: Clone a remote starter instead of generating files.
- `--title <title>`: Title for the `--site` and `--book` variants.
- `--no-input`: Never prompt (fail with usage when the name is missing).
```bash
krew new                         # interactive: name and kind
krew new my-service
krew new my-site --site --title "My Site"
```

### `krew init [flags]`
Initializes Krewire in the current directory in place: `git init`, `.gitignore`, and a default `krewire.yaml`. Without a variant flag it stops there (in a terminal it first asks which kind to equip; Enter keeps the minimal setup); with one it also equips the project for that kind.
- `--site`: Equips Static Site Generation (SSG) layout.
- `--book`: Equips a `content/` directory for technical books.
- `--cli`: Equips Terminal UI command scaffolding.
- `--app`: Equips a fullstack monolith application.
- `--worker`: Equips a background job-queue worker.
- `--service`: Equips a standalone microservice.
- `--gateway`: Equips an API gateway deployment.
- `--infra`: Equips a cloud infrastructure blueprint.
- `--template <git-url>`: Scaffolds from a remote git template.
- `--force` / `-f`: Reset an existing `krewire.yaml` without prompting.
- `--no-input`: Never prompt for the kind.
```bash
krew init           # interactive: choose a kind or keep the minimal setup
krew init --site
```

### `krew add <package[@version] | ui/<component> | ui:<pack>>`
Installs a Go module or activates an opt-in ecosystem plugin (such as Tailwind CSS, Svelte, or Vue), or copies Krewire UI (kui) components into `./components` (see [`krew ui`](#kiw-ui-subcommand)).
- **Pure Go by Default**: Krewire projects default to pure Go with native CSS design tokens and zero external toolchains.
- **Opt-In Technology**: 
  - Adding `twcss` configures the Tailwind plugin and auto-detects runners (standalone binary in `bin/tailwindcss`, Bun, PNPM, or npm/npx).
  - Adding `svelte` installs Svelte 5, configures `svelte.config.js` and `vite.config.js`, and sets up starter components in `components/`.
  - Adding `vue` installs Vue 3, configures `@vitejs/plugin-vue`, and sets up starter components in `components/`.
```bash
krew add twcss@latest
krew add svelte@latest              # Opt-in modular Svelte 5 integration
krew add vue@latest                 # Opt-in modular Vue 3 integration
krew add ui/button ui/card         # kui components (plus the components they nest)
krew add ui:dashboard              # starter pack: Table, Stat, Sidebar, Modal
krew add github.com/krewire/krewire@v0.2.0
```

### `krew remove <package>`
Uninstalls an external package or deactivates an opt-in plugin.
```bash
krew remove twcss
krew remove svelte
krew remove vue
```

### `krew update [packages...] [flags]`
Updates project dependencies to their latest compatible versions. When run without arguments, updates all direct and indirect dependencies in `go.mod` and tidies the module files.
- `--dry-run`: Previews available dependency updates without modifying `go.mod`.
- `--tidy`: Runs `go mod tidy` after updating (default `true`).
```bash
krew update                      # Update all dependencies in project
krew update --dry-run            # Preview available updates
krew update krewire ui           # Update specific ecosystem packages
krew update github.com/foo/bar   # Update specific external dependency
```

### `krew ui <subcommand>`
Discovers and installs Krewire UI (kui) components, copy-and-own style. The
catalog (props, defaults, dependencies, preview snippets, starter packs) comes
from the `components.json` manifest of the `github.com/krewire/ui` version kiw
was built with.
- `krew ui list [--json]`: catalog by category, starter packs, and the state of each installed component (`up to date`, `modified`, `update available`).
- `krew ui info <name>`: props, class defaults, dependencies and a preview snippet.
- `krew ui add <name|ui:pack>... [--force] [--dry-run]`: same as `krew add ui/<name>`. Dependencies are installed too. Unmodified copies are updated in place; a locally modified copy is kept (exit 1) unless `--force`.
- `krew ui diff <name>`: unified diff from the upstream component to your copy (`git diff --no-index`).

Copies are recorded in `components/.kui-lock.json` (kui version and SHA-256);
commit it with your components. Starter packs: `minimal` (Button, Card, Input),
`base` (Container, Button, Card, Input), `dashboard` (Table, Stat, Sidebar, Modal).
```bash
krew ui list
krew add ui:minimal
krew ui diff Card
```

### `krew make <kind> <name>`
Scaffolds code boilerplate (components, pages, handlers, migrations, models, middleware, services, jobs, chapters, docs). All paths are bounded and validated with `sec.SafePath` to prevent directory traversal escapes. `krew make component <Name>` creates a blank component; to start from a ready-made one, use `krew add ui/<name>`. Compound dotted names (`Card.Header`) scaffold nested subdirectories (`components/Card/Header.kiw`) for seamless `<Card.Header />` compound component invocation.
```bash
krew make component Card            # creates components/Card.kiw
krew make component Card.Header     # creates components/Card/Header.kiw
krew make page about
krew make handler CreateUser
krew make migration create_users_table
krew make model Account
krew make seeder User
krew make factory User
krew make repository User
krew make middleware Auth
krew make service Billing
krew make job SendInvoice
krew make chapter introduction
krew make doc architecture
```

### `kiw migrate <up [n] | down [n] | redo | status | create <name>>`
Applies the SQL files `krew make migration` writes (`migrations/<timestamp>_<name>.sql`, with `-- +up` and `-- +down` sections) using [`orm/migrate`](../../modules/orm.md#migrations-pkgormmigrate).
```bash
kiw migrate create create_users   # same file as krew make migration
kiw migrate up                    # apply every pending migration
kiw migrate up 1                  # apply only the next one
kiw migrate status                # STATE / MIGRATION / APPLIED AT
kiw migrate down                  # roll back the last migration (down 3: the last three)
kiw migrate redo                  # roll back the last migration and apply it again
```
- **Database**: `--dsn`, `--driver`, and `--dir` win over `APP_DATABASE_DSN`, `APP_DATABASE_DRIVER`, and `APP_DATABASE_MIGRATIONS` (`.env` is loaded), which win over the [`database:` section](./configuration.md#database-database) of `krewire.yaml`. The driver defaults to `sqlite` and the directory to `migrations`; a relative SQLite path is resolved from the project root.
- **Drivers**: `krew` links only the pure-Go SQLite driver ([`orm/sqlite`](../../modules/orm.md#sqlite-driver-pkgormsqlite)). For PostgreSQL, import a driver in your app and call `migrate.Up` from it.
- **Safety**: each migration runs in a transaction with its `schema_migrations` row; concurrent runners wait for one another (`--lock-timeout`, default 30s); an applied file that was edited stops `up` and `down`, and `status` exits 1 when an applied file was edited (`modified`) or deleted (`missing`).

### `kiw seed [--class <name>] [--dsn <dsn>] [--driver <driver>] [--dir <dir>]`
Alias: `kiw db:seed`
Seeds the database with records via Go seeders and model factories using [`orm/seeder`](../../modules/orm.md#seeder--model-factory-engine-pkgormseeder).
```bash
kiw seed                       # runs DatabaseSeeder (default)
kiw seed --class UserSeeder    # runs a specific seeder class
kiw db:seed                    # Laravel-like alias
kiw seed --dsn test.db         # override target database DSN
```
- **Database**: Inherits settings from `krewire.yaml` (`database:` section), environment variables (`APP_DATABASE_DSN`, `APP_DATABASE_DRIVER`), or explicit CLI flags.
- **Directory**: Resolves from `seeds/` or `database/seeds/` (overridable via `database.seeds` in `krewire.yaml` or `--dir`).

### `kiw pkg <init <name> | pack [dir] | add <src> | list | export <comp|dir>>`
Manages standalone `.kiw` component packages and distributed UI kits ([ADR-003](../../../../internal/docs/adr/ADR-003-ui-and-terminal-presentation-engine.md)). Enables modular packaging, tar.gz (`.kpkg`) archiving with SHA-256 integrity verification, local installation into `./.krewire/`, and automated style injection with native Go packaging.
```bash
kiw pkg init @acme/ui             # Scaffolds standalone package with components.json
kiw pkg pack ./my-kit              # Packs package into .kpkg distributable archive
kiw pkg add ./dist/ui-1.0.0.kpkg   # Installs package into ./.krewire/
kiw pkg list                       # Lists installed component packages
kiw pkg export Card                # Validates and exports self-contained component
```
- **Manifest (`components.json`)**: Declares package metadata, exported component paths, and package stylesheets (`styles`).
- **Namespace & Aliasing**: Components from installed packages can be invoked using their package namespace (e.g. `<acme:Button />`) or imported directly via `<script> import Button from '@acme/ui'; </script>`.
- **Integrity**: `kiw pkg pack` and `add` enforce SHA-256 checksums and `sec.SafePath` bounds to prevent ZipSlip and directory traversal vulnerabilities.

---

## 2. Inspect Commands

### `krew version`
Prints CLI, ecosystem version, Go runtime, and the verified build signature (`build with krewire`).
```bash
krew version
```

### `krew info`
Inspects and reports complete environment and project metadata:
- **Environment**: CLI version, ecosystem version, verifiable signature, Go version, active environment (`local`, `production`), and debug status.
- **Project**: Working directory, Go module path, Krewire foundation status, workload kind (`app`, `cli`, `site`, `book`, `worker`, `service`, `gateway`, `infra`), and detection marker.
- **Host Ecosystem**: Automatically detects foreign platforms (`Node.js`, `Laravel`, `PHP/Composer`), active package manager (`pnpm`, `yarn`, `npm`, `bun`, `composer`), and framework (`SvelteKit`, `Nuxt`, `Next.js`, `Remix`, `Astro`, `Laravel`).
- **Co-located Krewire**: Discovers and lists any Krewire sub-projects co-located in the directory tree.
- **Ports & Coexistence**: Displays allocated ports across Krewire and foreign runtimes, reporting any conflicting port bindings.
- **Multi-Runtime Build Suggestions**: Detects foreign build scripts (e.g. `npm run build`, `composer install`) and advises on `build.depends_on` orchestration.
```bash
krew info
```

### `krew doctor [--json]`
Diagnoses the developer environment: toolchain prerequisites, workspace layout, and the checks needed before a project can build.
```bash
krew doctor
krew doctor --json   # Machine-readable report for CI
```

### `krew compat`
Audits version compatibility across all imported packages and ecosystem modules.
```bash
krew compat
```

### `krew tier`
Inspects the current project's open-core tier (`free` $\rightarrow$ `pro` $\rightarrow$ `team` $\rightarrow$ `enterprise`) and validates open-core boundary invariants (single source of truth: `internal/docs/open-core.md`).
```bash
krew tier
krew tier --check free  # Assert project does not exceed specified tier
krew tier --json        # Machine-readable output for CI/CD gates
```

### `kiw self-update [flags]`
Checks for and installs the latest stable release of the `krew` devtool binary.
- `--check`: Checks if a newer stable version is available without installing.
- `--force`: Reinstalls or updates even if the tool is already on the latest version.
- `--dry-run`: Simulates the update process without replacing the binary.
- `--version <ver>`: Installs a specific target version.
```bash
kiw self-update          # Update kiw to latest stable release
kiw self-update --check  # Check for updates
kiw self-update --force  # Force reinstall
```

---

## 3. Build & Serve Commands

### `krew build [flags]`
Builds the project according to its detected capabilities:
- `site` / `book` $\rightarrow$ Compiles static assets into `.krewire/build`.
- `app` / `cli` / `service` / `worker` / `gateway` $\rightarrow$ Compiles standalone Go binary with embedded `-ldflags` signature (`build with krewire`).
- `infra` $\rightarrow$ Compiles and plans infrastructure state.

**Flags:**
- `--skip-deps`: Skips pre-build dependency commands declared in `build.depends_on`.
- `--deps-only`: Runs only `build.depends_on` commands without building Krewire artifacts.
- `--plan`: Generates an infrastructure execution plan (`infra` kind).
- `--bundle`: Bundles all eligible CSS and JS assets into `assets/krewire.css` and `assets/krewire.js`.
```bash
krew build
krew build --skip-deps
krew build --plan
krew build --bundle
```

### `krew bundle [flags]`
Bundles static CSS and JS assets into unified `assets/krewire.css` and `assets/krewire.js` bundles, hoisting `@charset` and `@import` rules, and ensuring semicolon isolation.

**Flags:**
- `--output, -o <dir>`: Target output directory (default: `.krewire/build`).
- `--minify, -m`: Enable safe minification for stylesheets and scripts.
```bash
krew bundle
krew bundle --minify
krew bundle -o dist --minify
```

### `krew serve [--addr <addr>]`
Previews production output without watching files: serves the built site/book from `.krewire/build` (run `krew build` first), or runs apps and CLIs as `krew run` does. For live reload while editing, use `krew dev`.
```bash
krew serve --addr :3000
```

---

## 4. Develop Commands

### `krew run [task|file.go] [-- args]`
Runs once, without watching files: executes the application or CLI binary, a standalone Go script, or a named task from `krewire.yaml`. On a site or book project it explains the alternatives (`krew dev`, or `krew build` then `krew serve`).
- **Process Bridge**: Passes through standard input (`os.Stdin`), allowing Node.js `child_process` or Laravel Symfony `Process` to stream JSON IPC directly to/from the running binary.
- **Signature Injection**: Compiles with canonical `build with krewire` ldflags.
```bash
krew run                 # Run project binary
krew run script.go       # Run standalone script with project environment
krew run migrate         # Run named script defined under 'scripts:' in krewire.yaml
```

### `krew dev`
Live development server: builds the project, then watches files and rebuilds with hot reload (sites and books) or restarts the process (apps, services, workers, gateways) on every change.
```bash
krew dev
```

### `krew test [--filter <pattern>]`
Runs Go unit and integration tests:
```bash
krew test
krew test --filter Unit
```

### `krew vet [--sec] [--vuln]`
Runs `go vet`, validates `krewire.yaml` against its JSON Schema (errors fail,
unknown/deprecated keys warn), lints every `.kiw` template (unbalanced
`{#if}`/`{#each}`, control blocks inside component attributes, invalid
frontmatter) with `file:line:col` diagnostics, and optionally runs the security
(`--sec`) and vulnerability (`--vuln`) audits.
```bash
krew vet --sec
```

### `krew config [validate|schema] [path] [--strict] [--json]`
Validates `krewire.yaml` (or `path`) against the canonical JSON Schema and
reports issues with line and column plus did-you-mean suggestions; `--strict`
also fails on warnings. `krew config schema` prints the schema. See
[Configuration](configuration.md#json-schema-and-editor-validation).
```bash
krew config
krew config validate --strict services/api/krewire.yaml
krew config schema > krewire.schema.json
```

### `krew fmt [--write]`
Audits formatting via `gofmt`. With `--write`, automatically formats files.
```bash
krew fmt --write
```

### `kiw worker`
Runs background queue workers in local supervision mode.
```bash
kiw worker
```

### `kiw dash` (alias: `kiw dashboard`)
Opens an interactive local developer dashboard and telemetry server visualizing microservices, background jobs, detected routes, memory metrics, and real-time SSE telemetry pulses, or manages AI-native virtual developer workspaces and isolated execution sandboxes.
```bash
kiw dash
kiw dash --port 4000 --host 127.0.0.1
kiw dash start --sandbox=go-1.27
kiw dash list
kiw dash status
kiw dash console
kiw dash stop
```

**Options & Subcommands:**
- `kiw dash` / `--port`, `--host`: Launch local dev dashboard & MCP telemetry server (default port `4000`).
- `start [--sandbox=<image>]`: Mount and attach to an isolated developer workspace sandbox.
- `list`: List active developer workspaces and sandboxes.
- `status`: Inspect Dash server connectivity, quota, and health.
- `console`: Open or print Web IDE console URL.
- `stop [id]`: Stop a running sandbox.

**Built-in Telemetry APIs:**
- `GET /`: Dark-themed Web Dashboard UI with live stats and SSE event feed.
- `GET /api/info`: JSON metadata for project, framework version, services, and routes.
- `GET /api/metrics`: Live memory allocation, system memory, goroutine count, and GC cycles.
- `GET /api/health`: Standardized health check response.
- `GET /api/events`: Real-time Server-Sent Events (SSE) pulse stream (`text/event-stream`).
- `GET /mcp/sse`: Native Model Context Protocol (MCP) SSE endpoint for AI coding agents.
- `POST /mcp/message`: Native MCP JSON-RPC message endpoint.

### `kiw gateway`
Runs the Krewire API Gateway reverse proxy for microservices, routing inbound HTTP requests to upstream services via local service discovery or static URLs with built-in tracing, rate limiting, and CORS.
```bash
kiw gateway
kiw gateway --port 8080 --host 0.0.0.0
kiw gateway --routes "/api/auth=auth-service,/api/pay=https://payment.internal"
```

**Options:**
- `--port`: Listen port for the API gateway (default: `8080`, or `gateway.port` in `krewire.yaml`).
- `--host`: Listen host for the API gateway (default: `0.0.0.0`, or `gateway.host` in `krewire.yaml`).
- `--routes`: Comma-separated route mapping (e.g. `/api/auth=auth-service,/api/pay=https://payment.internal`).

**Gateway Features:**
- Reads route rules and timeouts directly from `gateway:` in `krewire.yaml`.
- Propagates W3C `traceparent` headers and correlation IDs across upstream services.
- Integrates with `cloud/service/registry` for dynamic service resolution.

### `kiw mcp`
Starts a native Model Context Protocol (MCP) server for AI coding agents (Antigravity, Cursor, Claude Desktop, OpenCode). Exposes project metadata, routes, runtime metrics, health probes, and scaffolding tools directly to agents over stdio or HTTP/SSE.
```bash
kiw mcp --stdio                      # Run over stdio (standard for AI IDEs)
kiw mcp --port 4000 --host 127.0.0.1 # Run standalone over HTTP/SSE
```

**Options:**
- `--stdio`: Run MCP server over standard input/output (default: `true`).
- `--port`: Run MCP server over HTTP/SSE on the specified port.
- `--host`: Bind host address for HTTP/SSE (default: `127.0.0.1`).

**Supported Resources & Tools:**
- **Resources**: `krewire://project/info`, `krewire://project/routes`, `krewire://runtime/metrics`, `krewire://runtime/health`.
- **Tools**: `kiw_inspect_project`, `kiw_list_routes`, `kiw_make_scaffold`, `kiw_get_metrics`, `kiw_run_tests`.
- **Dash Integration**: `kiw dash` (or `kiw dashboard`) also mounts MCP at `http://127.0.0.1:4000/mcp/sse` and `http://127.0.0.1:4000/mcp/message`.

---

## 5. Ship Commands

### `krew deploy [--dry-run]`
Stages production artifacts into `.krewire/dist/` and runs deployment targets (`--target binary|systemd|gh-pages|infra`).
```bash
krew deploy --dry-run
krew deploy
```

### `kiw ws <subcommand>`
Manages multi-module workspaces, monorepos, and microservice topologies.
```bash
kiw ws info
```

### `krew boost <install|status>`
Manages the Boost AI Agent constitution, multi-agent presets, and tooling inside `.agents/`.
```bash
krew boost install            # Install Boost templates into project
krew boost status             # Inspect installation completeness and managed assets
krew boost install --force    # Overwrite/repair existing assets without prompt
```

### `krew release <module> --bump <patch|minor|major> [--apply]`
Bumps semver versions and coordinates multi-module releases. Module names come from the release manifest (`krew/internal/release`); `--all` releases every module. Without `--apply` the command is a dry run.
```bash
krew release krewire --bump minor --apply
krew release --all --bump patch
```
