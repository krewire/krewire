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
  name: "my-service"            # Project identifier (read by the site loader)
  kind: "site"                  # Workload: app | cli | site | book | worker | service | infra | kernel
  version: "v0.1.0"             # Semantic version injected as `.Version` into pages
  dirs:                         # Optional: Override canonical folder locations
    web: "web"                  # Web handlers and templates
    public: "public"            # Raw static assets
    internal: "internal"        # Private domain packages
    cmd: "cmd"                  # Executable entry points

# Build Output & Content (top level — not nested under build:)
output: ".krewire/build"        # Target compilation output directory
base: "/"                       # URL base the site is served under
input: "content"                # Content directory consumed by the book pipeline

# Devtool Build Pipeline
build:
  include:                      # Glob patterns for content inclusion
    - "**/*.md"
  exclude:                      # Glob patterns for exclusion
    - "**/README.md"
    - "**/readme.md"

# Automatic CSS/JS Injection (site kind; on by default)
auto_assets:
  enabled: true                 # Set false to link every asset by hand in layouts/
  js_placement: "head"          # head (default, first-paint scripts) | body
  exclude:                      # Never auto-inject these assets
    - "*.min.css"
  order:                        # Pin where a specific asset loads
    "assets/theme.css":
      layer: "page"             # scoped | vendor | component | theme | book | page

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
  - `kernel`: Scaffold-only project before it is equipped (`kiw init`)
- **`dirs` (Optional):** Allows customization of the default directory layout if integrating Krewire into an existing repository layout.

---

### 3.2 Output, Base & Input (top level)

These are top-level keys — they are **not** nested under `build:`:

- **`output`:** The directory where compiled static files or assets are staged. Defaults to `.krewire/build`.
- **`base`:** The URL prefix under which assets and pages are served. Defaults to `/`. For sites hosted under subpaths (e.g. `https://example.com/blog/`), set `base: "/blog/"`.
- **`input`:** The content directory consumed by the book (`mdbind`) pipeline. Defaults to `content`.

---

### 3.3 Build Block (`build:`)

Controls which content files the pipeline processes:

- **`include` & `exclude`:** Glob patterns controlling which Markdown manuscripts or content files are processed. Unset `include` defaults to `**/*.md`; unset `exclude` defaults to skipping `README.md`/`readme.md`. An empty list disables that filter.

---

### 3.4 Dev Server Port

There is no `dev:` block in `krewire.yaml`. The local server port is set
per invocation with `--addr` (default `:8080`):

```bash
kiw dev --addr :3000
kiw serve --addr :3000
```

---

### 3.5 Auto Assets Block (`auto_assets:`)

Applies to the `site` kind. By default the build injects a `<link
rel="stylesheet">` into `<head>` and a `<script src>` into `<head>` for every
`.css`/`.js` file under `public/assets/`, plus the generated scoped stylesheet
`assets/style.css`. Layouts do not need to name them, and a tag you write by
hand is never duplicated. Cache busting uses `?v=<project version>`.

- **`enabled`:** `false` restores fully manual asset tags.
- **`js_placement`:** `head` (default) keeps theme scripts running before first
  paint; `body` appends them at the end of `<body>` instead.
- **`exclude`:** Asset names, base names, or globs that must never be injected.
- **`order`:** Pins one asset to a cascade layer, when the default position is
  wrong for your project.

Assets are **not** injected alphabetically — alphabetical order silently loads
book content before the utilities meant to override it. They load in cascade
order, so each layer can win over the previous one:

| Layer         | Holds                                            |
| ------------- | ------------------------------------------------ |
| `scoped`      | styles generated from `<style>` in components/layouts |
| `vendor`      | third-party output such as plugin CSS (Tailwind) |
| `component`   | component-library styling (Forge)                |
| `theme`       | theme variables and last-mile design overrides   |
| `book`        | content-pipeline styling (mdbind)                |
| `page`        | per-page overrides — always last                 |

```yaml
auto_assets:
  enabled: true
  js_placement: "body"
  exclude:
    - "vendor.js"
  order:
    "assets/theme.css":
      layer: "page"   # move the theme to last-mile
```

---

### 3.6 Scripts Task Runner (`scripts:`)

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

Example:

```bash
KIW_ENV=production KIW_DEBUG=false kiw run
```

---

## Next Steps

With your configuration in place, proceed to [**2.3 Directory Structure →**](/getting-started/directory-structure) to explore the standard directory layouts for each workload.
