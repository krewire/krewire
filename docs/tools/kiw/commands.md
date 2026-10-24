# `kiw` CLI Command Reference

Comprehensive command-line reference for the unified developer devtool (`kiw`).

---

## Command Groups Overview

```text
USAGE
  kiw <command> [options]
  kiw help [command]
```

---

## 1. Project Commands

### `kiw new <project-name>`
Scaffolds a minimal Krewire kernel project with `go.mod`, `krewire.yaml`, and `main.go`.
```bash
kiw new my-service
```

### `kiw init [flags]`
Equips the current project with a specific workload kind in-place.
- `--site`: Equips Static Site Generation (SSG) layout.
- `--book`: Equips manuscript directory for technical books.
- `--cli`: Equips Terminal UI command scaffolding.
- `--template <name>`: Scaffolds from a starter template.
```bash
kiw init --site
```

### `kiw add <package@version>`
Adds or updates a domain package or plugin in the project.
```bash
kiw add twcss@latest
```

### `kiw remove <package>`
Removes a package or plugin from the project.
```bash
kiw remove twcss
```

### `kiw make <kind> <name>`
Scaffolds code boilerplate (components, pages, handlers, migrations, models, middleware, services).
```bash
kiw make component Card
kiw make page about
kiw make handler CreateUser
kiw make migration create_users_table
kiw make model Account
kiw make middleware Auth
kiw make service Billing
```

---

## 2. Inspect Commands

### `kiw version`
Prints CLI, ecosystem version, Go runtime, and the verified build signature (`build with krewire`).
```bash
kiw version
```

### `kiw info`
Inspects and reports complete environment and project metadata:
- **Environment**: CLI version, ecosystem version, verifiable signature, Go version, active environment (`local`, `production`), and debug status.
- **Project**: Working directory, Go module path, Krewire foundation status, workload kind (`app`, `cli`, `site`, `book`, `worker`, `service`, `infra`, `kernel`), and detection marker.
- **Host Ecosystem**: Automatically detects foreign platforms (`Node.js`, `Laravel`, `PHP/Composer`), active package manager (`pnpm`, `yarn`, `npm`, `bun`, `composer`), and framework (`SvelteKit`, `Nuxt`, `Next.js`, `Remix`, `Astro`, `Laravel`).
- **Co-located Krewire**: Discovers and lists any Krewire sub-projects co-located in the directory tree.
- **Ports & Coexistence**: Displays allocated ports across Krewire and foreign runtimes, reporting any conflicting port bindings.
- **Multi-Runtime Build Suggestions**: Detects foreign build scripts (e.g. `npm run build`, `composer install`) and advises on `build.depends_on` orchestration.
```bash
kiw info
```

### `kiw compat`
Audits version compatibility across all imported packages and ecosystem modules.
```bash
kiw compat
```

### `kiw tier`
Inspects the current project's open-core tier (`free` $\rightarrow$ `pro` $\rightarrow$ `team` $\rightarrow$ `enterprise`) and validates open-core boundary invariants (single source of truth: `internal/docs/open-core.md`).
```bash
kiw tier
kiw tier --check free  # Assert project does not exceed specified tier
kiw tier --json        # Machine-readable output for CI/CD gates
```

---

## 3. Build & Serve Commands

### `kiw build [flags]`
Builds the project according to its detected kind:
- `site` / `book` $\rightarrow$ Compiles static assets into `.krewire/build`.
- `app` / `cli` / `service` / `worker` $\rightarrow$ Compiles standalone Go binary with embedded `-ldflags` signature (`build with krewire`).
- `infra` $\rightarrow$ Compiles and plans infrastructure state.

**Flags:**
- `--skip-deps`: Skips pre-build dependency commands declared in `build.depends_on`.
- `--deps-only`: Runs only `build.depends_on` commands without building Krewire artifacts.
- `--plan`: Generates an infrastructure execution plan (`infra` kind).
- `--bundle`: Bundles all eligible CSS and JS assets into `assets/krewire.css` and `assets/krewire.js`.
```bash
kiw build
kiw build --skip-deps
kiw build --plan
kiw build --bundle
```

### `kiw bundle [flags]`
Bundles static CSS and JS assets into unified `assets/krewire.css` and `assets/krewire.js` bundles, hoisting `@charset` and `@import` rules, and ensuring semicolon isolation.

**Flags:**
- `--output, -o <dir>`: Target output directory (default: `.krewire/build`).
- `--minify, -m`: Enable safe minification for stylesheets and scripts.
```bash
kiw bundle
kiw bundle --minify
kiw bundle -o dist --minify
```

### `kiw serve [--port <port>]`
Serves static site preview over HTTP with WebSocket-driven live-reload (HMR), or starts local server for web apps.
```bash
kiw serve --port 3000
```

---

## 4. Develop Commands

### `kiw run [task|file.go] [-- args]`
Executes an application, a standalone Go script, or a named task from `krewire.yaml`:
- **Process Bridge**: Passes through standard input (`os.Stdin`), allowing Node.js `child_process` or Laravel Symfony `Process` to stream JSON IPC directly to/from the running binary.
- **Signature Injection**: Compiles with canonical `build with krewire` ldflags.
```bash
kiw run                 # Run project binary
kiw run script.go       # Run standalone script with project environment
kiw run migrate         # Run named script defined under 'scripts:' in krewire.yaml
```

