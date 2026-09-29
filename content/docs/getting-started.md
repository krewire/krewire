---
title: "Getting Started"
description: "Install kiw and go from zero to a running site in under 5 minutes."
date: "2026-08-24"
---

# Getting Started

Krewire is Go-first. One CLI `kiw`, one config `krewire.yaml`, zero toolchain fatigue.

## Step 1 — Install kiw

Run the installer (Linux & macOS):

```bash
curl -fsSL https://krewire.com/scripts/install.sh | sh
```

This detects your OS and architecture, downloads the correct binary, and installs `kiw` to `/usr/local/bin`.

**Requirements:** curl or wget. sudo is optional (installs to `~/.local/bin` without it).

> **Go developers** can also install via `go install`:
> ```bash
> go install github.com/krewire/kiw/cmd/kiw@latest
> ```

Verify the installation:

```bash
kiw --version
```

---

## Step 2 — Scaffold a new project

```bash
kiw new my-site --site
cd my-site
```

This generates a complete, production-ready project structure:

```
my-site/
├── krewire.yaml        # project config
├── pages/
│   └── index.kiw       # → /
├── layouts/
│   └── Base.kiw        # wraps {{.Content}}
├── components/         # reusable .kiw components
└── public/             # static files (copied verbatim)
```

---

## Step 3 — Start the dev server

```bash
kiw dev
# ⚡ Ready at http://localhost:3000
```

Hot-reload on save — changes to `.kiw` files, layouts, and components reflect instantly.

---

## Step 4 — Build for production

```bash
kiw build
```

Output lands in `.krewire/build/` — a fully self-contained static site ready to deploy anywhere.

---

## Step 5 — Deploy

```bash
# GitHub Pages
kiw deploy --target gh-pages

# Or ship the Docker image
docker compose up -d --build
```

---

## File-based routing

```
pages/index.kiw          → /
pages/docs/index.kiw     → /docs
pages/docs/[slug].kiw    → /docs/:slug  (from content/docs/*.md)
components/Hero.kiw      → {{component "Hero" .}}
layouts/Base.kiw         → wraps {{.Content}}
public/logo.svg          → copied verbatim
```

No `go.mod` needed for `site` — `krewire.yaml` with `project.kind: site` is enough.

---

## Frontmatter is optional

Components like `Hero.kiw` need no `---` boilerplate:

```kiw
<section class="hero">
  <h1>{{.Title}}</h1>
</section>
<style>.hero{padding:40px}</style>
```

Pages with metadata keep it:

```kiw
---
title: Hello
layout: Base
---
<h1>{{.Title}}</h1>
```

---

## Devtool Configuration (`krewire.yaml`)

`krewire.yaml` is the dedicated configuration file for **`kiw` (the devtool)**. It configures devtool behavior and project metadata, keeping application runtime configuration decoupled:

```yaml
# krewire.yaml — Devtool & Build Configuration
project:
  name: my-site
  kind: site
  version: 0.1.0
  author: Alice

build:
  output: .krewire/build

dev:
  port: 8080
```

### Separation of Concerns

`krewire.yaml` is never used for runtime application state:

- **Devtool & Build (`krewire.yaml`):** Pinned workload kind (`project.kind`), build output (`build.output`), dev server port (`dev.port`), and project identity (`name`, `version`, `author`).
- **Application Pages & Content:** Page titles, descriptions, and slugs belong in page frontmatter (`pages/*.kiw`) or markdown metadata (`content/**/*.md`).
- **UI & Navigation:** Navbar links, footer content, and theme palettes belong in layouts (`layouts/*.kiw`) and CSS design tokens (`theme.css`).
- **Application Runtime:** Backend configs, database URLs, and environment variables belong in `.env` and compiled Go configuration (`internal/config`).

---

## Next

- [Workloads](/docs/workloads) — the 8 kinds
- [.kiw DSL](/docs/dsl) — HTML · CSS · JS/TS · Go · Rust in one file
- [Security & HTTP API](/docs/security-and-api)

