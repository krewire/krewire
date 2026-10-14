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

### `kiw generate --kind <kind> --name <name>`
Generates code boilerplate (handlers, services, migrations, specs).
```bash
kiw generate --kind handler --name CreateUser
```

---

## 2. Inspect Commands

### `kiw version`
Prints CLI, framework, and Go runtime version information.
```bash
kiw version
```

### `kiw info`
Prints detected project kind, configuration details, scripts, and environment summary.
```bash
kiw info
```

### `kiw compat`
Audits version compatibility across all imported packages and ecosystem modules.
```bash
kiw compat
```

---

## 3. Build & Serve Commands

### `kiw build [flags]`
Builds the project according to its detected kind:
- `site` / `book` $\rightarrow$ Compiles static assets into `.krewire/build`.
- `app` / `cli` / `service` / `worker` $\rightarrow$ Compiles standalone Go binary into `bin/`.
- `infra` $\rightarrow$ Compiles and plans infrastructure state.
```bash
kiw build
kiw build --plan
```

### `kiw serve [--port <port>]`
Serves static site preview over HTTP with WebSocket-driven live-reload (HMR).
```bash
kiw serve --port 3000
```

---

## 4. Develop Commands

### `kiw run [task|file.go] [-- args]`
Executes an application, a standalone Go script, or a named task from `krewire.yaml`:
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

### `kiw test [--spec <SpecID>]`
Runs Go unit, integration, and spec-driven tests:
```bash
kiw test
kiw test --spec KWL-KERN-X8P3L
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
Opens local developer dashboard visualizing microservices, background jobs, logs, and traces.
```bash
kiw dashboard
```

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

### `kiw boost install` (Alias: `kiw guild install`)
Installs the Boost AI Agent constitution and multi-agent workflows into `.agents/`.
```bash
kiw boost install
```

### `kiw release <module> --bump <patch|minor|major> --apply`
Bumps semver versions and coordinates multi-module releases.
```bash
kiw release framework --bump minor --apply
```