### `kiw dev`
Boots the application in development mode with watch-and-restart on file modifications.
```bash
kiw dev
```

### `kiw test [--filter <pattern>]`
Runs Go unit and integration tests:
```bash
kiw test
kiw test --filter Unit
```

### `kiw vet [--sec]`
Runs `go vet` and static security audits across the codebase.
```bash
kiw vet --sec
```

### `kiw fmt [--write]`
Audits formatting via `gofmt`. With `--write`, automatically formats files.
```bash
kiw fmt --write
```

### `kiw worker`
Runs background queue workers in local supervision mode.
```bash
kiw worker
```

### `kiw dashboard`
Opens an interactive local developer dashboard and telemetry server visualizing microservices, background jobs, detected routes, memory metrics, and real-time SSE telemetry pulses.
```bash
kiw dashboard
kiw dashboard --port 4000 --host 127.0.0.1
```

**Options:**
- `--port`: HTTP listen port (default: `4000`).
- `--host`: Bind host address (default: `127.0.0.1`).
- `--env`: Override runtime environment (`local`, `production`, `testing`).
- `--kind`: Force project kind detection (`app`, `site`, `service`, `worker`, etc.).

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
- Integrates with `packages/cloud/service/registry` for dynamic service resolution.

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
- **Dashboard Integration**: `kiw dashboard` also mounts MCP at `http://127.0.0.1:4000/mcp/sse` and `http://127.0.0.1:4000/mcp/message`.

### `kiw krew`
The Krewire in-CLI autonomous AI coding companion (analogous to `gh copilot`). Embeds the platform-agnostic Krew agent engine directly into your terminal workflow.
```bash
# Suggest terminal commands or code snippets (like 'gh copilot suggest')
kiw krew suggest "how to build site with minified bundle"

# Explain commands, error messages, or code structures (like 'gh copilot explain')
kiw krew explain "kiw bundle --minify"

# Interactive terminal chat session
kiw krew chat

# Quick prompt execution / headless automation
kiw krew "inspect the workspace and add unit tests for pkg/assets"
kiw krew run "refactor pkg/model to add status enum" --persona=coding

# Background HTTP/SSE server or MCP stdio daemon
kiw krew serve --port 8080
kiw krew mcp
```

**Subcommands & Modes:**
- `suggest <task>`: Suggests exact commands or code snippets with clear explanations.
- `explain <cmd|code>`: Explains syntax, flags, potential side effects, and Krewire ecosystem context.
- `chat`: Starts an interactive coding assistant REPL in the terminal.
- `run <prompt>`: Executes multi-turn autonomous coding loops with tool sandboxing.
- `serve`: Launches background HTTP/REST and Server-Sent Events (SSE) daemon.
- `mcp`: Launches standard Model Context Protocol (stdio) server.

**Options:**
- `--persona`: Agent role definition (`coding`, `devops`, `worker`, `general`) (default: `coding`).
- `--model`: LLM model name override (e.g. `gemini-2.5-flash`, `gpt-4o`, `claude-3-7-sonnet`).
- `--max-steps`: Maximum reasoning-action loop iterations (default: `25`).
- `--dir`: Workspace directory root (default: `.`).
- `--port`: Listen port for HTTP daemon mode (default: `8080`).
- `--host`: Listen host for HTTP daemon mode (default: `0.0.0.0`).

---

## 5. Ship Commands

### `kiw deploy [--preview]`
Stages production artifacts into `dist/` and runs cloud deployment runners.
```bash
kiw deploy --preview
kiw deploy
```

### `kiw ws <subcommand>`
Manages multi-module workspaces, monorepos, and microservice topologies.
```bash
kiw ws status
```

### `kiw boost <install|status>`
Manages the Boost AI Agent constitution, multi-agent presets, and tooling inside `.agents/`.
```bash
kiw boost install            # Install Boost templates into project
kiw boost status             # Inspect installation completeness and managed assets
kiw boost install --force    # Overwrite/repair existing assets without prompt
```

### `kiw release <module> --bump <patch|minor|major> --apply`
Bumps semver versions and coordinates multi-module releases.
```bash
kiw release framework --bump minor --apply
```

---

## 6. Package & Plugin Commands (Opt-In)

### `kiw add <package[@version]>`
Installs a Go module or activates an opt-in ecosystem plugin (such as Tailwind CSS).
- **Zero JS by Default**: Krewire projects default to pure Go with native CSS design tokens and zero external toolchains.
- **Opt-In Technology**: Adding `twcss` configures the Tailwind plugin and auto-detects runners (standalone binary in `bin/tailwindcss`, Bun, PNPM, or npm/npx).
```bash
kiw add twcss@latest
kiw add tailwindcss
kiw add github.com/krewire/krewire@v0.1.0
```

### `kiw remove <package>`
Uninstalls an external package or deactivates an opt-in plugin.
```bash
kiw remove twcss
```
