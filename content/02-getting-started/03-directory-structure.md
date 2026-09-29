---
title: "Directory Structure"
description: "Anatomy and conventions of Krewire project directory layouts across all workload variants."
date: "2026-09-29"
---

# Directory Structure

Krewire establishes predictable, standardized project structures designed to scale gracefully from single-file prototypes into enterprise modular monoliths.

---

## 1. Standard Project Anatomy

Below is the standard, unified file layout of a comprehensive Krewire project:

```text
my-project/
├── krewire.yaml          # Devtool & build pipeline configuration
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
│
├── pages/                # File-based routes (.kiw component files)
│   ├── index.kiw         # Serves "/"
│   ├── about.kiw         # Serves "/about"
│   └── 404.html          # Custom error page
│
├── layouts/              # Shared page wrappers (.kiw)
│   ├── Base.kiw          # Default HTML document layout
│   └── Landing.kiw       # Full-bleed landing page layout
│
├── components/           # Reusable scoped UI components (.kiw)
│   ├── Header.kiw        # Navigation bar
│   ├── Footer.kiw        # Footer section
│   └── CodeBlock.kiw     # Syntax highlighter component
│
├── content/              # Documentation manuscripts & books (mdbind)
│   ├── 01-overview/      # Chapter 1 directory
│   │   ├── index.md      # Chapter landing page
│   │   └── 01-arch.md    # Subchapter 1.1
│   └── 02-guides/        # Chapter 2 directory
│
├── public/               # Raw static assets served directly
│   ├── favicon.svg       # Browser favicon
│   ├── robots.txt        # Web crawler configuration
│   ├── sitemap.xml       # Search engine index
│   └── assets/           # Client scripts, stylesheets, and images
│
├── cmd/                  # Executable Go entry points
│   └── my-service/
│       └── main.go       # Service entry point
│
├── internal/             # Private application logic (enforced by Go compiler)
│   ├── domain/           # Core business entities & models
│   ├── handler/          # HTTP & RPC request handlers
│   └── service/          # Domain services & business rules
│
└── .krewire/             # Toolchain state & build artifacts (git-ignored)
    ├── build/            # Default compilation output
    └── .kiw-build-manifest
```

---

## 2. Directory Roles & Responsibilities

### `pages/` (File-Based Routing)
Files in `pages/` map directly to public URL routes:
- `pages/index.kiw` → `https://example.com/`
- `pages/pricing.kiw` → `https://example.com/pricing`
- `pages/blog/first-post.kiw` → `https://example.com/blog/first-post`

Each `.kiw` page file encapsulates its YAML frontmatter, HTML template markup, and scoped `<style>` block.

---

### `layouts/` (Document Shells)
Layouts provide the outer HTML shell (`<!doctype html>`, `<head>`, `<nav>`, `<footer>`) wrapping page content via `{{.Content}}`. Pages declare their desired layout in frontmatter:

```yaml
---
title: "Dashboard"
layout: AppLayout
---
```

---

### `components/` (Reusable Scoped Blocks)
Components are standalone `.kiw` files imported into pages or layouts via the `{{component "Name" .}}` template helper. Styles inside components are scoped automatically at compile time with zero CSS collisions.

---

### `content/` (Manuscript Engine)
Stores Markdown documentation manuscripts compiled by `mdbind`. Numeric prefixes (`01-`, `02-`) dictate chapter and subchapter reading order in the generated navigation sidebar.

---

### `public/` (Uncompiled Static Assets)
Files inside `public/` are served verbatim without preprocessing:
- `public/favicon.svg` is served at `/favicon.svg`.
- `public/robots.txt` is served at `/robots.txt`.
- `public/assets/` contains client scripts and CSS stylesheets.

---

### `cmd/` & `internal/` (Go Application Core)
- **`cmd/<name>/main.go`:** Main entry points for fullstack monoliths (`app`), CLI utilities (`cli`), or background workers (`worker`).
- **`internal/`:** Go compiler-enforced private packages. Code inside `internal/` cannot be imported by external modules, preserving clean architectural encapsulation (`KWF-5ZHQV`).

---

### `.krewire/` (Build Artifacts)
Created automatically by `kiw dev` and `kiw build`:
- **`.krewire/build/`:** Houses the compiled production output ready for static hosting or Docker image packaging.
- This directory is managed by `kiw` and should always be added to your `.gitignore`.

---

## 3. Workload-Specific Layouts

When scaffolding a new project via `kiw new <project> [flags]`, `kiw` equips the directory structure tailored to that workload:

### 1. Static Site (`kiw new my-site --site`)
```text
my-site/
├── krewire.yaml
├── go.mod
├── pages/
│   └── index.kiw
├── layouts/
│   └── Base.kiw
├── components/
└── public/
```

### 2. Documentation Book (`kiw new my-docs --book`)
```text
my-docs/
├── krewire.yaml
├── go.mod
└── content/
    ├── index.md
    └── 01-getting-started.md
```

### 3. Fullstack Monolith (`kiw new my-app --app`)
```text
my-app/
├── krewire.yaml
├── go.mod
├── cmd/
│   └── my-app/
│       └── main.go
├── web/
│   ├── router.go
│   └── handlers/
├── internal/
│   └── domain/
└── public/
```

### 4. Terminal CLI (`kiw new my-cli --cli`)
```text
my-cli/
├── krewire.yaml
├── go.mod
├── cmd/
│   └── my-cli/
│       └── main.go
└── internal/
    └── commands/
```

---

## 4. Customizing Directory Locations

If you are incorporating Krewire into an established repository with non-standard folder conventions, override them in `krewire.yaml`:

```yaml
project:
  name: "custom-app"
  kind: "app"
  dirs:
    web: "src/web"
    public: "static"
    internal: "pkg"
    cmd: "entrypoints"
```

The `kiw` compiler resolves all paths against these mappings automatically.

---

## Summary

With the installation, configuration, and directory layout understood, you are equipped to build robust applications across any Krewire workload.

Proceed to explore the [**1.3 Krewire Workloads Matrix →**](/docs/krewire-workloads) or return to [**1. Overview →**](/docs).
