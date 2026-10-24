# Configuration Reference — `krewire.yaml` Specification

## Description
The `krewire.yaml` file is the Single Source of Truth (SSOT) for configuring project metadata, workload kind, multi-runtime coexistence, port assignments, server settings, custom scripts, and build dependencies across all Krewire projects. The schema is owned by [`tools/kiw/internal/config/config.go`](../../../tools/kiw/internal/config/config.go).

---

## Minimal Configuration

Every Krewire project requires a minimal `krewire.yaml` defining the project name and workload kind:

```yaml
version: "0.1.0"
project:
  name: my-app
  kind: app
```

---

## Full Schema Reference

A complete `krewire.yaml` schema with all supported options and multi-runtime controls:

```yaml
version: "0.1.0"          # Top-level product version (injected into builds)

# Project metadata & workload invariants
project:
  name: my-service          # Required: kebab-case ^[a-z][a-z0-9-]*$
  kind: service             # app | cli | site | book | worker | service | gateway | infra
  version: "v0.1.0"         # Alternative version location (folds into top-level version)
  dirs:
    cmd: "cmd/app"          # Custom entrypoint directory
    internal: "internal"    # Internal package directory
    web: "web"              # Web templates & components directory
    public: "public"        # Static public assets directory

# Environment & diagnostics
env: local                  # local | production | testing
debug: false

# Shared site/book metadata
title: "My Service"
author: "Krewire Contributors"
base: "/"                   # URL base prefix
input: "content"            # Content directory (default: content)
output: ".krewire/build"    # Output directory (default: .krewire/build)

# Light/dark theme switcher (shared by site and book)
theme:
  default: "auto"           # auto | light | dark
  light:
    primary: "#7c3aed"
  dark:
    primary: "#a78bfa"

# Automatic CSS/JS injection and bundling (top-level; ssg.auto_assets wins when both are set)
auto_assets:
  enabled: true
  exclude: ["*.min.css"]
  js_placement: "head"      # head | body
  bundle: true

# Build pipeline & foreign dependency orchestration (KWN-BLD5Y)
build:
  base: "/"
  output: ".krewire/build"
  include: ["**/*.md"]      # Content globs; unset uses book defaults
  exclude: ["**/README.md"] # Removed after include filtering
  depends_on:               # Ordered pre-build shell commands
    - "npm run build"
    - "go generate ./..."
  outputs:                  # Runtime name -> output directory
    nodejs: "dist"

# Development server
dev:
  port: 8080

# Primary HTTP server settings (for app, service, site)
server:
  host: "0.0.0.0"
  port: 8080                # Default port (must be between 1 and 65535)

# Multi-runtime coordination & environment binding (KWN-MRCN, KWN-Q7X4M)
runtime:
  namespace: "api"          # Prefix for environment variables: KREWIRE_API_<KEY>
  env_file: ".env.local"    # Custom dotenv file path (defaults to .env)
  port: 8080                # Default runtime port

# Single microservice descriptor (kind: service)
service:
  name: "auth"
  port: 8081

# Co-located microservices with distinct port allocations (FRK-CONF-032)
services:
  - name: "auth"
    port: 8081
  - name: "billing"
    port: 8082

# API gateway reverse proxy configuration (KWF-L5H2F)
gateway:
  host: "0.0.0.0"
  port: 8080
  routes:
    - path: "/api/auth"
      service: "auth"
      timeout: "5s"
    - path: "/api/billing"
      service: "billing"
      strip_prefix: true
      rate_limit: 100
    - path: "/api/external"
      target_url: "https://api.external.com"
      method: "GET"
      prefix: true
      middleware: ["cors"]

# Internationalization settings (kind: site, book)
i18n:
  default_locale: "en"
  fallback_locale: "en"
  base_path: "lang"
  locales: ["en", "id"]

# SSG static site settings (kind: site)
ssg:
  description: "Product documentation"
  layouts: []               # {name, body, style} or {name, ui}
  components: []            # {name, body, style}
  pages: []                 # Declarative output pages
  assets: {}                # Output path -> file contents
  auto_assets:              # Optional override of the top-level block
    enabled: true
    bundle: true

# Technical book settings (kind: book)
# Title, author, input, output, base, version, nav, footer, and theme are
# shared top-level keys. The book block only controls placement and TOC.
book:
  mount: "/docs"            # Export the book under a sub-path
  toc: true                 # Force the generated root table of contents

# Background worker settings (kind: worker)
worker:
  concurrency: 4
  queues: ["default", "critical"]

# WASM client build (KWF-T4X9P)
wasm:
  entry: "./wasm"           # Main package path (default: ./wasm)
  name: "runtime"           # Output module name (default: runtime)

# Cloud infrastructure blueprint (kind: infra, KWF-B7N3D)
infra:
  provider: "aws"
  region: "ap-southeast-1"
  state: ".krewire/infra/state.json"
  resources:
    - kind: "service"
      id: "api"
      properties:
        image: "krewire/api:latest"

# Custom project tasks executable via `kiw run <task>`
scripts:
  migrate: "go run ./cmd/migrate"
  seed: "go run ./cmd/seed"
  lint: "golangci-lint run"
  test-e2e: "go test -tags=e2e ./..."
```

