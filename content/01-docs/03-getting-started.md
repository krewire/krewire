---
title: "Getting Started"
description: "Install kiw and go from zero to a running site, app, or book in under 5 minutes."
date: "2026-09-29"
---

# Getting Started

Krewire is a **unified Go framework for every workload** — web monoliths, static sites, documentation books, CLI tools, background workers, microservices, cloud infrastructure, and WebAssembly runtimes.

Everything is driven by **one CLI (`kiw`)** and **one devtool configuration (`krewire.yaml`)** with zero JavaScript toolchain fatigue.

---

## Prerequisites

Before starting, ensure you have **Go 1.22+** installed on your system:

```bash
go version
```

If you don't have Go installed, download it from [golang.org/dl](https://golang.org/dl).

---

## Step 1 — Install the `kiw` CLI

`kiw` is the single binary developer tool used to scaffold, develop, build, test, and deploy Krewire projects.

### Option A: Shell Installer (Linux & macOS)

Run the automated installer script:

```bash
curl -fsSL https://krewire.com/scripts/install.sh | sh
```

The script detects your OS and architecture, downloads the latest binary, and installs `kiw` to `/usr/local/bin` (or `~/.local/bin` if running without sudo).

### Option B: Via `go install` (Cross-Platform / Windows)

If you already have Go configured with `$GOPATH/bin` in your `PATH`:

```bash
go install github.com/krewire/kiw/cmd/kiw@latest
```

### Verify the Installation

Check that `kiw` is installed and ready:

```bash
kiw version
```

You should see output similar to:

```text
kiw Krewire Devtool
  CLI          v0.1.0
  Framework    Krewire Framework v0.1.0 (dev)
  Libraries    github.com/krewire/libs v0.1.0 (dev)
  Go           go1.27.1 (linux/amd64)
```

---

## Step 2 — Create Your First Project (Static Site)

To create a new static site project equipped with modern file-based routing:

```bash
kiw new my-site --site
cd my-site
```

This creates a complete, production-ready directory structure:

```text
my-site/
├── krewire.yaml        # Devtool configuration (kind, build, dev)
├── pages/              # File-based routes (.kiw files)
│   └── index.kiw       # Landing page (serves /)
├── layouts/            # Reusable page shells
│   └── Base.kiw        # Base HTML layout wrapping {{.Content}}
├── components/         # Reusable template components
│   └── Hero.kiw        # Reusable component with scoped CSS
├── public/             # Static assets copied verbatim to build
│   └── favicon.svg     # Favicon and icons
└── README.md           # Project documentation
```

> [!NOTE]
> Unlike heavy frontend frameworks, a Krewire static site requires **no Node.js**, **no npm/yarn**, and **no node_modules**. Everything is compiled natively with Go.

---

## Step 3 — Start the Development Server

Start the local development server with file watching and auto-rebuild:

```bash
kiw dev
```

You will see the dev server start:

```text
INFO building site for development
INFO dev server running url=http://localhost:8080 dir=.krewire/build
INFO watching for changes root=. interval=500ms
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

### Live Editing & Auto-Rebuild

Edit `pages/index.kiw` or `components/Hero.kiw` in your editor and save. `kiw dev` automatically detects the modification, re-renders the site in milliseconds, and refreshes the build output:

```text
INFO change detected, rebuilding...
INFO site rebuilt successfully
```

To change the port or listen address:

```bash
kiw dev --addr :3000
```

Or configure it permanently in `krewire.yaml` under `dev.port: 3000`.

---

## Step 4 — Anatomy of a `.kiw` File

A `.kiw` file is a single-file component containing HTML structure, Go template expressions, scoped CSS, and optional client-side JavaScript.

### Page Route (`pages/index.kiw`)

Pages define route metadata in YAML frontmatter between `---` delimiters:

```html
---
title: Welcome to My Site
layout: Base
---

<div class="hero">
  <h1>Welcome to {{.Title}}</h1>
  <p>A fast, modern static site built with Go and Krewire.</p>
  {{component "Hero" .}}
</div>

<style>
  .hero {
    max-width: 720px;
    margin: 4rem auto;
    text-align: center;
  }
  .hero h1 {
    font-size: 2.75rem;
    color: #39D353;
  }
