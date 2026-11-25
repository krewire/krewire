# Workload: Static Website (`site`)

The `site` workload powers static websites, landing pages, documentation portals, and blogs using Krewire's native Static Site Generation (SSG) engine (`web/ssg`), `.kiw` component compiler (`ui/kiw`), and Markdown pipeline (`markdown`).

It produces zero-dependency static HTML/CSS/JS artifacts with automatic scoped styles, content fingerprinting, and WebSocket-driven Hot Module Replacement (HMR) during local development.

In production, this workload powers the official [`krewire.com`](https://krewire.com) website.

---

## 1. Architectural Capabilities

- **Pure-Go SSG Compiler**: Generates clean static HTML without requiring Node.js, npm, webpack, or Vite.
- **Scoped CSS Isolation**: `<style>` blocks inside `.kiw` components are automatically isolated via deterministic `data-kiw-*` scope attributes, preventing global CSS collisions.
- **Progressive Hydration Islands**: Static pages can mount interactive reactive islands (Go WASM, Svelte 5, Vue 3) using standard hydration contracts (`load`, `idle`, `visible`).
- **Live Preview & WebSocket HMR (`krew serve`)**: In-process development server watches files and pushes instantaneous updates to browser clients without full-page reloads.
- **Asset Pipeline & Cache Busting**: Scans `public/assets/`, bundles CSS/JS, minifies output, and injects fingerprint queries (`?v=0.2.3`).

---

## 2. Standard Directory Layout

Scaffolded via `krew new <site-name> --site` or `krew init --site`:

```text
my-site/
├── krewire.yaml          # Project configuration manifest (no go.mod required!)
├── pages/                # Route entrypoints (file-based routing)
│   ├── index.kiw         # Maps to /
│   └── about.kiw         # Maps to /about
├── layouts/              # Shared layout shells
│   └── Landing.kiw
├── components/           # Reusable clean, modern .kiw components
│   ├── Navbar.kiw
│   ├── Hero.kiw
│   └── Footer.kiw
└── public/               # Static assets directly copied to output
    ├── favicon.svg
    └── assets/
        └── styles.css
```

---

## 3. Configuration Contract (`krewire.yaml`)

```yaml
project:
  name: "my-site"
  version: "0.2.0"

output: ".krewire/build"
base: "/"

theme:
  default: "auto"         # auto | light | dark
  toggle: true

ssg:
  auto_assets:
    enabled: true         # Automatically inject styles and scripts into layouts
    bundle: true          # Bundle into assets/krewire.css and assets/krewire.js
    minify: true          # Minify HTML, CSS, and JS

server:
  port: 8080
  host: "127.0.0.1"
```

---

## 4. Authoring `.kiw` Components & Pages

### Example Layout (`layouts/Landing.kiw`)

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}} — Krewire</title>
</head>
<body>
  <Navbar />
  <main>
    {{slot}}
  </main>
  <Footer />
</body>
</html>
```

### Example Page (`pages/index.kiw`)

```html
---
layout: "Landing"
title: "Home"
---

<section class="hero">
  <h1>Build Fast with Pure Go</h1>
  <p>Pure Go efficiency, 10-30 MB memory footprints, and instant cold starts.</p>
  <a href="/docs" class="btn btn-primary">Read Documentation</a>
</section>

<style>
.hero {
  text-align: center;
  padding: 4rem 1rem;
}
.hero h1 {
  font-size: 2.5rem;
  font-weight: 800;
  margin-bottom: 1rem;
}
</style>
```

---

## 5. Development & Deployment Workflow

```bash
# 1. Start local preview server with instant WebSocket HMR
krew serve

# 2. Compile static production output into .krewire/build/
krew build --minify

# 3. Deploy static site to GitHub Pages or CDN
krew deploy --target gh-pages
```

---

## 6. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Produces static HTML/CSS with no server-side attack surface; contextual template auto-escaping prevents XSS.
- **Sustainable (S2)**: Pure Go runtime; works with or without `go.mod`; builds hundreds of pages in sub-second time on a minimal $5 VPS.
- **Scalable (S3)**: Seamlessly enhances from pure static pages into fullstack applications (`app`) or multi-language books (`book`) using the same `.kiw` templates.