> The `ssg:` block carries only the keys `tools/kiw/internal/config` parses: `description`, `layouts`, `components`, `pages`, `assets`, and `auto_assets`. Content collections, asset pipelines, and draft/future toggles exist on `pkg/web/ssg.Config` for programmatic callers; `kiw` does not read them from `krewire.yaml`. A file-based site (`pages/`, `layouts/`, `components/`, `public/`) needs no `ssg:` block at all.

---

## Multi-Runtime & Environment Coordination

Krewire coordinates cleanly with other runtimes (Node.js, Laravel/PHP) in the same directory:

### 1. Dotenv Loading & Precedence
- `kiw` loads `.env` files automatically on boot (`kiw run`, `kiw dev`, `kiw build`).
- If `runtime.env_file` is specified, that file is loaded with higher precedence over `.env`.
- If the target env file is missing but `.env.example` exists, `kiw` logs a warning and scaffolds a template `.env` (`KWL-CFGV-005`).

### 2. Namespace Prefixing & Synchronization
When `runtime.namespace` is declared (e.g. `namespace: "api"`):
- All keys are mirrored into the prefixed form: `PORT=8080` is synchronized to `KREWIRE_API_PORT=8080`.
- Conversely, external systems setting `KREWIRE_API_DATABASE_URL` will have it available to the application as both `KREWIRE_API_DATABASE_URL` and `DATABASE_URL`.
- This eliminates variable collisions between Node.js/Laravel host applications and co-located Go services.

### 3. Port Allocation & Conflict Detection
- `kiw info` and runtime boot (`kiw dev`, `kiw run`) automatically detect ports declared across:
  - `krewire.yaml` (`runtime.port`, `dev.port`, `server.port`, `services[].port`)
  - Node.js `package.json` (scripts containing `--port`, `PORT=`)
  - Laravel/PHP `composer.json` or `artisan serve` defaults (:8000)
  - Active `.env` files (`PORT=`, `APP_PORT=`, `SERVER_PORT=`)
- If two services or runtimes claim the same port without explicit disambiguation, `kiw` warns of the conflict immediately (`FRK-CONF-030`).
- The `--addr` command-line flag (`kiw run --addr :9000`) always takes precedence over all configuration and environment declarations.

---

## Build Orchestration (`build.depends_on`)

When building fullstack applications composed of a Node.js frontend and a Go backend, declare dependencies in `build.depends_on`:
- Commands are executed sequentially in the project root prior to Go compilation.
- If any command fails with a non-zero exit code, the build aborts immediately with that exact exit code.
- Control flags:
  - `kiw build --skip-deps`: Skips pre-build dependencies and builds only Krewire targets.
  - `kiw build --deps-only`: Executes pre-build dependency scripts without running Krewire compiler.

---

## Quick References

- [`kiw` Commands Reference](./commands.md) — CLI flags and operational commands.
- [Development & Build Pipelines](./dev-and-build.md) — Compilation, dev watch mode, and artifact staging.
- [Multi-Runtime Coexistence Guide](../../guides/multi-runtime-coexistence.md) — Node.js and Laravel architectural patterns.