</style>
```

- **Frontmatter:** `layout: Base` specifies that this page is wrapped by `layouts/Base.kiw`.
- **Go Templates:** Use standard Go template variables like `{{.Title}}` and helpers like `{{component "Name" .}}`.
- **Scoped Styles:** CSS inside `<style>` is automatically scoped using `data-kiw-component` attributes to eliminate style collisions.

### Reusable Layout (`layouts/Base.kiw`)

Layouts provide the HTML skeleton and wrap page content via `{{.Content}}`:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="icon" href="/favicon.svg" type="image/svg+xml">
</head>
<body>
  <nav>
    <a href="/">Home</a>
    <a href="/about">About</a>
  </nav>

  <main>
    {{.Content}}
  </main>
</body>
</html>

<style>
  body {
    font-family: system-ui, -apple-system, sans-serif;
    margin: 0;
    padding: 2rem;
    background: #0d1117;
    color: #e6edf3;
  }
  a { color: #39D353; }
</style>
```

### Components (`components/Hero.kiw`)

Components are self-contained building blocks. Frontmatter is completely optional for components:

```html
<div class="card">
  <h3>Getting Started</h3>
  <p>Edit <code>pages/index.kiw</code> to customize this page.</p>
</div>

<style>
  .card {
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 1.5rem;
    background: #161b22;
  }
</style>
```

---

## Step 5 — Build for Production

When you are ready to produce a production build:

```bash
kiw build
```

This compiles your pages, components, layouts, scoped CSS, and static assets into the `.krewire/build/` directory:

```text
.krewire/build/
├── index.html
├── assets/
│   └── style.css       # Concatenated and scoped CSS
├── favicon.svg
└── .kiw-build-manifest
```

### Previewing the Production Build

You can preview the production output locally without file watching:

```bash
kiw serve
```

This serves `.krewire/build/` with extensionless URL resolution (e.g. `/docs` resolves to `docs.html`).

---

## Step 6 — Exploring Other Workload Variants

Krewire is designed for multiple workloads with a unified developer experience. You can scaffold different application variants with `kiw new`:

### 1. Fullstack Web Monolith (`--app`)

A single-binary web application with SSR, embedded assets, and typed JSON API handlers:

```bash
kiw new my-app --app
cd my-app
kiw run
```

### 2. Documentation Book (`--book`)

A documentation site assembled from Markdown chapters under `content/` powered by the integrated `mdbind` engine:

```bash
kiw new my-docs --book
cd my-docs
kiw dev
```

### 3. Interactive CLI Tool (`--cli`)

A command-line tool built on Krewire's terminal runtime (`framework/tui`):

```bash
kiw new my-cli --cli
cd my-cli
kiw run -- help
```

---

## Step 7 — Understanding `krewire.yaml`

`krewire.yaml` is the dedicated configuration file for **`kiw` (the devtool)**. It configures build pipelines, development settings, and project metadata:

```yaml
# krewire.yaml — Devtool & Build Configuration
project:
  name: my-site
  kind: site
  version: 0.1.0
  author: "Krewire Team"

build:
  output: .krewire/build
  base: /

dev:
  port: 8080
```

### Separation of Concerns

Krewire follows a clean separation between devtool settings and application data:

| Concern | Where It Belongs |
| :--- | :--- |
| **Devtool & Build** | `krewire.yaml` (`project.kind`, `build.output`, `dev.port`) |
| **Page Content & SEO** | Page frontmatter (`pages/*.kiw`) or Markdown frontmatter (`content/**/*.md`) |
| **Layout & Navigation** | Layout templates (`layouts/*.kiw`) and component trees |
| **Runtime Configuration** | Environment variables (`.env`) and compiled Go config (`internal/config`) |

---

## Next Steps

Now that you have your project running, dive deeper into the ecosystem:

- Proceed to [**1.4 Krewire Workloads →**](/docs/krewire-workloads) to explore the 8 workload kinds in detail.
- Read [**1.5 Upgrade Guide →**](/docs/upgrade-guide) for versioning policies and configuration migrations.
