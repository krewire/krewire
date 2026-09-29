---
title: "Configuration"
description: "Detailed specification and guide for krewire.yaml — the unified declarative devtool configuration."
date: "2026-09-29"
---

# Configuration

Every Krewire project is driven by a single, declarative configuration file: **`krewire.yaml`**.

`krewire.yaml` acts as the **Single Source of Truth (SSOT)** for project metadata, compiler pipeline settings, local development servers, and custom automation scripts.

---

## 1. Architectural Philosophy

Krewire maintains a strict separation of concerns between devtool configuration and application runtime logic:

1. **Devtool Controls (`krewire.yaml`):** Instructs the `kiw` CLI on how to build, run, test, and package the project.
2. **Application Settings:** Web titles, layouts, meta tags, and component styling belong inside pages, templates, and Go code—never duplicated in YAML.
3. **Decoupled Toolchain:** Projects compile cleanly with standard `go build` even if `krewire.yaml` is absent, falling back to sensible Go conventions.

---

## 2. Complete Schema Reference

Below is the complete, canonical schema of `krewire.yaml` with production defaults:

```yaml
# =============================================================================
# Krewire Devtool Configuration (krewire.yaml)
# =============================================================================

# Project Identity & Workload Kind
project:
  name: "my-service"            # Unique project or module identifier
  kind: "site"                  # Workload: app | cli | site | book | worker | service | infra | runtime
  version: "v0.1.0"             # Semantic version string
  author: "Engineering Team"    # Author or organization
  dirs:                         # Optional: Override canonical folder locations
    web: "web"                  # Web handlers and templates
    public: "public"            # Raw static assets
    internal: "internal"        # Private domain packages
    cmd: "cmd"                  # Executable entry points

# Devtool Build Pipeline
build:
  output: ".krewire/build"      # Target compilation output directory
  base: "/"                     # URL base the site is served under
  include:                      # Glob patterns for content inclusion
    - "**/*.md"
  exclude:                      # Glob patterns for exclusion
    - "**/README.md"
    - "**/readme.md"

# Development Server
dev:
  port: "8080"                  # Local HTTP listening port for `kiw dev`

# Documentation Book Pipeline (mdbind)
book:
  mount: "/docs/"               # Mount path in output URL space (hybrid mode)
  toc: true                     # Generate root table-of-contents page

# WebAssembly Client Runtime (KWF-T4X9P)
wasm:
  entry: "./wasm"               # Main package entry point for GOOS=js
  name: "runtime"               # Emitted wasm module filename

# Custom Task Runner Scripts
scripts:
  lint: "golangci-lint run ./..."
  fmt: "gofmt -s -w ."
  test: "go test -race -v ./..."
  seed: "go run cmd/seed/main.go"
```

---

## 3. Configuration Breakdown

### 3.1 Project Block (`project:`)

The `project:` section defines the identity and workload behavior:

- **`kind` (Required):** Pins the architectural workload. This instructs `kiw` which engine to invoke:
  - `app`: Fullstack web monolith with embedded assets (`kiw run`)
  - `cli`: Terminal command-line tool or TUI (`kiw run`)
  - `site`: Scoped `.kiw` component static site generator (`kiw build`)
  - `book`: Markdown documentation manuscript via `mdbind` (`kiw build --target book`)
  - `worker`: Asynchronous background task processor (`kiw worker`)
  - `service`: High-throughput microservice (`kiw run`)
  - `infra`: Infrastructure as code in Go (`kiw deploy --target infra`)
  - `runtime`: Go WebAssembly browser runtime (`kiw build --target wasm`)
- **`dirs` (Optional):** Allows customization of the default directory layout if integrating Krewire into an existing repository layout.

---

### 3.2 Build Block (`build:`)

Controls asset compilation and artifact emission:

- **`output`:** The directory where compiled static files or assets are staged. Defaults to `.krewire/build`.
- **`base`:** The URL prefix under which assets and pages are served. Defaults to `/`. For sites hosted under subpaths (e.g. `https://example.com/blog/`), set `base: "/blog/"`.
- **`include` & `exclude`:** Glob patterns controlling which Markdown manuscripts or content files are processed.

---

### 3.3 Dev Server Block (`dev:`)

Configures local hot-reloading server behavior:

- **`port`:** Sets the TCP port `kiw dev` and `kiw serve` bind to (default: `8080`).
- You can override this at runtime with the `--addr` flag:
  ```bash
  kiw dev --addr :3000
  ```

---

### 3.4 Scripts Task Runner (`scripts:`)

Krewire includes an integrated task runner, eliminating the need for `Makefile` or external runners. Any task declared in `scripts:` can be executed directly via `kiw run <task>`:

```yaml
scripts:
  lint: "golangci-lint run ./..."
  audit: "go run cmd/security-audit/main.go"
  generate: "go generate ./..."
```

Run named tasks:

```bash
kiw run lint
kiw run audit
```

Tasks are executed using the native operating system shell (`sh -c` on Unix/macOS, `cmd /C` on Windows).

---

## 4. Environment Variables

Runtime configuration can be overridden using environment variables without modifying `krewire.yaml`:

| Variable | Values | Description |
| :--- | :--- | :--- |
| `KIW_ENV` | `local`, `production`, `testing` | Sets active environment profile. |
| `KIW_DEBUG` | `true`, `false`, `1`, `0` | Enables verbose debug logging and diagnostics. |
| `KIW_PORT` | Port number (e.g. `8080`) | Overrides the HTTP dev server listening port. |

Example:

```bash
KIW_ENV=production KIW_DEBUG=false kiw run
```

---

## Next Steps

With your configuration in place, proceed to [**2.3 Directory Structure →**](/quick-start/directory-structure) to explore the standard directory layouts for each workload.
